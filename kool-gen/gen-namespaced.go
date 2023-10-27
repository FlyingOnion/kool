package main

import "strings"

func (c Controller) namespacedListerFields() []string {
	fields := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		fields = append(fields, strings.ToLower(c.Resources[i].Kind)+"Lister kool.NamespacedLister["+c.Resources[i].Kind+"]")
	}
	return fields
}

func (c Controller) namespacedInformerInits() []string {
	expressions := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		expressions = append(expressions, strings.ToLower(c.Resources[i].Kind)+`Informer := kool.NewNamespacedInformer[`+c.Resources[i].Kind+`](client, `+c.Namespace+`, 30*time.Second)`)
	}
	return expressions
}
