package requrl

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"strconv"
	"strings"
	"time"

	"github.com/hprose/hprose-go/hprose"
)

type CallbackObject struct {
	Table string `json:table`
	Id    int    `json:id`
	Value string `json:value`
}

type Requrl struct {
	Interval    int    `json:interval`    //间隔时间
	Callbackurl string `json:callbackurl` //请求URL
	Sucmsg      string `json:sucmsg`      //对比信息
	Frequency   int    `json:frequency`   //请求次数
}

var (
	ss     hprose.Client //定义hprose客户端
	hp_tmp *clientsub    //HProse 调用结构体
	loger  *log.Logger
)

type clientsub struct {
	Notify_callback func(map[string]string) `name:"notify_callback"`
}

func init() {
	file, err := os.OpenFile("requrl.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	loger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	ss = hprose.NewHttpClient("http://center.zsj.test/Open")
	ss.UseService(&hp_tmp)
}

func (r Requrl) Geturl(thisC CallbackObject) {
	defer func() {
		if err := recover(); err != nil {
			loger.SetPrefix("warning")
			loger.Println(err)
		}
	}()
	var now time.Time
	//fmt.Printf("%#v\n", r)
	if r.Callbackurl == "" {
		return
	}
	ticker := time.NewTicker(time.Second * time.Duration(r.Interval))
	fmt.Println(r)
	for i := 0; i < r.Frequency; i++ {
		http.Header.Add("User-Agent", "notify")
		rs, err := http.Get(r.Callbackurl)
		if err != nil {
			loger.Println(err)
		}
		defer rs.Body.Close()
		data, err := ioutil.ReadAll(rs.Body)
		now = <-ticker.C
		//sttmp, err := rs.String(); strings.Contains(sttmp, r.Sucmsg)
		if strings.Contains(string(data), r.Sucmsg) && err == nil {
			hp_tmp.Notify_callback(map[string]string{
				"type":      "3",
				"client_id": "5",
				"token":     "jiazhuangbao005",
				"callback":  thisC.Jsonstring(),
				"code":      "200",
				"msg":       "",
				"send_time": strconv.FormatInt(now.Unix(), 10),
			})
			loger.SetPrefix("[info_requrl]")
			loger.Printf("请求地址:%v 成功!", r.Callbackurl)
			return
		} else {
			if i == r.Frequency-1 {
				hp_tmp.Notify_callback(map[string]string{
					"type":      "3",
					"client_id": "5",
					"token":     "jiazhuangbao005",
					"callback":  thisC.Jsonstring(),
					"code":      "300",
					"msg":       "未完成",
					"send_time": strconv.FormatInt(now.Unix(), 10),
				})
				loger.SetPrefix("[info_requrl]")
				loger.Printf("请求地址:%v 失败!", r.Callbackurl)
				return
			}
		}
	}

}
func (c CallbackObject) Jsonstring() string {
	return fmt.Sprintf(`{"table": "%v","id": %v,"value": %v}`, c.Table, c.Id, c.Value)
}
