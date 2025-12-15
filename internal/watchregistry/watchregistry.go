package watchregistry

import (
	"slices"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SupportedGVKs = []schema.GroupVersionKind{
	// Core
	{Group: "", Version: "v1", Kind: "ConfigMap"},
	{Group: "", Version: "v1", Kind: "Secret"},

	// Workloads
	{Group: "apps", Version: "v1", Kind: "Deployment"},
	{Group: "apps", Version: "v1", Kind: "StatefulSet"},

	// Batch
	{Group: "batch", Version: "v1", Kind: "Job"},
}

// Parse GroupVersionKind from apiVersion and kind strings.
func CanonicalGVK(apiVersion, kind string) (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}, nil
}

// Check if the given GroupVersionKind is supported.
func IsSupportedGVK(gvk schema.GroupVersionKind) bool {
	return slices.Contains(SupportedGVKs, gvk)
}

// Get a list of supported GroupVersionKind strings.
func SupportedGVKStrings() []string {
	result := make([]string, 0, len(SupportedGVKs))
	for _, gvk := range SupportedGVKs {
		result = append(result, gvk.String())
	}
	return result
}
