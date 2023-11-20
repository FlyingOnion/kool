package main

func (c Controller) globalListerFields() []string {
	fields := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		fields = append(fields, c.Resources[i].LowerKind+"Lister kool.Lister["+c.Resources[i].GoType+"]")
	}
	return fields
}

func (c Controller) globalInformerInits() []string {
	expressions := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		expressions = append(expressions, c.Resources[i].LowerKind+`Informer := kool.NewInformer[`+c.Resources[i].GoType+`](client, 30*time.Second)`)
	}
	return expressions
}

func (c Controller) globalNewControllerArgs() []string {
	expressions := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		expressions = append(expressions, c.Resources[i].LowerKind+`Informer kool.Informer[`+c.Resources[i].GoType+`],`)
	}
	return expressions
}
