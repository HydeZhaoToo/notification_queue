package main

import (
	"runtime"
	//用于hprose部分
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hprose/hprose-go/hprose"

	//用于ppid变1 以及reload 配置
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	//功能实现部分
	"github.com/HydeZhaoToo/notification_queue/message"
	"github.com/HydeZhaoToo/notification_queue/requrl"
	"github.com/HydeZhaoToo/notification_queue/sendmail"
)

var (
	//短信数据结构
	meginfoS       map[string]map[string]string //短信账户信息表
	mailinfoS      map[string]map[string]string //邮件账户信息表
	messageSigList map[int]chan int             //短信goroutine通道维护表
	mailSigList    map[int]chan int             //邮件goroutine通道维护表

)

//HProse  处理函数
func Notice_service(tp int, jsonstr string, callbackstr string) map[string]string {
	var mess message.Message_info           //初始化短信结构
	var messcallback message.CallbackObject //初始化短信callback结构
	var mai sendmail.Mail_info              //初始化邮件结构
	var maicallback sendmail.CallbackObject //初始化邮件callback结构
	var reurl requrl.Requrl
	var reurlcallback requrl.CallbackObject
	switch tp { //判断类型 1短信 2邮件 3url
	case 1:
		//解析json
		err := json.Unmarshal([]byte(jsonstr), &mess)
		err1 := json.Unmarshal([]byte(callbackstr), &messcallback)
		//	fmt.Printf("解析结果%#v\n", mess)
		if err != nil && err1 != nil {
			return map[string]string{
				"code":  "300",
				"error": fmt.Sprintf("json解析失败:%v callback:%v", err, err1),
				"date":  "",
			}
		} else {
			messageSigList[mess.Id] = make(chan int) //初始化通道
			go func() {
				CB_tmp := mess.Send_msg(meginfoS[mess.Sms_id], messageSigList[mess.Id], messcallback)
				if CB_tmp != nil {
					delete(messageSigList, mess.Id)
				}
			}() //goroutine执行
			return map[string]string{
				"code":  "200",
				"error": "",
				"date":  "",
			}
		}
	case 2:
		err := json.Unmarshal([]byte(jsonstr), &mai)
		err1 := json.Unmarshal([]byte(callbackstr), &maicallback)

		if err != nil || err1 != nil {
			return map[string]string{
				"code":  "300",
				"error": fmt.Sprintf("json解析失败:%v callback:%v", err, err1),
				"date":  "",
			}
		} else {
			mailSigList[mai.Id] = make(chan int)
			go func() {
				CB_tmp := mai.Send_mail(mailinfoS[mai.Smtp_id], mailSigList[mai.Id], maicallback)
				if CB_tmp != nil {
					delete(mailSigList, mai.Id)
				}
			}()
			return map[string]string{
				"code":  "200",
				"error": "",
				"date":  "",
			}
		}
	case 3:
		err := json.Unmarshal([]byte(jsonstr), &reurl)
		err1 := json.Unmarshal([]byte(callbackstr), &reurlcallback)
		if err != nil || err1 != nil {
			return map[string]string{
				"code":  "300",
				"error": fmt.Sprintf("json解析失败:%v callback:%v", err, err1),
				"date":  "",
			}
		} else {
			go reurl.Geturl(reurlcallback)
			return map[string]string{
				"code":  "200",
				"error": "",
				"date":  "",
			}
		}

	default:
		return map[string]string{
			"code":  "404",
			"error": "你传的啥破类型!",
			"date":  "",
		}
	}
}

//撤回 任务
func Remove(tp int, task_id int) map[string]string {
	switch tp {
	case 1:
		if v, ok := messageSigList[task_id]; ok {
			v <- task_id                    //终止goroutine
			delete(messageSigList, task_id) //删除已终止任务
			return map[string]string{
				"code":  "200",
				"error": "",
				"date":  "",
			}
		} else {
			//return fmt.Sprintf("撤销失败,没找到%v", task_id)
			return map[string]string{
				"code":  "300",
				"error": fmt.Sprintf("撤销失败,没找到%v", task_id),
				"date":  "",
			}
		}
	case 2:
		if v, ok := mailSigList[task_id]; ok {
			v <- task_id
			delete(mailSigList, task_id)
			return map[string]string{
				"code":  "200",
				"error": "",
				"date":  "",
			}
		} else {
			return map[string]string{
				"code":  "300",
				"error": fmt.Sprintf("撤销失败,没找到%v", task_id),
				"date":  "",
			}
		}
	case 3:
		return map[string]string{
			"code":  "404",
			"error": "",
			"date":  "",
		}

	default:
		return map[string]string{
			"code":  "404",
			"error": "传的啥破类型!",
			"date":  "",
		}
	}
}

