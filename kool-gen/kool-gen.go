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
	"Pod":                   corev1,
	"Node":                  corev1,
	"Namespace":             corev1,
	"Service":               corev1,
	"ConfigMap":             corev1,
	"Secret":                corev1,
	"PersistentVolume":      corev1,
	"PersistentVolumeClaim": corev1,

	"Deployment":  appsv1,
	"StatefulSet": appsv1,
	"ReplicaSet":  appsv1,
	"DaemonSet":   appsv1,

	"Job":     batchv1,
	"CronJob": batchv1,
}

func main() {
	f, err := os.Open("koolpod-controller.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	config := Controller{
		Enqueue: "ratelimiting",
		Retry:   3,
	}
	yaml.NewDecoder(f).Decode(&config)
	if len(config.Resources) == 0 {
		fmt.Println("No resource to control")
		os.Exit(1)
	}
	// init config
	if len(config.Name) == 0 {
		config.Name = "Controller"
	}
	for i := range config.Resources {
		if len(config.Resources[i].Group) == 0 && len(config.Resources[i].Version) == 0 {

		}
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
