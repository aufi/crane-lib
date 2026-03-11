package kustomize

import (
	"strings"
	"testing"
)

func TestGeneratePatchFileName(t *testing.T) {
	tests := []struct {
		name   string
		target PatchTarget
		want   string
	}{
		{
			name: "namespaced core resource",
			target: PatchTarget{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Name:      "nginx",
				Namespace: "default",
			},
			want: "default--core-v1--Service--nginx.patch.yaml",
		},
		{
			name: "cluster-scoped core resource",
			target: PatchTarget{
				Group:     "",
				Version:   "v1",
				Kind:      "Namespace",
				Name:      "test-ns",
				Namespace: "",
			},
			want: "_cluster--core-v1--Namespace--test-ns.patch.yaml",
		},
		{
			name: "namespaced non-core resource",
			target: PatchTarget{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Name:      "myapp",
				Namespace: "production",
			},
			want: "production--apps-v1--Deployment--myapp.patch.yaml",
		},
		{
			name: "openshift route",
			target: PatchTarget{
				Group:     "route.openshift.io",
				Version:   "v1",
				Kind:      "Route",
				Name:      "frontend",
				Namespace: "myns",
			},
			want: "myns--route-openshift-io-v1--Route--frontend.patch.yaml",
		},
		{
			name: "name with special characters",
			target: PatchTarget{
				Group:     "",
				Version:   "v1",
				Kind:      "ConfigMap",
				Name:      "my@config#map",
				Namespace: "default",
			},
			want: "default--core-v1--ConfigMap--my-config-map.patch.yaml",
		},
		{
			name: "very long name",
			target: PatchTarget{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Name:      strings.Repeat("verylongname", 20),
				Namespace: "namespace",
			},
			// Should be truncated with hash suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePatchFileName(tt.target)
			if tt.want != "" && got != tt.want {
				t.Errorf("GeneratePatchFileName() = %v, want %v", got, tt.want)
			}
			// Check that result is always valid
			if !strings.HasSuffix(got, ".patch.yaml") {
				t.Errorf("GeneratePatchFileName() = %v, does not end with .patch.yaml", got)
			}
			if len(got) > MaxPatchFileNameLength {
				t.Errorf("GeneratePatchFileName() = %v, length %d exceeds max %d", got, len(got), MaxPatchFileNameLength)
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal-name", "normal-name"},
		{"route.openshift.io", "route-openshift-io"},
		{"my@special#chars!", "my-special-chars"},
		{"multiple---dashes", "multiple-dashes"},
		{"-leading-trailing-", "leading-trailing"},
		{"", "unnamed"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeForFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeForFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGeneratePatchFileNameDeterministic(t *testing.T) {
	target := PatchTarget{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Name:      "myapp",
		Namespace: "default",
	}

	name1 := GeneratePatchFileName(target)
	name2 := GeneratePatchFileName(target)

	if name1 != name2 {
		t.Errorf("GeneratePatchFileName is not deterministic: %q != %q", name1, name2)
	}
}
