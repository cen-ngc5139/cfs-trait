package controllers

import "github.com/ghostbaby/cfs-trait/controllers/models"

func GenerateLabels(labels map[string]string) map[string]string {
	dynLabels := map[string]string{
		"trait.ghostbaby.com/v1/managed-by": models.DefaultCfsTraitLabel,
	}
	return MergeLabels(dynLabels, labels)
}

func MergeLabels(allLabels ...map[string]string) map[string]string {
	res := map[string]string{}

	for _, labels := range allLabels {
		for k, v := range labels {
			res[k] = v
		}
	}
	return res
}
