package main

import (
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Controller struct {
	Base string `yaml:"base"`
	Name string `yaml:"name"`
	// Enqueue   string     `yaml:"enqueue"`
	Retry     int        `yaml:"retryOnError"`
	Namespace string     `yaml:"namespace"`
	Resources []Resource `yaml:"resources"`

	// template: controller
	//  type Controller struct {
	//      xxxLister kool.Lister           // global
	//      xxxLister kool.NamespacedLister // namespaced
	//  }
	ListerFields []string `yaml:"-"`

	// template: controller
	//  type Controller struct {
	//      xxxHasSynced cache.InformerSynced // common
	//  }
	HasSyncedFields []string `yaml:"-"`

	// template: controller
	//  c.xxxLister := xxxInformer.Lister()             // common
	//  c.xxxSynced := xxxInformer.Informer().HasSynced // common
	StructFieldInits []string `yaml:"-"`

	// template: main
	//  xxxInformer := kool.NewInformer           // global
	//  xxxInformer := kool.NewNamespacedInformer // namespaced
	InformerInits []string `yaml:"-"`

	// template: controller
	//  func NewController(
	//      xxxInformer kool.Informer,           // global
	//      xxxInformer kool.NamespacedInformer, // namespaced
	//  )
	NewControllerArgs []string `yaml:"-"`

	Imports []Import `yaml:"-"`
}

type Resource struct {
	Group   string
	Version string
	Kind    string

	Package string
	// Alias   string

	CustomHandlers []string `yaml:"customHandlers"`

	LowerKind    string `yaml:"-"`
	GoType       string `yaml:"-" yaml:"-"`
	CustomAdd    bool   `yaml:"-"`
	CustomUpdate bool   `yaml:"-"`
	CustomDelete bool   `yaml:"-"`
	GenDeepCopy  bool   `yaml:"-"`
}

type Import struct {
	Alias string
	Pkg   string
}

type ImportList []Import

func (i ImportList) Len() int           { return len(i) }
func (l ImportList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ImportList) Less(i, j int) bool { return l[i].Pkg < l[j].Pkg }

const (
	msgConfigInvalid             = `config is invalid`
	msgInvalidRetry              = `retryOnError must be between 0 and 10`
	msgNoResources               = `no resource to control`
	msgUnknownResourceKind       = `unknown resource kind`
	msgUnknownResourceKindTip    = `if you need to control a builtin resource, set package to k8s.io/api/<package-group>/<version> and try again`
	msgNoVersionInPackage        = `no version information in package`
	msgUseDefaultVersionV1       = `use default version "v1" as resource version`
	msgIncompatibility           = `this may cause incompatibility`
	msgInconsistentVersion       = `version information in package is inconsistent with resource version`
	msgInvalidThirdPartyGroup    = `invalid third-party group name; group name cannot be any of ` + k8sBuiltinGroupsString + ` or ends with ".k8s.io" because they are k8s builtin groups`
	msgInvalidThirdPartyGroupTip = `if you need a builtin resource, leave group empty, set package to k8s.io/api/<package-group>/<version> and try again`
)

func defaultController() *Controller {
	return &Controller{
		Base:  ".",
		Name:  "Controller",
		Retry: 3,
	}
}

func (c *Controller) initAndValidate() {
	if len(c.Name) == 0 {
		c.Name = "Controller"
	}
	if c.Retry < 0 || c.Retry > 10 {
		log.Fatal(msgConfigInvalid, "cause", msgInvalidRetry)
	}
	// initializations below uses len(c.Resources)
	// so we need to ensure that it is not 0
	if len(c.Resources) == 0 {
		log.Fatal(msgConfigInvalid, "cause", msgNoResources)
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
			initAndValidateBuiltinResource(&(c.Resources[i]), imports)
		} else {
			initAndValidateThirdPartyResource(&(c.Resources[i]), imports)
		}
	}
	importList := imports.UnsortedList()
	sort.Sort(ImportList(importList))
	c.Imports = importList

	if len(c.Namespace) == 0 {
		c.ListerFields = c.globalListerFields()
		c.InformerInits = c.globalInformerInits()
		c.NewControllerArgs = c.globalNewControllerArgs()
	} else {
		c.ListerFields = c.namespacedListerFields()
		c.InformerInits = c.namespacedInformerInits()
		c.NewControllerArgs = c.namespacedNewControllerArgs()
	}
	c.HasSyncedFields = c.hasSyncedFields()
	c.StructFieldInits = c.structureFieldInits()
}

func getVersionFromPackage(pkg string) (string, bool) {
	for _, str := range strings.Split(pkg, "/") {
		if versionRegex.MatchString(str) {
			return str, true
		}
	}
	return "v1", false
}

func initAndValidateThirdPartyResource(r *Resource, imports sets.Set[Import]) {
	if isK8sBuiltinGroup(r.Group) {
		log.Fatal(msgConfigInvalid,
			"cause", msgInvalidThirdPartyGroup,
			"group", r.Group,
			"tip", msgInvalidThirdPartyGroupTip,
		)
	}

	version, found := getVersionFromPackage(r.Package)
	if !found {
		log.Warn(msgNoVersionInPackage, "package", r.Package)
		log.Warn(msgIncompatibility)
		r.Version = version
		return
	}
	// version found but inconsistent
	if version != r.Version {
		log.Warn(msgInconsistentVersion, "package version", version, "resource version", r.Version)
		log.Warn(msgIncompatibility)
	}

	// check if resource is defined in the same package
	if len(r.Package) == 0 {
		// g > 0 && p == 0 => local import
		// no need to import
		r.GoType = r.Kind
		return
	}

	alias := getAlias(r.Package)
	r.GoType = alias + "." + r.Kind
	imports.Insert(Import{Alias: alias, Pkg: r.Package})
}

func initAndValidateBuiltinResource(r *Resource, imports sets.Set[Import]) {
	pkgGroup, ok := kind2Group(r.Kind)
	if !ok && len(r.Package) == 0 {
		log.Fatal(
			msgConfigInvalid,
			"cause", msgUnknownResourceKind,
			"kind", r.Kind,
			"tip", msgUnknownResourceKindTip,
		)
	}

	emptyVersion, emptyPackage := len(r.Version) == 0, len(r.Package) == 0
	switch {
	case emptyVersion && emptyPackage:
		r.Version = "v1"
		r.Package = "k8s.io/api/" + pkgGroup + "/v1"
	case emptyVersion:
		version, found := getVersionFromPackage(r.Package)
		if !found {
			log.Warn(msgNoVersionInPackage, "package", r.Package)
			log.Warn(msgIncompatibility)
		}
		r.Version = version
	case emptyPackage:
		r.Package = "k8s.io/api/" + pkgGroup + "/" + r.Version
	default:
		version, found := getVersionFromPackage(r.Package)
		if found && version != r.Version {
			log.Warn(msgInconsistentVersion, "kind", r.Kind, "package version", version, "resource version", r.Version)
			log.Warn(msgIncompatibility)
		}
	}
	alias := getAlias(r.Package)
	r.GoType = alias + "." + r.Kind
	imports.Insert(Import{Alias: alias, Pkg: r.Package})
	if r.GenDeepCopy {
		log.Info(msgNoNeedToGenDeepCopy, "kind", r.Kind)
		r.GenDeepCopy = false
	}
}
