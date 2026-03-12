package kustomize

import (
	"testing"
)

func TestBuildPipeline(t *testing.T) {
	tests := []struct {
		name              string
		pluginNames       []string
		priorityOverrides map[string]int
		commentOverrides  map[string]string
		wantStageCount    int
		wantFirstStageID  string
		wantErr           bool
	}{
		{
			name:           "single kubernetes plugin",
			pluginNames:    []string{"KubernetesPlugin"},
			wantStageCount: 1,
			wantFirstStageID: "10_KubernetesPlugin",
			wantErr:        false,
		},
		{
			name:           "multiple plugins automatic priority",
			pluginNames:    []string{"KubernetesPlugin", "OpenShiftPlugin", "ImageStreamPlugin"},
			wantStageCount: 3,
			wantErr:        false,
		},
		{
			name:              "with priority overrides",
			pluginNames:       []string{"PluginA", "PluginB"},
			priorityOverrides: map[string]int{"PluginA": 5, "PluginB": 15},
			wantStageCount:    2,
			wantFirstStageID:  "5_PluginA",
			wantErr:           false,
		},
		{
			name:             "with comments",
			pluginNames:      []string{"TestPlugin"},
			commentOverrides: map[string]string{"TestPlugin": "test_comment"},
			wantStageCount:   1,
			wantFirstStageID: "10_TestPlugin:test_comment",
			wantErr:          false,
		},
		{
			name:        "empty plugins",
			pluginNames: []string{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := BuildPipeline(tt.pluginNames, tt.priorityOverrides, tt.commentOverrides)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(pipeline.Stages) != tt.wantStageCount {
				t.Errorf("BuildPipeline() stage count = %d, want %d", len(pipeline.Stages), tt.wantStageCount)
			}

			if tt.wantFirstStageID != "" && len(pipeline.Stages) > 0 {
				if pipeline.Stages[0].ID != tt.wantFirstStageID {
					t.Errorf("BuildPipeline() first stage ID = %s, want %s", pipeline.Stages[0].ID, tt.wantFirstStageID)
				}
			}
		})
	}
}

func TestGenerateStageID(t *testing.T) {
	tests := []struct {
		priority   int
		pluginName string
		comment    string
		want       string
	}{
		{10, "KubernetesPlugin", "", "10_KubernetesPlugin"},
		{20, "OpenShiftPlugin", "route_fix", "20_OpenShiftPlugin:route_fix"},
		{30, "Plugin.With.Dots", "", "30_Plugin-With-Dots"},
		{5, "test", "my comment", "5_test:my-comment"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GenerateStageID(tt.priority, tt.pluginName, tt.comment)
			if got != tt.want {
				t.Errorf("GenerateStageID() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPipelineSorting(t *testing.T) {
	pluginNames := []string{"PluginC", "PluginB", "PluginA", "KubernetesPlugin"}
	priorityOverrides := map[string]int{
		"PluginA": 30,
		"PluginB": 20,
		"PluginC": 25,
		"KubernetesPlugin": 10,
	}

	pipeline, err := BuildPipeline(pluginNames, priorityOverrides, nil)
	if err != nil {
		t.Fatalf("BuildPipeline() error = %v", err)
	}

	// Should be sorted by priority: 10, 20, 25, 30
	expectedOrder := []string{"KubernetesPlugin", "PluginB", "PluginC", "PluginA"}
	for i, stage := range pipeline.Stages {
		if stage.Plugin != expectedOrder[i] {
			t.Errorf("Stage %d: got plugin %s, want %s", i, stage.Plugin, expectedOrder[i])
		}
	}
}

func TestSerializeDeserializePipeline(t *testing.T) {
	pluginNames := []string{"KubernetesPlugin", "TestPlugin"}
	originalPipeline, err := BuildPipeline(pluginNames, nil, nil)
	if err != nil {
		t.Fatalf("BuildPipeline() error = %v", err)
	}

	// Serialize
	yamlBytes, err := SerializePipeline(originalPipeline)
	if err != nil {
		t.Fatalf("SerializePipeline() error = %v", err)
	}

	// Deserialize
	deserializedPipeline, err := DeserializePipeline(yamlBytes)
	if err != nil {
		t.Fatalf("DeserializePipeline() error = %v", err)
	}

	// Compare
	if len(deserializedPipeline.Stages) != len(originalPipeline.Stages) {
		t.Errorf("Stage count mismatch: got %d, want %d", len(deserializedPipeline.Stages), len(originalPipeline.Stages))
	}

	for i := range originalPipeline.Stages {
		if deserializedPipeline.Stages[i].ID != originalPipeline.Stages[i].ID {
			t.Errorf("Stage %d ID mismatch: got %s, want %s", i, deserializedPipeline.Stages[i].ID, originalPipeline.Stages[i].ID)
		}
	}
}

func TestFilterStages(t *testing.T) {
	pluginNames := []string{"PluginA", "PluginB", "PluginC"}
	pipeline, err := BuildPipeline(pluginNames, nil, nil)
	if err != nil {
		t.Fatalf("BuildPipeline() error = %v", err)
	}

	tests := []struct {
		name     string
		stageIDs []string
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "filter none (all stages)",
			stageIDs: []string{},
			wantLen:  3,
			wantErr:  false,
		},
		{
			name:     "filter single stage",
			stageIDs: []string{pipeline.Stages[0].ID},
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "filter multiple stages",
			stageIDs: []string{pipeline.Stages[0].ID, pipeline.Stages[2].ID},
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "filter non-existent stage",
			stageIDs: []string{"99_NonExistent"},
			wantLen:  0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := pipeline.FilterStages(tt.stageIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("FilterStages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(filtered) != tt.wantLen {
				t.Errorf("FilterStages() returned %d stages, want %d", len(filtered), tt.wantLen)
			}
		})
	}
}

func TestListStages(t *testing.T) {
	pluginNames := []string{"KubernetesPlugin", "TestPlugin"}
	commentOverrides := map[string]string{"TestPlugin": "test_comment"}

	pipeline, err := BuildPipeline(pluginNames, nil, commentOverrides)
	if err != nil {
		t.Fatalf("BuildPipeline() error = %v", err)
	}

	output := pipeline.ListStages()

	// Check that output contains expected information
	expectedStrings := []string{
		"Pipeline Stages:",
		"10_KubernetesPlugin",
		"TestPlugin",
		"test_comment",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("ListStages() output missing expected string: %s", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && (len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
