package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type TeamsClient struct {
	MaxMessages    int
	MaxWaitSeconds string
	Endpoint       string
	in             chan string
}

func (t *TeamsClient) Start() {
	duration, err := time.ParseDuration(t.MaxWaitSeconds)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	go func() {
		count := 0
		msg := Webhook{ThemeColor: "#dddddd"}
		var line string
		for {
			select {
			case line = <-t.in:
				msg.Text += line
				msg.Text += "<br />"
				count++
				if count >= t.MaxMessages {
					sendSilently(t.Endpoint, msg)
					count = 0
					msg.Text = ""
				}
			case <-time.After(duration):
				if count > 0 {
					sendSilently(t.Endpoint, msg)
					count = 0
					msg.Text = ""
				}
			}
		}
	}()
}

func (t *TeamsClient) EnqueueMsg(in string) {
	t.in <- in
}

type Webhook struct {
	Text       string `json:"text,omitempty"`
	Title      string `json:"title,omitempty"`
	ThemeColor string `json:"themeColor,omitempty"`
}

func sendSilently(endpoint string, msg Webhook) {
	enc, err := json.Marshal(msg)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	b := bytes.NewBuffer(enc)
	res, err := http.Post(endpoint, "application/json", b)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	if res.StatusCode >= 299 {
		logrus.Errorln(fmt.Errorf("Error on message: %s\n", res.Status))
		return
	}
	//TODO: handle end
	fmt.Println(res.Status)
}
