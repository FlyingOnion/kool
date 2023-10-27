package main

import (
	"errors"
)

type Controller struct {
	Name string `json:"name"`
	// Enqueue   string     `json:"enqueue"`
	Retry     int        `json:"retryOnError"`
	Namespace string     `json:"namespace"`
	Resources []Resource `json:"resources"`

	HasSyncedFields []string `json:"-"`

	ListerFields  []string `json:"-"`
	InformerInits []string `json:"-"`
}

type Resource struct {
	Group   string
	Version string
	Kind    string
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
	if len(c.Namespace) == 0 {
		c.ListerFields = c.globalListerFields()
		c.InformerInits = c.globalInformerInits()
	} else {
		c.ListerFields = c.namespacedListerFields()
		c.InformerInits = c.namespacedInformerInits()
	}
	c.HasSyncedFields = c.hasSyncedFields()
	if len(c.Resources) == 0 {
		return errNoResources
	}

	return nil
}
