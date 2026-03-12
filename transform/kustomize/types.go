package kustomize

import (
	jsonpatch "github.com/evanphx/json-patch"
)

// TransformArtifact represents the structured output for a single resource transformation
type TransformArtifact struct {
	// Target identifies the resource being transformed
	Target PatchTarget
	// Patches contains the sanitized JSONPatch operations
	Patches jsonpatch.Patch
	// IgnoredPatches contains operations that were discarded due to conflicts
	IgnoredPatches []IgnoredPatch
	// IsWhiteOut indicates if this resource should be excluded from output
	IsWhiteOut bool
	// WhiteOutRequestedBy tracks which plugins requested whiteout
	WhiteOutRequestedBy []string
}

// PatchTarget contains metadata for Kustomize target selector
type PatchTarget struct {
	Group     string
	Version   string
	Kind      string
	Name      string
	Namespace string
}

// IgnoredPatch represents a patch operation that was discarded due to conflict
type IgnoredPatch struct {
	Path           string
	SelectedPlugin string
	IgnoredPlugin  string
	Reason         string
	Operation      jsonpatch.Operation
}

// WhiteOutReport represents a resource that was whiteouted
type WhiteOutReport struct {
	APIVersion  string   `json:"apiVersion"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace,omitempty"`
	RequestedBy []string `json:"requestedBy,omitempty"`
}

// IgnoredPatchReport represents a patch that was ignored due to conflict
type IgnoredPatchReport struct {
	Resource       ResourceIdentity `json:"resource"`
	Path           string           `json:"path"`
	SelectedPlugin string           `json:"selectedPlugin"`
	IgnoredPlugin  string           `json:"ignoredPlugin"`
	Reason         string           `json:"reason"`
}

// ResourceIdentity identifies a Kubernetes resource
type ResourceIdentity struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
}

// KustomizationFile represents the structure of kustomization.yaml
type KustomizationFile struct {
	APIVersion string              `yaml:"apiVersion,omitempty"`
	Kind       string              `yaml:"kind,omitempty"`
	Resources  []string            `yaml:"resources,omitempty"`
	Patches    []PatchConfiguration `yaml:"patches,omitempty"`
}

// PatchConfiguration represents a patch entry in kustomization.yaml
type PatchConfiguration struct {
	Path   string       `yaml:"path"`
	Target TargetConfig `yaml:"target"`
}

// TargetConfig represents the target selector for a patch
type TargetConfig struct {
	Group     string `yaml:"group,omitempty"`
	Version   string `yaml:"version"`
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

// Pipeline types for multi-stage execution

// Pipeline represents the complete transform pipeline
type Pipeline struct {
	Stages []Stage `yaml:"stages"`
}

// Stage represents one plugin execution stage in the pipeline
type Stage struct {
	ID       string `yaml:"id"`
	Plugin   string `yaml:"plugin"`
	Priority int    `yaml:"priority"`
	Required bool   `yaml:"required"`
	Enabled  bool   `yaml:"enabled"`
	Comment  string `yaml:"comment,omitempty"`
}

// StageArtifacts contains all outputs for a single stage
type StageArtifacts struct {
	Stage              Stage
	ResourceArtifacts  []TransformArtifact
	RenderedManifests  []byte // Output of kubectl kustomize for this stage
}
