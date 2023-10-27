package main

import (
	"regexp"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v\d+((alpha|beta|rc)\d+)?$`)

var (
	appsv1  = Import{"appsv1", "k8s.io/api/apps/v1"}
	batchv1 = Import{"batchv1", "k8s.io/api/batch/v1"}
	corev1  = Import{"corev1", "k8s.io/api/core/v1"}
)

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
