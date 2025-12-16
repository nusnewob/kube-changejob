package resourcepoller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Poller fetches and hashes Kubernetes resources
type Poller struct {
	Client client.Client
}

// PollResult contains extracted data and its hash
type PollResult struct {
	Hash string
	Data map[string]any
}

// Poll fetches the resource, extracts fields, and hashes them
func (p *Poller) Poll(ctx context.Context, ref ResourceRef) (*PollResult, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ref.APIVersion)
	obj.SetKind(ref.Kind)

	key := client.ObjectKey{
		Name: ref.Name,
	}
	if ref.Namespace != "" {
		key.Namespace = ref.Namespace
	}

	if err := p.Client.Get(ctx, key, obj); err != nil {
		return nil, err
	}

	extracted := make(map[string]any)

	for _, field := range ref.Fields {
		val, found, err := unstructured.NestedFieldNoCopy(
			obj.Object,
			strings.Split(field, ".")...,
		)
		if err != nil {
			return nil, err
		}
		if found {
			extracted[field] = val
		}
	}

	hash, err := hashObject(extracted)
	if err != nil {
		return nil, err
	}

	return &PollResult{
		Hash: hash,
		Data: extracted,
	}, nil
}

// hashObject produces a stable hash for arbitrary JSON data
func hashObject(obj map[string]any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
