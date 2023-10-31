package main

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
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

	Imports []Import `json:"-"`
}

type Resource struct {
	Group   string
	Version string
	Kind    string

	Package string
	Alias   string

	CustomHandlers []string `yaml:"customHandlers"`

	LowerKind    string `json:"-"`
	GoType       string `json:"-" yaml:"-"`
	CustomAdd    bool   `json:"-"`
	CustomUpdate bool   `json:"-"`
	CustomDelete bool   `json:"-"`
}

type Import struct {
	Alias string
	Pkg   string
}

type ImportList []Import

func (i ImportList) Len() int           { return len(i) }
func (l ImportList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ImportList) Less(i, j int) bool { return l[i].Pkg < l[j].Pkg }

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

	// imports is used to deal with extra imports
	// it collects all unique imports and generates Controller.Imports
	imports := sets.Set[Import]{}

	for i := range c.Resources {
		// field initializations
		c.Resources[i].LowerKind = strings.ToLower(c.Resources[i].Kind)
		c.Resources[i].CustomAdd = i == 0 || len(c.Resources[i].CustomHandlers) == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Add")
		c.Resources[i].CustomUpdate = i == 0 || len(c.Resources[i].CustomHandlers) == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Update")
		c.Resources[i].CustomDelete = i == 0 || len(c.Resources[i].CustomHandlers) == 0 || slices.Contains(c.Resources[i].CustomHandlers, "Delete")

		if len(c.Resources[i].Group) == 0 {
			// it's a k8s builtin type
			imp, ok := importMap[c.Resources[i].Kind]
			if !ok {
				return errors.New("unknown builtin kind: " + c.Resources[i].Kind)
			}
			c.Resources[i].Package = imp.Pkg
			c.Resources[i].Alias = imp.Alias
			if len(c.Resources[i].Version) == 0 {
				c.Resources[i].Version = "v1"
			}
			c.Resources[i].GoType = c.Resources[i].Alias + "." + c.Resources[i].Kind
			imports.Insert(imp)
			continue
		}
		// custom type
		if len(c.Resources[i].Package) == 0 {
			// the resource definition is in the same package
			// no need to import
			c.Resources[i].GoType = c.Resources[i].Kind
		} else {
			c.Resources[i].Alias = getAlias(c.Resources[i].Package)
			c.Resources[i].GoType = c.Resources[i].Alias + "." + c.Resources[i].Kind
			imports.Insert(Import{Alias: c.Resources[i].Alias, Pkg: c.Resources[i].Package})
		}
	}
	c.Imports = imports.UnsortedList()
	sort.Sort(ImportList(c.Imports))

	if len(c.Namespace) == 0 {
		c.ListerFields = c.globalListerFields()
		c.InformerInits = c.globalInformerInits()
	} else {
		c.ListerFields = c.namespacedListerFields()
		c.InformerInits = c.namespacedInformerInits()
	}
	c.HasSyncedFields = c.hasSyncedFields()
	c.StructFieldInits = c.structureFieldInits()

	return nil
}
