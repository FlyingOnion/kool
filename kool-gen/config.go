package main

import (
	"errors"
	"slices"
	"strings"
)

type Controller struct {
	Name string `json:"name"`
	// Enqueue   string     `json:"enqueue"`
	Retry     int        `json:"retryOnError"`
	Namespace string     `json:"namespace"`
	Resources []Resource `json:"resources"`

	ListerFields    []string `json:"-"`
	HasSyncedFields []string `json:"-"`

	StructFieldInits []string `json:"-"`
	InformerInits    []string `json:"-"`
}

type Resource struct {
	Group   string
	Version string
	Kind    string

	Package string
	Alias   string

	CustomHandlers []string `yaml:"customHandlers"`

	LowerKind    string `json:"-"`
	CustomAdd    bool   `json:"-"`
	CustomUpdate bool   `json:"-"`
	CustomDelete bool   `json:"-"`
}

type Import struct {
	Alias string
	Pkg   string
}

var (
	errInvalidRetry = errors.New("retryOnError must be between 0 and 10")
	errNoResources  = errors.New("no resource to control")
)

func (c *Controller) initAndValidate() error {
	if len(c.Name) == 0 {
		c.Name = "Controller"
	}
	if c.Retry < 0 || c.Retry > 10 {
		return errInvalidRetry
	}
	// initializations below uses len(c.Resources)
	// so we need to ensure that it is not 0
	if len(c.Resources) == 0 {
		return errNoResources
	}
	for i := range c.Resources {
		c.Resources[i].LowerKind = strings.ToLower(c.Resources[i].Kind)
		c.Resources[i].CustomAdd = i == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Add")
		c.Resources[i].CustomUpdate = i == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Update")
		c.Resources[i].CustomDelete = i == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Delete")
	}

	if len(c.Namespace) == 0 {
		c.ListerFields = c.globalListerFields()
		c.InformerInits = c.globalInformerInits()
	} else {
		c.ListerFields = c.namespacedListerFields()
		c.InformerInits = c.namespacedInformerInits()
	}
	c.HasSyncedFields = c.hasSyncedFields()
	c.StructFieldInits = c.structureFieldInits()

	// for i := range c.Resources {
	// 	if len(c.Resources[i].Group) == 0 {
	// 		// k8s builtin types
	// 		pkg, ok := importMap[c.Resources[i].Kind]
	// 		if !ok {
	// 			return errors.New("unknown kind: " + c.Resources[i].Kind)
	// 		}
	// 		c.Resources[i].Package = pkg.Pkg
	// 		c.Resources[i].Alias = pkg.Alias
	// 		if len(c.Resources[i].Version) == 0 {
	// 			c.Resources[i].Version = "v1"
	// 		}
	// 		continue
	// 	}
	// }

	return nil
}
