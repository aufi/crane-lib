package kustomize

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// GenerateKustomizationWithPaths creates a kustomization.yaml with explicit resource paths
func GenerateKustomizationWithPaths(artifacts []TransformArtifact, resourcePaths map[string]string) ([]byte, error) {
	kustomization := KustomizationFile{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{},
		Patches:    []PatchConfiguration{},
	}

	// Collect resources and patches
	resourceMap := make(map[string]bool)
	patchList := []PatchConfiguration{}

	for _, artifact := range artifacts {
		// Skip whiteouted resources
		if artifact.IsWhiteOut {
			continue
		}

		// Build key for resource path lookup
		key := fmt.Sprintf("%s/%s/%s/%s", artifact.Target.Namespace, artifact.Target.Group, artifact.Target.Kind, artifact.Target.Name)
		resourcePath, ok := resourcePaths[key]
		if !ok {
			// Fallback to building path
			resourcePath = fmt.Sprintf("../export/%s/%s_%s_%s_%s.yaml",
				artifact.Target.Namespace,
				artifact.Target.Group,
				artifact.Target.Version,
				artifact.Target.Kind,
				artifact.Target.Name)
		}

		// Add to resource list (deduplicate)
		if !resourceMap[resourcePath] {
			resourceMap[resourcePath] = true
			kustomization.Resources = append(kustomization.Resources, resourcePath)
		}

		// Add patch if there are operations
		if len(artifact.Patches) > 0 {
			patchFileName := GeneratePatchFileName(artifact.Target)
			patchConfig := PatchConfiguration{
				Path:   filepath.Join("patches", patchFileName),
				Target: artifact.Target.ToTargetConfig(),
			}
			patchList = append(patchList, patchConfig)
		}
	}

	// Sort for deterministic output
	sort.Strings(kustomization.Resources)
	sort.Slice(patchList, func(i, j int) bool {
		return patchList[i].Path < patchList[j].Path
	})
	kustomization.Patches = patchList

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kustomization to YAML: %w", err)
	}

	return yamlBytes, nil
}

// GenerateKustomization creates a kustomization.yaml content from transform artifacts
// Deprecated: Use GenerateKustomizationWithPaths instead
func GenerateKustomization(artifacts []TransformArtifact, exportDir string) ([]byte, error) {
	kustomization := KustomizationFile{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{},
		Patches:    []PatchConfiguration{},
	}

	// Collect resources and patches
	resourceMap := make(map[string]bool)
	patchList := []PatchConfiguration{}

	for _, artifact := range artifacts {
		// Skip whiteouted resources
		if artifact.IsWhiteOut {
			continue
		}

		// Build resource path relative to transform dir
		resourcePath, err := buildResourcePath(artifact.Target, exportDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build resource path: %w", err)
		}

		// Add to resource list (deduplicate)
		if !resourceMap[resourcePath] {
			resourceMap[resourcePath] = true
			kustomization.Resources = append(kustomization.Resources, resourcePath)
		}

		// Add patch if there are operations
		if len(artifact.Patches) > 0 {
			patchFileName := GeneratePatchFileName(artifact.Target)
			patchConfig := PatchConfiguration{
				Path:   filepath.Join("patches", patchFileName),
				Target: artifact.Target.ToTargetConfig(),
			}
			patchList = append(patchList, patchConfig)
		}
	}

	// Sort for deterministic output
	sort.Strings(kustomization.Resources)
	sort.Slice(patchList, func(i, j int) bool {
		return patchList[i].Path < patchList[j].Path
	})
	kustomization.Patches = patchList

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kustomization to YAML: %w", err)
	}

	return yamlBytes, nil
}

// buildResourcePath constructs the relative path from transform dir to export resource file
// The path format follows the export directory structure
func buildResourcePath(target PatchTarget, exportDir string) (string, error) {
	// Build filename based on resource identity
	// Format: <group>_<version>_<kind>_<name>.yaml
	// This matches the crane export format which uses lowercase for group
	group := target.Group
	if group == "" {
		group = "v1"
	} else {
		group = fmt.Sprintf("%s_%s", group, target.Version)
	}

	// Build actual filename as it appears in export
	var fileName string
	if target.Group == "" {
		// Core resources: v1_<kind>_<name>.yaml
		fileName = fmt.Sprintf("v1_%s_%s.yaml",
			strings.ToLower(target.Kind),
			target.Name,
		)
	} else {
		// Non-core: <group>_<version>_<kind>_<name>.yaml
		fileName = fmt.Sprintf("%s_v1_%s_%s.yaml",
			target.Group,
			strings.ToLower(target.Kind),
			target.Name,
		)
	}

	// Build path with namespace if present
	var resourcePath string
	if target.Namespace != "" {
		resourcePath = filepath.Join("..", filepath.Base(exportDir), target.Namespace, fileName)
	} else {
		resourcePath = filepath.Join("..", filepath.Base(exportDir), "_cluster", fileName)
	}

	return resourcePath, nil
}

// GenerateWhiteOutReport creates JSON report of whiteouted resources
func GenerateWhiteOutReport(artifacts []TransformArtifact) ([]byte, error) {
	reports := []WhiteOutReport{}

	for _, artifact := range artifacts {
		if !artifact.IsWhiteOut {
			continue
		}

		identity := artifact.Target.ToResourceIdentity()
		report := WhiteOutReport{
			APIVersion:  identity.APIVersion,
			Kind:        identity.Kind,
			Name:        identity.Name,
			Namespace:   identity.Namespace,
			RequestedBy: artifact.WhiteOutRequestedBy,
		}
		reports = append(reports, report)
	}

	// Sort for deterministic output
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Namespace != reports[j].Namespace {
			return reports[i].Namespace < reports[j].Namespace
		}
		if reports[i].Kind != reports[j].Kind {
			return reports[i].Kind < reports[j].Kind
		}
		return reports[i].Name < reports[j].Name
	})

	if len(reports) == 0 {
		return nil, nil
	}

	jsonBytes, err := yaml.Marshal(reports)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal whiteout report: %w", err)
	}

	return jsonBytes, nil
}

// GenerateIgnoredPatchesReport creates JSON report of ignored patches
func GenerateIgnoredPatchesReport(artifacts []TransformArtifact) ([]byte, error) {
	reports := []IgnoredPatchReport{}

	for _, artifact := range artifacts {
		for _, ignored := range artifact.IgnoredPatches {
			identity := artifact.Target.ToResourceIdentity()
			report := IgnoredPatchReport{
				Resource:       identity,
				Path:           ignored.Path,
				SelectedPlugin: ignored.SelectedPlugin,
				IgnoredPlugin:  ignored.IgnoredPlugin,
				Reason:         ignored.Reason,
			}
			reports = append(reports, report)
		}
	}

	// Sort for deterministic output
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Resource.Namespace != reports[j].Resource.Namespace {
			return reports[i].Resource.Namespace < reports[j].Resource.Namespace
		}
		if reports[i].Resource.Kind != reports[j].Resource.Kind {
			return reports[i].Resource.Kind < reports[j].Resource.Kind
		}
		if reports[i].Resource.Name != reports[j].Resource.Name {
			return reports[i].Resource.Name < reports[j].Resource.Name
		}
		return reports[i].Path < reports[j].Path
	})

	if len(reports) == 0 {
		return nil, nil
	}

	jsonBytes, err := yaml.Marshal(reports)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ignored patches report: %w", err)
	}

	return jsonBytes, nil
}
