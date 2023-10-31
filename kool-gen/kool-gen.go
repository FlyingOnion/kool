package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

var importMap = map[string]Import{
	"Deployment":  appsv1,
	"StatefulSet": appsv1,
	"ReplicaSet":  appsv1,
	"DaemonSet":   appsv1,

	"Job":     batchv1,
	"CronJob": batchv1,

	"Binding":               corev1,
	"Pod":                   corev1,
	"PodTemplate":           corev1,
	"Endpoints":             corev1,
	"ReplicationController": corev1,
	"Node":                  corev1,
	"Namespace":             corev1,
	"Service":               corev1,
	"ServiceAccount":        corev1,
	"ConfigMap":             corev1,
	"Secret":                corev1,
	"LimitRange":            corev1,
	"ResourceQuota":         corev1,
	"PersistentVolume":      corev1,
	"PersistentVolumeClaim": corev1,

	"EndpointSlice": discoveryv1,

	"Ingress":       networkingv1,
	"IngressClass":  networkingv1,
	"NetworkPolicy": networkingv1,

	"Role":               rbacv1,
	"RoleBinding":        rbacv1,
	"ClusterRole":        rbacv1,
	"ClusterRoleBinding": rbacv1,

	"PriorityClass": schedulingv1,

	"CSIDriver":          storagev1,
	"CSINodes":           storagev1,
	"CSIStorageCapacity": storagev1,
	"StorageClass":       storagev1,
}

func main() {
	f, err := os.Open("koolpod-controller.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	config := Controller{
		// Enqueue: "ratelimiting",
		Retry: 3,
	}
	yaml.NewDecoder(f).Decode(&config)
	if err = config.initAndValidate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tmpl := template.New("base").Funcs(sprig.FuncMap())
	d, err := os.ReadDir("../tmpl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, dirEntry := range d {
		name := dirEntry.Name()
		if strings.HasSuffix(name, "go.tmpl") {
			tmpl2, err := tmpl.New(name).ParseFiles("../tmpl/" + name)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			f, err := os.Create("./gen/" + strings.TrimSuffix(name, ".tmpl"))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			tmpl2.Execute(f, config)
			f.Close()
		}
	}
}
