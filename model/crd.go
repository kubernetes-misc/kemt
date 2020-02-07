package model

import "k8s.io/apimachinery/pkg/runtime/schema"

var KemtV1CRDSchema = schema.GroupVersionResource{
	Group:    "kubernetes-misc.xyz",
	Version:  "v1",
	Resource: "kempts",
}

type KemtV1 struct {
	Metadata MetadataV1 `json:"metadata"`
	Spec     SpecV1     `json:"spec"`
}

func (i KemtV1) GetID() string {
	return "kemtV1." + "TODO" //TODO: implement
}

type MetadataV1 struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type SpecV1 struct {
	SecretName string `json:"secretName"`
}
