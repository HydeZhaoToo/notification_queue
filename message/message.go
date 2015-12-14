//实现:1.发送短信,2.init读取用户配置3,检测json参数的正确性
package message

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/Unknwon/goconfig"
	"github.com/astaxie/beego/httplib"
	"github.com/hprose/hprose-go/hprose"
)

type CallbackObject struct {
	Table string `json:table`
	Id    int    `json:id`
	Value string `json:value`
}

type Message_info struct {
	Id            int    `json:id`
	Recipient     string `json:recipient` //
	Content       string `json:content`
	Sms_id        string `json:sms_id`
	Sendtime      int64  `json:sendtime`
	Plan_sendtime int64  `json:plan_sendtime`
	From          string `json:from`
}

type clientsub struct {
	Notify_callback func(map[string]string) `name:"notify_callback"`
}

var (
	g         *goconfig.ConfigFile //定义配置
	hp_Client hprose.Client        //定义hprose客户端
	hp_tmp    *clientsub           //HProse 调用结构体
	loger     *log.Logger
)

func init() {
	var err error
	file, err := os.OpenFile("message.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	loger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	g, err = goconfig.LoadConfigFile("conf/message.ini")
	chkerr(err)
	hp_Client = hprose.NewHttpClient("http://center.zsj.test/Open")
	hp_Client.UseService(&hp_tmp)
}

func (accinfo *Message_info) Send_msg(this map[string]string, sig <-chan int, thisc CallbackObject) error {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case run := <-ticker.C:
			if run.Unix() > accinfo.Plan_sendtime {
				oburl := fmt.Sprintf("%s?accountname=%s&accountpwd=%s&mobilecodes=%s&msgcontent=%s", this["url"], this["accountname"], this["accountpwd"], accinfo.Recipient, url.QueryEscape(accinfo.Content))
				//fmt.Println("url 生成信息")
				//	fmt.Println(oburl)
				resben := httplib.Get(oburl)
				if rs, err := resben.String(); err == nil && rs == "0" {
					hp_tmp.Notify_callback(map[string]string{
						"type":      "1",
						"client_id": "5",
						"token":     "jiazhuangbao005",
						"callback":  thisc.Jsonstring(),
						"code":      "200",
						"msg":       "",
						"send_time": strconv.FormatInt(run.Unix(), 10),
					})
					loger.SetPrefix("[info]")
					loger.Printf("任务ID:%v  用户:%v发送成功!", accinfo.Id, accinfo.Recipient)
					return errors.New("1")
				} else {
					hp_tmp.Notify_callback(map[string]string{
						"type":      "1",
						"client_id": "5",
						"token":     "jiazhuangbao005",
						"callback":  thisc.Jsonstring(),
						"code":      "300",
						"msg":       fmt.Sprintf("失败%v httpcode%v", err, rs),
						"send_time": strconv.FormatInt(run.Unix(), 10),
					})
					loger.SetPrefix("[info]")
					loger.Printf("任务ID:%v  用户:%v发送失败!", accinfo.Id, accinfo.Recipient)
					return errors.New("1")
				}
			}
		case <-sig:
			return nil
		}
	}
}

func Create_msg_ini() (map[string]map[string]string, error) {
	defer func() {
		if err := recover(); err != nil {
			loger.SetPrefix("warning")
			loger.Println(err)
		}
	}()
	err := g.Reload()
	chkerr(err)
	msg_tmp := make(map[string]map[string]string)
	for _, v := range g.GetSectionList() {
		msg_tmp[v], err = g.GetSection(v)
		chkerr(err)
	}

	//fmt.Printf("我建立的配置:%v\n", msg_tmp)
	return msg_tmp, nil

}

func Save_msg_ini(this map[string]map[string]string) {
	defer func() {
		if err := recover(); err != nil {
			loger.SetPrefix("warning")
			loger.Println(err)
		}
	}()
	filehand, err := os.OpenFile("conf/message.ini", os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0600)
	chkerr(err)
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	for k, v := range this {
		fmt.Fprintf(filehand, "[%s]\n", k)
		for u, i := range v {
			fmt.Fprintf(filehand, "%s=%s\n", u, i)
		}
	}
	filehand.Close()
}
func (c CallbackObject) Jsonstring() string {
	return fmt.Sprintf(`{"table": "%v","id": %v,"value": %v}`, c.Table, c.Id, c.Value)
}

func chkerr(err error) {
	if err != nil {
		panic(err)
	}
}
