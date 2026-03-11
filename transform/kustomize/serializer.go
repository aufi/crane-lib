package kustomize

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"sigs.k8s.io/yaml"
)

// SerializePatchToYAML converts a JSONPatch operation list to YAML format
// suitable for Kustomize JSON6902 patches
func SerializePatchToYAML(patch jsonpatch.Patch) ([]byte, error) {
	if len(patch) == 0 {
		return []byte("[]"), nil
	}

	// Convert jsonpatch.Patch to a slice of operation maps
	operations := make([]map[string]interface{}, 0, len(patch))
	for _, op := range patch {
		opMap, err := operationToMap(op)
		if err != nil {
			return nil, fmt.Errorf("failed to convert operation to map: %w", err)
		}
		operations = append(operations, opMap)
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(operations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch to YAML: %w", err)
	}

	return yamlBytes, nil
}

// operationToMap converts a JSONPatch operation to a map for YAML serialization
func operationToMap(op jsonpatch.Operation) (map[string]interface{}, error) {
	opMap := make(map[string]interface{})

	// Get operation kind (add, remove, replace, etc.)
	kind := op.Kind()
	opMap["op"] = kind

	// Get path
	path, err := op.Path()
	if err != nil {
		return nil, fmt.Errorf("failed to get operation path: %w", err)
	}
	opMap["path"] = path

	// Get value if present (not for remove operations)
	if kind != "remove" {
		value, err := op.ValueInterface()
		if err != nil {
			// Some operations (like "test") might not have value
			// Check if it's a missing value error, which is acceptable for some ops
			if !isValueMissingError(err) {
				return nil, fmt.Errorf("failed to get operation value: %w", err)
			}
		} else {
			opMap["value"] = value
		}
	}

	// Handle "from" for move/copy operations
	if kind == "move" || kind == "copy" {
		// Try to extract "from" from the operation
		// The jsonpatch library doesn't expose "from" directly in the interface,
		// so we need to use json.Marshal on the operation
		opBytes, err := json.Marshal(op)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal operation: %w", err)
		}
		var fullOp map[string]interface{}
		if err := json.Unmarshal(opBytes, &fullOp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
		}
		if from, ok := fullOp["from"]; ok {
			opMap["from"] = from
		}
	}

	return opMap, nil
}

// isValueMissingError checks if the error indicates a missing value
func isValueMissingError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "missing value" || errStr == "value is null"
}
