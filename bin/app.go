package main

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/kubernetes-misc/kemt/controller"
	"github.com/kubernetes-misc/kemt/model"
	cronV3 "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"os"
)

const DefaultCronSpec = "*/30 * * * * *"

func main() {
	logrus.Println("Kubernetes Declarative Teams Integration")
	logrus.Println("Starting up...")

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
