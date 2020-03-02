package web

import (
	"encoding/json"
	"github.com/kubernetes-misc/kemt/client"
	"net/http"
)

func handleAPINamespaces(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(client.GetNS())
	w.Write(b)
}

func handleAPIDeployments(w http.ResponseWriter, r *http.Request) {
	ns := getGetParam(r, "namespace")
	b, _ := json.Marshal(client.GetDeployments(ns))
	w.Write(b)
}

func handleAPIPods(w http.ResponseWriter, r *http.Request) {
	ns := getGetParam(r, "namespace")
	b, _ := json.Marshal(client.GetPods(ns))
	w.Write(b)
}

func getGetParam(r *http.Request, s string) string {
	keys, ok := r.URL.Query()[s]
	if !ok || len(keys[0]) < 1 {
		return ""
	}
	return keys[0]
}