//修改配置
func Change_config(tp int, jsonini string) map[string]string {
	switch tp {
	case 1:
		//更改短信配置
		meginfoS = make(map[string]map[string]string)
		json.Unmarshal([]byte(jsonini), &meginfoS)
		message.Save_msg_ini(meginfoS)
		return map[string]string{
			"code":  "200",
			"error": "",
			"date":  "",
		}
	case 2:
		//更改邮件配置
		mailinfoS = make(map[string]map[string]string)
		json.Unmarshal([]byte(jsonini), &mailinfoS)
		sendmail.Save_mail_ini(mailinfoS)
		return map[string]string{
			"code":  "200",
			"error": "",
			"date":  "",
		}
	case 3:
		return map[string]string{
			"code":  "404",
			"error": "",
			"date":  "",
		}
	default:
		return map[string]string{
			"code":  "404",
			"error": "",
			"date":  "",
		}
	}

}

//配置重读
func init() {

	var err error
	//初始化短信配置以及 goroutine表
	meginfoS, err = message.Create_msg_ini()
	chkerr(err)
	messageSigList = make(map[int]chan int)
	//初始化邮件配置以及 goroutine表
	mailinfoS, err = sendmail.Create_mail_ini()
	chkerr(err)
	mailSigList = make(map[int]chan int)
}

func Status(rw http.ResponseWriter, req *http.Request) {
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	fmt.Fprintf(rw, "maseeage模块: %v个在执行\nmail模块: %v个在执行\n程序使用内存%v", len(messageSigList), len(mailSigList), memStat.Alloc)
}

func main() {
	if os.Getppid() != 1 {
		ss := exec.Command(os.Args[0], os.Args[1:]...)
		ss.Stdout = os.Stdout
		ss.Start()
		fmt.Println("[PID]", ss.Process.Pid)
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	sig := make(chan os.Signal, 1) //建立系统signal通道
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGUSR1)
	//syscall.SIGUSR2) //监听信号
	//,syscall.SIGTERM,syscall.SIGQUIT,syscall.SIGINT,syscall.SIGSTOP)
	go func() {
		for {
			s := <-sig
			switch s {
			case syscall.SIGHUP:
				fmt.Println("重读配置...")
				if os.Getppid() != 1 {
					fmt.Println("reload失败 不是后台进程")
				} else {
					//重读短信配置
					tmp_msg, err := message.Create_msg_ini()
					chkerr(err)
					meginfoS = tmp_msg

					//重读邮件配置
					tmp_mail, err := sendmail.Create_mail_ini()
					chkerr(err)
					mailinfoS = tmp_mail
				}
			case syscall.SIGUSR1:
				fmt.Println("\n将平滑关闭服务...")

				if cou := len(messageSigList); cou > 0 {
					fmt.Printf("maseeage模块: %v个在执行\n\n", cou)
					//关闭所有还没执行的goroutine
					for k, v := range messageSigList {
						v <- k
						delete(messageSigList, k)
					}
					fmt.Println("message模块所有goroutine已经关闭")
				} else {
					fmt.Println("message没有任务")
				}

				if cou := len(mailSigList); cou > 0 {
					fmt.Printf("mail模块: %v个在执行\n", cou)
					for k, v := range mailSigList {
						v <- k
						delete(mailSigList, k)
					}
					fmt.Println("mail模块所有进程已经关闭\n")
				} else {
					fmt.Println("mail没有任务\n\n")
				}
				/*
					case syscall.SIGUSR2:
						fmt.Printf("\nmaseeage模块: %v个在执行\n", len(messageSigList))
						fmt.Printf("mail模块: %v个在执行\n", len(mailSigList))*/
			default:
				fmt.Println("\n未监听信号")
			}
		}
	}()

	go func() {
		http.HandleFunc("/status", Status)
		http.ListenAndServe(":9977", nil)
	}()

	//HProse启动
	Service := hprose.NewHttpService()
	Service.AddFunction("Service", Notice_service)
	Service.AddFunction("Remove", Remove)
	Service.AddFunction("Changeini", Change_config)
	if err := http.ListenAndServe(":9966", Service); err != nil {
		fmt.Println(err)
	}

}

func chkerr(err error) {
	if err != nil {
		panic(err)

	}
}
