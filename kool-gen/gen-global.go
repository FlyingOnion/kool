package main

import "strings"

func (c Controller) globalListerFields() []string {
	fields := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		fields = append(fields, strings.ToLower(c.Resources[i].Kind)+"Lister kool.Lister["+c.Resources[i].Kind+"]")
	}
	return fields
}

func (c Controller) globalInformerInits() []string {
	expressions := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		expressions = append(expressions, strings.ToLower(c.Resources[i].Kind)+`Informer := kool.NewInformer[`+c.Resources[i].Kind+`](client, 30*time.Second)`)
	}
	return expressions
}
