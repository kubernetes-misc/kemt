package client

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kubernetes-misc/kemt/model"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var clientset *kubernetes.Clientset
var dynClient dynamic.Interface

func BuildClient() (err error) {
	var config *rest.Config

	authInCluster := os.Getenv("authInCluster")
	if authInCluster == "true" {
		config, err = rest.InClusterConfig()
		if err != nil {
			logrus.Errorln("could not auth in cluster")
			panic(err.Error())
		}
	} else {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			logrus.Errorln(err)
			return
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	dynClient, err = dynamic.NewForConfig(config)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("Error received creating client %v", err))
		return
	}
	return
}

type WrappedKemtV1 struct {
	Type string
	Kemt model.KemtV1
}

func GetPods(ns string) []string {
	result := []string{}
	ls, err := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorln(err)
		return result
	}
	for _, item := range ls.Items {
		result = append(result, item.Name)
	}
	return result
}

func GetDeployments(ns string) []string {
	result := []string{}
	ls, err := clientset.AppsV1().Deployments(ns).List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorln(err)
		return result
	}
	for _, item := range ls.Items {
		result = append(result, item.Name)
	}
	return result
}

func GetNS() []string {
	var result []string
	l, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorln(err)
		return result
	}
	for _, ns := range l.Items {
		result = append(result, ns.Name)
	}
	sort.Strings(result)
	return result

}

func WatchCRD(ns string, crd schema.GroupVersionResource) (err error, out chan WrappedKemtV1) {
	crdClient := dynClient.Resource(crd)
	w, err := crdClient.Namespace(ns).Watch(metav1.ListOptions{})
	if err != nil {
		logrus.Errorln("could not watch CRD")
		logrus.Errorln(err)
		return
	}
	out = make(chan WrappedKemtV1, 256)
	go func() {
		for r := range w.ResultChan() {
			logrus.Println("New change to CRD")
			logrus.Println(r.Type)
			b, err := json.Marshal(r.Object)
			if err != nil {
				logrus.Errorln("could not marshal event.Object")
				logrus.Errorln(err)
				continue
			}
			item := &model.KemtV1{}
			err = json.Unmarshal(b, item)
			if err != nil {
				logrus.Errorln("could not unmarshal event.Object into model.KemtV1")
				logrus.Errorln(err)
				continue
			}
			b, _ = json.Marshal(*item)
			logrus.Println(string(pretty.Pretty(b)))
			result := WrappedKemtV1{
				Type: string(r.Type),
				Kemt: *item,
			}
			out <- result
		}

	}()
	return
}

func GetAllCRD(namespace string, crd schema.GroupVersionResource) (result []model.KemtV1, err error) {
	logrus.Debugln("== getting CRDs ==")
	crdClient := dynClient.Resource(crd)
	ls, err := crdClient.Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorln(fmt.Errorf("could not list %s", err))
		return
	}

	result = make([]model.KemtV1, len(ls.Items))
	for i, entry := range ls.Items {
		b, err := entry.MarshalJSON()
		if err != nil {
			logrus.Errorln(err)
			continue
		}
		cs := model.KemtV1{}
		err = json.Unmarshal(b, &cs)
		if err != nil {
			logrus.Errorln(err)
		}
		result[i] = cs
	}
	return
}

type Event struct {
	EventType string `json:"Type"`
	Object    Object `json:"Object"`
}

func (e Event) ToString() string {
	oc, err := time.LoadLocation("Local")
	if err != nil {
		oc, _ = time.LoadLocation("")
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", e.Object.LastTimestamp.In(oc), e.EventType, pad(e.Object.Reason, 20), pad(fmt.Sprintf("%s/%s", strings.ToLower(e.Object.InvolvedObject.Kind), e.Object.InvolvedObject.Name), 35), e.Object.Message)
}

func pad(in string, size int) string {
	for len(in) < size {
		in += " "
	}
	return in
}

type Metadata struct {
	UID string `json:"uid"`
}

type Object struct {
	Metadata       Metadata       `json:"metadata"`
	InvolvedObject InvolvedObject `json:"involvedObject"`
	Reason         string         `json:"reason"`
	Message        string         `json:"message"`
	Type           string         `json:"type"`
	Count          int            `json:"count"`
	LastTimestamp  time.Time      `json:"lastTimeStamp"`
}

type InvolvedObject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func GetLogs(ns, item string) (result chan string, err error, close func() error) {
	s, err := clientset.CoreV1().Pods(ns).GetLogs(item, &v1.PodLogOptions{
		TypeMeta: metav1.TypeMeta{},
		Follow:   true,
	}).Stream()
	if err != nil {
		logrus.Errorln(err)
		return
	}

	result = make(chan string)
	close = func() error { return s.Close() }

	go func() {
		rd := bufio.NewReader(s)
		for {
			str, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			result <- str
		}
	}()
	return
}

func GetEvents(ns string) chan Event {
	result := make(chan Event)
	go func() {
		logrus.Debugln("== getting events ==")
		w, err := clientset.CoreV1().Events(ns).Watch(metav1.ListOptions{})
		if err != nil {
			logrus.Errorln(err)
			return
		}
		for e := range w.ResultChan() {
			b, _ := json.Marshal(e)
			e := Event{}
			err := json.Unmarshal(b, &e)
			if err != nil {
				logrus.Errorln(err)
				continue
			}
			result <- e
		}
		close(result)
	}()
	return result
}
