package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/olivere/elastic.v5"
	"mictract/config"
	"mictract/global"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"


	"github.com/jordan-wright/email"
)

type LAlert struct {
	Client  		*elastic.Client
	Interval      	int
	Size 			int
	res   			[]*regexp.Regexp

	SMTPHost		string
	SMTPPort		int
	SMTPUsername	string
	SMTPPassword	string
	SMTPRecvier		[]string

}

func NewLAlert(interval, size int, eshosts []string, smtpHost string, smtpPort int, smtpUsername, smtpPassword string, smtpRecvier []string) (*LAlert, error) {
	la := &LAlert{
		Interval: 		interval,
		Size: 			size,
		res: 			[]*regexp.Regexp{},

		SMTPHost: 		smtpHost,
		SMTPPort: 		smtpPort,
		SMTPUsername: 	smtpUsername,
		SMTPPassword: 	smtpPassword,
		SMTPRecvier: 	smtpRecvier,
	}
	client, err := elastic.NewClient(elastic.SetURL(eshosts...))
	if err != nil {
		return la, err
	}
	la.Client = client

	for _, s := range []string{
		`^([^\"]+) \[([A-Z]+)\] ([\s\S]*)$`, // ca
		`^([\s\S]*) -> ([A-Z]+) ([\s\S]+)$`, // orderer and peer
	} {
		la.res = append(la.res, regexp.MustCompile(s))
	}

	return la, nil
}

func (la *LAlert)getIndex() string {
	return fmt.Sprintf("logstash-%s",time.Now().Format("2006.01.02"))
}

func (la *LAlert)GetLogs(from int) ([]string, error) {
	ret := []string{}
	boolQ := elastic.NewBoolQuery()
	boolQ.Must(elastic.NewMatchQuery("kubernetes.labels.app", "mictract"))
	//boolQ.Filter(elastic.NewRangeQuery("age").Gt(30))
	res, err := la.Client.Search(la.getIndex()).
		Type("fluentd").
		Size(la.Size).
		From(from).
		Query(boolQ).
		Do(context.Background())
	if err != nil {
		return []string{}, err
	}
	for _, hit := range res.Hits.Hits {
		var mylog struct {
			Log  string `json:"log"`
		}
		err := json.Unmarshal(*hit.Source, &mylog)
		if err != nil {
			return ret, err
		} else {
			ret = append(ret, mylog.Log)
		}
	}
	return ret, nil
}

func (la *LAlert)GetTotalHits() (int64, error) {
	boolQ := elastic.NewBoolQuery()
	boolQ.Must(elastic.NewMatchQuery("kubernetes.labels.app", "mictract"))
	res, err := la.Client.Search(la.getIndex()).
		Type("fluentd").
		Size(1).
		From(0).
		Query(boolQ).
		Do(context.Background())
	if err != nil {
		return -1, err
	}
	return res.Hits.TotalHits, nil
}

func (la *LAlert)Rule(log string) bool {
	for _, re := range la.res {
		m := re.FindStringSubmatch(log)
		if len(m) >=2  && (m[2] == "ERRO" || m[2] == "ERROR" || m[2] == "WARN"){
			return true
		} else {

		}
	}
	return false
}

func (la *LAlert)Alert(message string) {
	e := email.NewEmail()
	e.From = la.SMTPUsername
	e.To = la.SMTPRecvier
	e.Subject = "alert"
	e.Text = []byte(message)
	if err := e.Send(
		fmt.Sprintf("%s:%d", la.SMTPHost, la.SMTPPort),
		smtp.PlainAuth("", la.SMTPUsername, la.SMTPPassword, la.SMTPHost)); err != nil {
		global.Logger.Error("fail to send email", zap.Error(err))
	}
}


func RunAlert(la *LAlert) {
	from := 0
	tot, err := la.GetTotalHits()
	if err != nil {
		global.Logger.Error("fail to query es", zap.Error(err))
		return
	}
	message := ""
	for {
		if int64(from) >= tot {
			break
		}
		logs, err := la.GetLogs(from)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
		} else {
			for _, log := range logs {
				if la.Rule(log) {
					message += log + "\n"
				}
			}
		}
		from = from + la.Size
	}
	if message != "" {
		la.Alert(message)
	}
}

func StartMyAlert() {
	if config.ALERT_ENABLE == "true" {
		go func() {
			port, _ := strconv.Atoi(config.SMTPPort)
			interval := 10 * 60
			var la *LAlert
			var err error
			for {
				if la, err = NewLAlert(
					interval,
					1000,
					config.ES_HOSTS,
					config.SMTPHost,
					port,
					config.SMTPUsername,
					config.SMTPPassword,
					strings.Split(config.SMTPRecvier, ";")); err != nil {
					global.Logger.Error("fail to get alert object", zap.Error(err))
					global.Logger.Info("retrying...")
					time.Sleep(60 * time.Second)
				} else {
					break
				}
			}
			for {
				go RunAlert(la)
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}()
	}
}



