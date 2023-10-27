package main

import "strings"

func (c Controller) hasSyncedFields() []string {
	fields := make([]string, 0, len(c.Resources))
	for i := range c.Resources {
		fields = append(fields, strings.ToLower(c.Resources[i].Kind)+"Synced cache.InformerSynced")
	}
	return fields
}
