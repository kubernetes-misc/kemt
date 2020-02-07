package main

import (
	"github.com/kubernetes-misc/kemt/client"
)

func main() {

	client.BuildClient()
	client.GetEvents("default")

}
