package kustomize

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDeriveTarget(t *testing.T) {
	tests := []struct {
		name    string
		obj     unstructured.Unstructured
		want    PatchTarget
		wantErr bool
	}{
		{
			name: "core resource",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "test-svc",
						"namespace": "default",
					},
				},
			},
			want: PatchTarget{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Name:      "test-svc",
				Namespace: "default",
			},
			wantErr: false,
		},
		{
			name: "non-core resource",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "test-deploy",
						"namespace": "kube-system",
					},
				},
			},
			want: PatchTarget{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Name:      "test-deploy",
				Namespace: "kube-system",
			},
			wantErr: false,
		},
		{
			name: "cluster-scoped resource",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"name": "test-ns",
					},
				},
			},
			want: PatchTarget{
				Group:     "",
				Version:   "v1",
				Kind:      "Namespace",
				Name:      "test-ns",
				Namespace: "",
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Service",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata":   map[string]interface{}{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveTarget(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DeriveTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAPIVersion(t *testing.T) {
	tests := []struct {
		apiVersion  string
		wantGroup   string
		wantVersion string
	}{
		{"v1", "", "v1"},
		{"apps/v1", "apps", "v1"},
		{"route.openshift.io/v1", "route.openshift.io", "v1"},
		{"batch/v1beta1", "batch", "v1beta1"},
	}

	for _, tt := range tests {
		t.Run(tt.apiVersion, func(t *testing.T) {
			gotGroup, gotVersion := parseAPIVersion(tt.apiVersion)
			if gotGroup != tt.wantGroup || gotVersion != tt.wantVersion {
				t.Errorf("parseAPIVersion(%q) = (%q, %q), want (%q, %q)",
					tt.apiVersion, gotGroup, gotVersion, tt.wantGroup, tt.wantVersion)
			}
		})
	}
}
