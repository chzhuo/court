/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

var feishu string

func sendMsg(msg string) error {
	// json
	contentType := "application/json"
	// data
	data := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": fmt.Sprintf("文体中心羽毛球夜晚场地更新：\n%s", msg),
		},
	}

	bs, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return err
	}
	logrus.Infof("send msg: %s", string(bs))
	// request
	result, err := http.Post(feishu, contentType, bytes.NewReader(bs))
	if err != nil {
		fmt.Printf("post failed, err:%v\n", err)
		return err
	}
	defer result.Body.Close()

	if result.StatusCode/100 != 2 {
		fmt.Printf("post failed, status:%v\n", result.StatusCode)
		return fmt.Errorf("post failed, status:%v", result.StatusCode)
	}
	return nil
}

var preMsg = "fake msg"

func sendAlert(alertInfo []string) {
	msg := strings.Join(alertInfo, "\n")
	if preMsg == msg {
		return
	}
	err := sendMsg(msg)
	if err != nil {
		logrus.Errorf("send msg: %v", err)
	}
	preMsg = msg
}

func check() {
	today, err := time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), time.Local)
	if err != nil {
		logrus.Errorf("parse day: %v", err)
		return
	}
	alertInfo := make([]string, 0)
	for i := 0; i < 5; i++ {
		t := today.Add(time.Hour * 24 * time.Duration(i))
		url := fmt.Sprintf("https://xihuwenti.juyancn.cn/wechat/product/details?id=753&time=%d", t.Unix())
		fmt.Println(url)
		res, err := http.Get(url)
		if err != nil {
			logrus.Errorf("get: %v", err)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			logrus.Errorf("status code error: %d %s", res.StatusCode, res.Status)
			return
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			logrus.Errorf("new document: %v", err)
			return
		}

		thresholdT, err := time.Parse("15:04", "18:00")
		if err != nil {
			logrus.Errorf("parse start time: %v", err)
			return
		}

		// Find the review items
		doc.Find(".can-select").Each(func(i int, s *goquery.Selection) {
			start, _ := s.Attr("data-start")
			end, _ := s.Attr("data-end")
			hall, _ := s.Attr("data-hall_name")

			startT, err := time.Parse("15:04", start)
			if err != nil {
				logrus.Errorf("parse start time: %v", err)
				return
			}
			if startT.Sub(thresholdT) >= 0 {
				info := fmt.Sprintf("星期%d, %s-%s %s", int(t.Weekday()), start, end, hall)
				alertInfo = append(alertInfo, info)
			}
		})
		logrus.Infof("check %s", t.Format("2006-01-02 15:04:05"))
	}
	sendAlert(alertInfo)
}

func main() {
	interval := 60
	flag.IntVar(&interval, "interval", interval, "check interval")
	flag.StringVar(&feishu, "feishu", "", "feishu webhook url")
	flag.Parse()

	for {
		check()
		time.Sleep(time.Second * time.Duration(interval))
	}
}
