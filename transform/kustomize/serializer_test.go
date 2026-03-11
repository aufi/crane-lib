package kustomize

import (
	"strings"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	"sigs.k8s.io/yaml"
)

func TestSerializePatchToYAML(t *testing.T) {
	tests := []struct {
		name    string
		patch   string // JSON patch input
		wantErr bool
	}{
		{
			name:    "remove operation",
			patch:   `[{"op":"remove","path":"/spec/clusterIP"}]`,
			wantErr: false,
		},
		{
			name:    "replace operation",
			patch:   `[{"op":"replace","path":"/spec/type","value":"NodePort"}]`,
			wantErr: false,
		},
		{
			name:    "add operation",
			patch:   `[{"op":"add","path":"/metadata/labels/app","value":"myapp"}]`,
			wantErr: false,
		},
		{
			name:    "multiple operations",
			patch:   `[{"op":"remove","path":"/spec/clusterIP"},{"op":"replace","path":"/spec/type","value":"NodePort"}]`,
			wantErr: false,
		},
		{
			name:    "empty patch",
			patch:   `[]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := jsonpatch.DecodePatch([]byte(tt.patch))
			if err != nil {
				t.Fatalf("Failed to decode test patch: %v", err)
			}

			got, err := SerializePatchToYAML(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializePatchToYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify it's valid YAML
				var yamlCheck interface{}
				if err := yaml.Unmarshal(got, &yamlCheck); err != nil {
					t.Errorf("SerializePatchToYAML() produced invalid YAML: %v\nOutput: %s", err, string(got))
				}

				// Verify it contains expected operation type
				if tt.name != "empty patch" && !strings.Contains(string(got), "op:") {
					t.Errorf("SerializePatchToYAML() output missing 'op:' field: %s", string(got))
				}
			}
		})
	}
}

func TestOperationToMap(t *testing.T) {
	tests := []struct {
		name    string
		opJSON  string
		wantOp  string
		wantErr bool
	}{
		{
			name:    "remove operation",
			opJSON:  `{"op":"remove","path":"/spec/field"}`,
			wantOp:  "remove",
			wantErr: false,
		},
		{
			name:    "replace with string value",
			opJSON:  `{"op":"replace","path":"/spec/type","value":"NodePort"}`,
			wantOp:  "replace",
			wantErr: false,
		},
		{
			name:    "add with object value",
			opJSON:  `{"op":"add","path":"/metadata/labels","value":{"app":"test"}}`,
			wantOp:  "add",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := jsonpatch.DecodePatch([]byte("[" + tt.opJSON + "]"))
			if err != nil {
				t.Fatalf("Failed to decode test operation: %v", err)
			}

			got, err := operationToMap(patch[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("operationToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if op, ok := got["op"].(string); !ok || op != tt.wantOp {
					t.Errorf("operationToMap() op = %v, want %v", op, tt.wantOp)
				}
				if _, ok := got["path"]; !ok {
					t.Errorf("operationToMap() missing 'path' field")
				}
			}
		})
	}
}
