package kustomize

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DeriveTarget extracts PatchTarget metadata from an unstructured object
func DeriveTarget(obj unstructured.Unstructured) (PatchTarget, error) {
	target := PatchTarget{}

	// Extract APIVersion
	apiVersion := obj.GetAPIVersion()
	if apiVersion == "" {
		return target, fmt.Errorf("missing apiVersion in resource")
	}

	// Parse group and version from apiVersion
	group, version := parseAPIVersion(apiVersion)
	target.Group = group
	target.Version = version

	// Extract Kind
	target.Kind = obj.GetKind()
	if target.Kind == "" {
		return target, fmt.Errorf("missing kind in resource")
	}

	// Extract Name
	target.Name = obj.GetName()
	if target.Name == "" {
		return target, fmt.Errorf("missing metadata.name in resource")
	}

	// Extract Namespace (optional)
	target.Namespace = obj.GetNamespace()

	return target, nil
}

// parseAPIVersion splits apiVersion into group and version
// Examples:
//   - "v1" -> ("", "v1")
//   - "apps/v1" -> ("apps", "v1")
//   - "route.openshift.io/v1" -> ("route.openshift.io", "v1")
func parseAPIVersion(apiVersion string) (group, version string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 1 {
		// Core group (e.g., "v1")
		return "", parts[0]
	}
	// Non-core group (e.g., "apps/v1")
	return parts[0], parts[1]
}

// ToTargetConfig converts PatchTarget to TargetConfig for kustomization.yaml
func (t PatchTarget) ToTargetConfig() TargetConfig {
	return TargetConfig{
		Group:     t.Group,
		Version:   t.Version,
		Kind:      t.Kind,
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}

// ToResourceIdentity converts PatchTarget to ResourceIdentity for reports
func (t PatchTarget) ToResourceIdentity() ResourceIdentity {
	apiVersion := t.Version
	if t.Group != "" {
		apiVersion = t.Group + "/" + t.Version
	}
	return ResourceIdentity{
		APIVersion: apiVersion,
		Kind:       t.Kind,
		Name:       t.Name,
		Namespace:  t.Namespace,
	}
}
