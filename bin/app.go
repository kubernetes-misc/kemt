package main

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Println("ooh")
	client.BuildClient()
	client.GetEvents("")

}
