package main

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/kubernetes-misc/kemt/controller"
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
	go web.StartServer(":7000")

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
		F: update,
	})
	c.Start()
	update()
	if err != nil {
		panic(err)
	}
	logrus.SetLevel(logrus.InfoLevel)

	err, out := client.WatchCRD("", model.KemtV1CRDSchema)
	if err != nil {
		logrus.Errorln("could not watch CRD KemtV1")
		logrus.Panicln(err)
	}
	for o := range out {
		if o.Type == "ADDED" || o.Type == "UPDATED" {
			controller.CreateIfNotExists(o.Kemt)
			continue
		}
		logrus.Println("new line", o.Type)
	}
	select {}

}

func update() {
	crds, err := client.GetAllCRD("", model.KemtV1CRDSchema)
	if err != nil {
		logrus.Errorln("could not get all CRDs of Kemt V1")
		logrus.Errorln(err)
		return
	}
	for _, crd := range crds {
		controller.CreateIfNotExists(crd)
	}

}
