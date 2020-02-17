package controller

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/kubernetes-misc/kemt/model"
	"github.com/sirupsen/logrus"
)

var watchers = make(map[string]*watcher)

func CreateIfNotExists(item model.KemtV1) {
	//TODO: support for deleting CRDs
	_, found := watchers[item.ID()]
	if found {
		logrus.Debugln("already aware of", item.ID())
		//TODO: update the info in the CRD
		return
	}
	logrus.Println("> Watching", item.Metadata.Namespace)
	w := newWatcher(item)
	watchers[item.ID()] = w
}

type watcher struct {
	stop chan interface{}
}

func newWatcher(k model.KemtV1) *watcher {
	r := &watcher{
		stop: make(chan interface{}),
	}
	c := client.GetEvents(k.Metadata.Namespace)
	tc := client.NewTeamsClient(10, 1, k.Spec.WebHook)
	tc.Start()
	go func() {
		for i := range c {
			tc.EnqueueMsg(i.ToString())
		}
	}()
	return r
}
