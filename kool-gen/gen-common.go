package main

func (c Controller) hasSyncedFields() []string {
	fields := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		fields = append(fields, c.Resources[i].LowerKind+"Synced cache.InformerSynced")
	}
	return fields
}

func (c Controller) structureFieldInits() []string {
	expressions := make([]string, 0, 2*len(c.Resources))
	for i := range c.Resources {
		// c.<kind>Lister := <kind>Informer.Lister()
		// c.<kind>Synced := <kind>Informer.Informer().HasSynced
		expressions = append(expressions, "c."+c.Resources[i].LowerKind+"Lister = c."+c.Resources[i].LowerKind+"Informer.Lister()")
		expressions = append(expressions, "c."+c.Resources[i].LowerKind+"Synced = c."+c.Resources[i].LowerKind+"Informer.Informer().HasSynced")
	}
	return expressions
}
