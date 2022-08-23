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

var pageId = 753
var interval = 5
var startTime time.Time
var endTime time.Time

func check() {
	alertInfo := make([]string, 0)

	dates := make([]string, 0)
	currentDateIndex := -1
	for {
		url := fmt.Sprintf("https://xihuwenti.juyancn.cn/wechat/product/details?id=%d", pageId)
		if currentDateIndex >= 0 {
			url = fmt.Sprintf("%s&time=%s", url, dates[currentDateIndex])
		}
		logrus.WithField("url", url).Info("fetch page")
		res, err := http.Get(url)
		if err != nil {
			logrus.WithField("url", url).WithError(err).Error("get page error")
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			logrus.WithField("url", url).WithField("status", res.Status).Error("status code error")
			return
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			logrus.WithField("url", url).WithError(err).Errorf("parse document")
			return
		}

		texts := make([]string, 0)
		for _, n := range doc.Find(".date-box .cur span").Nodes {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				texts = append(texts, c.Data)
			}
		}
		currentDateText := strings.Join(texts, " ")

		// Find the available court items
		doc.Find(".can-select").Each(func(i int, s *goquery.Selection) {
			start, _ := s.Attr("data-start")
			end, _ := s.Attr("data-end")
			hall, _ := s.Attr("data-hall_name")

			startT, err := time.Parse("15:04", start)
			if err != nil {
				logrus.Errorf("parse start time: %v", err)
				return
			}
			endT, err := time.Parse("15:04", end)
			if err != nil {
				logrus.Errorf("parse end time: %v", err)
				return
			}
			if startT.Sub(startTime) >= 0 && endT.Sub(endTime) <= 0 {
				info := fmt.Sprintf("%s, %s-%s %s", currentDateText, start, end, hall)
				alertInfo = append(alertInfo, info)
			}
		})

		// Find the date items
		if currentDateIndex == -1 {
			doc.Find(".date-box li").Each(func(i int, s *goquery.Selection) {
				if s.HasClass("cur") {
					return
				}
				date, ok := s.Attr("data-time")
				if ok {
					dates = append(dates, date)
				} else {
					logrus.WithField("url", url).Warn("not founed the date-time attribute")
				}
			})
		}
		logrus.Infof("check done %s", currentDateText)

		currentDateIndex++
		if currentDateIndex >= len(dates) {
			break
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	sendAlert(alertInfo)
}

func main() {
	start := "18:00"
	end := "23:00"
	flag.IntVar(&interval, "interval", interval, "check interval")
	flag.IntVar(&pageId, "pageID", pageId, "page id, 753:羽毛球")
	flag.StringVar(&feishu, "feishu", "", "feishu webhook url")
	flag.StringVar(&start, "start", start, "court start time")
	flag.StringVar(&end, "end", end, "court end time")
	flag.Parse()

	var err error
	startTime, err = time.Parse("15:04", start)
	if err != nil {
		logrus.WithError(err).Errorf("parse start time")
		return
	}
	endTime, err = time.Parse("15:04", end)
	if err != nil {
		logrus.WithError(err).Errorf("parse end time")
		return
	}

	for {
		check()
		time.Sleep(time.Second * time.Duration(interval))
	}
}
