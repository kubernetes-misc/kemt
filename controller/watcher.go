package controller

import (
	"github.com/kubernetes-misc/kemt/client"
	"github.com/kubernetes-misc/kemt/model"
	"github.com/sirupsen/logrus"
)

var watchers = make(map[string]*watcher)

func CreateIfNotExists(item model.KemtV1) {
	//TODO: support for deleting CRDs
	watcher, found := watchers[item.ID()]
	if found {
		logrus.Debugln("already aware of", item.ID())
		watcher.tc.UpdateEndpoint(item.Spec.WebHook)
		//TODO: update the info in the CRD
		return
	}
	logrus.Println("> Watching", item.Metadata.Namespace)
	w := newWatcher(item)
	watchers[item.ID()] = w
}

type watcher struct {
	stop chan interface{}
	tc   *client.TeamsClient
}

func newWatcher(k model.KemtV1) *watcher {
	r := &watcher{
		stop: make(chan interface{}),
		tc:   client.NewTeamsClient(10, 1, k.Spec.WebHook),
	}
	c := client.GetEvents(k.Metadata.Namespace)
	r.tc.Start()
	go func() {
		for i := range c {
			r.tc.EnqueueMsg(i.ToString())
		}
	}()
	return r
}
