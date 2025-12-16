package resourcepoller

// ResourceRef describes any Kubernetes object
type ResourceRef struct {
	APIVersion string // e.g. "apps/v1", "v1"
	Kind       string // e.g. "Deployment", "ConfigMap"
	Name       string
	Namespace  string   // empty = cluster-scoped
	Fields     []string // JSON field paths, e.g. "spec.replicas"
}
