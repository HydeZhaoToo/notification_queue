//在程序上线前 需要替换每个模块的 hprose的回调接口地址
package sendmail

import (
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"github.com/Unknwon/goconfig"
	"github.com/hprose/hprose-go/hprose"
)

type CallbackObject struct {
	Table string `json:table`
	Id    int    `json:id`
	Value string `json:value`
}

type Mail_info struct {
	Id            int    `json:id`            //邮件任务ID
	Recipient     string `json:recipient`     //收件人
	Title         string `json:title`         //邮件标题
	Content       string `json:content`       //邮件内容
	Sendtime      int64  `json:sendtime`      //实际发送时间
	Smtp_id       string `json:smtp_id`       // 使用账户
	Plan_sendtime int64  `json:plan_sendtime` //计划发送时间
	From          string `json:from`          //邮件别名
}

type clientsub struct {
	Notify_callback func(map[string]string) `name:"notify_callback"`
}

var (
	g      *goconfig.ConfigFile //定义配置
	ss     hprose.Client        //定义hprose客户端
	hp_tmp *clientsub           //HProse 调用结构体
	loger  *log.Logger
)

func init() {
	//初始化
	var err error
	file, err := os.OpenFile("sendmail.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	loger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	g, err = goconfig.LoadConfigFile("conf/mail.ini")
	if err != nil {
		log.Fatalln(err)
	}
	ss = hprose.NewHttpClient("http://center.zsj.test/Open")
	ss.UseService(&hp_tmp)
}

func (mailinfo *Mail_info) Send_mail(this map[string]string, sig <-chan int, thisc CallbackObject) error {
	ticker := time.NewTicker(time.Second * 3)
	for {
		select {
		case run := <-ticker.C:
			if run.Unix() > mailinfo.Plan_sendtime {
				sub := fmt.Sprintf("To: %s\r\nFrom: %s<%s>\r\nSubject: %s\r\nContent-Type: text/html; Charset=UTF-8\r\n\r\n%s", mailinfo.Recipient, mailinfo.From, this["smtp_user"], mailinfo.Title, mailinfo.Content)
				auth := smtp.PlainAuth("", this["smtp_user"], this["smtp_pwd"], this["smtp_server"])
				au := fmt.Sprintf("%s:%s", this["smtp_server"], this["smtp_port"])
				err := smtp.SendMail(au, auth, this["smtp_user"], []string{mailinfo.Recipient}, []byte(sub))
				if err == nil {
					hp_tmp.Notify_callback(map[string]string{
						"type":      "2",
						"client_id": "5",
						"token":     "jiazhuangbao005",
						"callback":  thisc.Jsonstring(),
						"code":      "200",
						"msg":       "",
						"send_time": strconv.FormatInt(run.Unix(), 10),
					})
					loger.SetPrefix("[info]")
					loger.Printf("任务ID:%v  用户:%v发送成功!", mailinfo.Id, mailinfo.Recipient)
					return errors.New("1")
				} else {
					hp_tmp.Notify_callback(map[string]string{
						"type":      "2",
						"client_id": "5",
						"token":     "jiazhuangbao005",
						"callback":  thisc.Jsonstring(),
						"code":      "300",
						"msg":       fmt.Sprintf("失败%v httpcode%v", err),
						"send_time": strconv.FormatInt(run.Unix(), 10),
					})
					return errors.New("1")
				}
			}
		case <-sig:
			return nil
		}
	}
}

func Create_mail_ini() (map[string]map[string]string, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	err := g.Reload()
	chkerr(err)
	msg_tmp := make(map[string]map[string]string)
	for _, v := range g.GetSectionList() {
		msg_tmp[v], err = g.GetSection(v)
		chkerr(err)
	}
	return msg_tmp, nil
}

func Save_mail_ini(this map[string]map[string]string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	filehand, err := os.OpenFile("conf/mail.ini", os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0600)
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
