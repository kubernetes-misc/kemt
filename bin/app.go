package main

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/kubernetes-misc/kemt/model"
	"github.com/kubernetes-misc/kemt/web"
	cronV3 "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"os"
)

const DefaultCronSpec = "*/30 * * * * *"

func main() {
	logrus.Println("Kubernetes Declarative Teams Integration")
	logrus.Println("Starting up...")

	listen := os.Getenv("listen")
	if listen == "" {
		listen = ":8080"
		logrus.Println("defaulting listen to", listen)
	}

	go web.StartServer(listen)

	err := client.BuildClient()
	if err != nil {
		panic(err)
	}
	cronSpec := os.Getenv("cronSpec")
	if cronSpec == "" {
		logrus.Println("cronSpec env is empty. Defaulting to", DefaultCronSpec)
		cronSpec = DefaultCronSpec
	}
	c := cronV3.New(cronV3.WithSeconds())
	_, err = c.AddJob(cronSpec, model.Job{
		F: func() {

		},
	})
	c.Start()
	if err != nil {
		panic(err)
	}
	logrus.SetLevel(logrus.InfoLevel)

	select {}

}
