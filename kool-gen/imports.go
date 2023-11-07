package main

import (
	"regexp"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v\d+((alpha|beta|rc)\d+)?$`)

var (
	appsv1       = Import{"appsv1", "k8s.io/api/apps/v1"}
	batchv1      = Import{"batchv1", "k8s.io/api/batch/v1"}
	corev1       = Import{"corev1", "k8s.io/api/core/v1"}
	discoveryv1  = Import{"discoveryv1", "k8s.io/api/discovery/v1"}
	networkingv1 = Import{"networkingv1", "k8s.io/api/networking/v1"}
	rbacv1       = Import{"rbacv1", "k8s.io/api/rbac/v1"}
	schedulingv1 = Import{"schedulingv1", "k8s.io/api/scheduling/v1"}
	storagev1    = Import{"storagev1", "k8s.io/api/storage/v1"}
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

func getAlias(pkg string) string {
	s := strings.Split(pkg, "/")
	switch len(s) {
	case 1:
		return pkg
	case 2:
		return s[1]
	}
	if versionRegex.MatchString(s[len(s)-1]) {
		return s[len(s)-2] + s[len(s)-1]
	}
	return s[len(s)-1]
}
