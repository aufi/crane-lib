package transform

import (
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform/kustomize"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type mockPlugin struct {
	name       string
	patches    jsonpatch.Patch
	isWhiteOut bool
}

func (m *mockPlugin) Run(req PluginRequest) (PluginResponse, error) {
	return PluginResponse{
		Version:    "v1",
		IsWhiteOut: m.isWhiteOut,
		Patches:    m.patches,
	}, nil
}

func (m *mockPlugin) Metadata() PluginMetadata {
	return PluginMetadata{
		Name:    m.name,
		Version: "v1",
	}
}

func TestRunForKustomize(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	t.Run("basic patch generation", func(t *testing.T) {
		obj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name":      "test-svc",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"clusterIP": "10.96.0.1",
				},
			},
		}

		patch, _ := jsonpatch.DecodePatch([]byte(`[{"op":"remove","path":"/spec/clusterIP"}]`))
		plugin := &mockPlugin{
			name:    "TestPlugin",
			patches: patch,
		}

		runner := Runner{
			Log:              log,
			PluginPriorities: map[string]int{},
			OptionalFlags:    map[string]string{},
		}

		artifact, err := runner.RunForKustomize(obj, []Plugin{plugin})
		if err != nil {
			t.Fatalf("RunForKustomize() error = %v", err)
		}

		if artifact.Target.Kind != "Service" {
			t.Errorf("Expected Kind=Service, got %s", artifact.Target.Kind)
		}

		if artifact.Target.Name != "test-svc" {
			t.Errorf("Expected Name=test-svc, got %s", artifact.Target.Name)
		}

		if len(artifact.Patches) != 1 {
			t.Errorf("Expected 1 patch, got %d", len(artifact.Patches))
		}

		if artifact.IsWhiteOut {
			t.Error("Expected IsWhiteOut=false")
		}
	})

	t.Run("whiteout handling", func(t *testing.T) {
		obj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      "test-cm",
					"namespace": "default",
				},
			},
		}

		plugin := &mockPlugin{
			name:       "WhiteOutPlugin",
			isWhiteOut: true,
		}

		runner := Runner{
			Log:              log,
			PluginPriorities: map[string]int{},
			OptionalFlags:    map[string]string{},
		}

		artifact, err := runner.RunForKustomize(obj, []Plugin{plugin})
		if err != nil {
			t.Fatalf("RunForKustomize() error = %v", err)
		}

		if !artifact.IsWhiteOut {
			t.Error("Expected IsWhiteOut=true")
		}

		if len(artifact.WhiteOutRequestedBy) != 1 || artifact.WhiteOutRequestedBy[0] != "WhiteOutPlugin" {
			t.Errorf("Expected WhiteOutRequestedBy=[WhiteOutPlugin], got %v", artifact.WhiteOutRequestedBy)
		}

		if len(artifact.Patches) != 0 {
			t.Errorf("Expected 0 patches for whiteout, got %d", len(artifact.Patches))
		}
	})

	t.Run("conflict resolution with priorities", func(t *testing.T) {
		obj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name":      "test-svc",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"type": "ClusterIP",
				},
			},
		}

		patch1, _ := jsonpatch.DecodePatch([]byte(`[{"op":"replace","path":"/spec/type","value":"NodePort"}]`))
		plugin1 := &mockPlugin{
			name:    "Plugin1",
			patches: patch1,
		}

		patch2, _ := jsonpatch.DecodePatch([]byte(`[{"op":"replace","path":"/spec/type","value":"LoadBalancer"}]`))
		plugin2 := &mockPlugin{
			name:    "Plugin2",
			patches: patch2,
		}

		runner := Runner{
			Log: log,
			PluginPriorities: map[string]int{
				"Plugin1": 1, // Higher priority (lower number)
				"Plugin2": 2,
			},
			OptionalFlags: map[string]string{},
		}

		artifact, err := runner.RunForKustomize(obj, []Plugin{plugin1, plugin2})
		if err != nil {
			t.Fatalf("RunForKustomize() error = %v", err)
		}

		if len(artifact.Patches) != 1 {
			t.Errorf("Expected 1 patch after conflict resolution, got %d", len(artifact.Patches))
		}

		if len(artifact.IgnoredPatches) != 1 {
			t.Errorf("Expected 1 ignored patch, got %d", len(artifact.IgnoredPatches))
		}

		if len(artifact.IgnoredPatches) > 0 {
			ignored := artifact.IgnoredPatches[0]
			if ignored.IgnoredPlugin != "Plugin2" {
				t.Errorf("Expected IgnoredPlugin=Plugin2, got %s", ignored.IgnoredPlugin)
			}
			if ignored.SelectedPlugin != "Plugin1" {
				t.Errorf("Expected SelectedPlugin=Plugin1, got %s", ignored.SelectedPlugin)
			}
		}
	})

	t.Run("target derivation for non-core resource", func(t *testing.T) {
		obj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "test-deploy",
					"namespace": "production",
				},
			},
		}

		plugin := &mockPlugin{
			name: "TestPlugin",
		}

		runner := Runner{
			Log:              log,
			PluginPriorities: map[string]int{},
			OptionalFlags:    map[string]string{},
		}

		artifact, err := runner.RunForKustomize(obj, []Plugin{plugin})
		if err != nil {
			t.Fatalf("RunForKustomize() error = %v", err)
		}

		expected := kustomize.PatchTarget{
			Group:     "apps",
			Version:   "v1",
			Kind:      "Deployment",
			Name:      "test-deploy",
			Namespace: "production",
		}

		if artifact.Target != expected {
			t.Errorf("Expected Target=%+v, got %+v", expected, artifact.Target)
		}
	})
}
