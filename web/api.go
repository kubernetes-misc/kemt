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
