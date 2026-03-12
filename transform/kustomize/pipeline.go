package kustomize

import (
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	// DefaultKubernetesPriority is the default priority for the built-in Kubernetes plugin
	DefaultKubernetesPriority = 10
	// DefaultPluginPriorityStep is the step between plugin priorities
	DefaultPluginPriorityStep = 10
	// KubernetesPluginName is the name of the built-in Kubernetes sanitization plugin
	KubernetesPluginName = "kubernetes"
)

// PluginMetadata is a minimal interface to avoid import cycle with transform package
type PluginMetadata interface {
	GetName() string
}

// BuildPipeline creates a pipeline from plugin names with automatic priority assignment
func BuildPipeline(pluginNames []string, priorityOverrides map[string]int, commentOverrides map[string]string) (*Pipeline, error) {
	if len(pluginNames) == 0 {
		return nil, fmt.Errorf("no plugins provided")
	}

	stages := []Stage{}
	assignedPriorities := make(map[string]int)

	// First pass: assign priorities
	for _, pluginName := range pluginNames {
		priority := DefaultKubernetesPriority + (len(assignedPriorities) * DefaultPluginPriorityStep)

		// Check for override
		if overridePriority, ok := priorityOverrides[pluginName]; ok {
			priority = overridePriority
		} else if strings.ToLower(pluginName) == KubernetesPluginName {
			// Kubernetes plugin always gets default priority
			priority = DefaultKubernetesPriority
		}

		assignedPriorities[pluginName] = priority
	}

	// Second pass: create stages
	for _, pluginName := range pluginNames {
		priority := assignedPriorities[pluginName]

		comment := ""
		if overrideComment, ok := commentOverrides[pluginName]; ok {
			comment = overrideComment
		}

		stageID := GenerateStageID(priority, pluginName, comment)

		stage := Stage{
			ID:       stageID,
			Plugin:   pluginName,
			Priority: priority,
			Required: strings.ToLower(pluginName) == KubernetesPluginName,
			Enabled:  true,
			Comment:  comment,
		}

		stages = append(stages, stage)
	}

	// Sort stages by priority, then by plugin name
	sort.Slice(stages, func(i, j int) bool {
		if stages[i].Priority != stages[j].Priority {
			return stages[i].Priority < stages[j].Priority
		}
		return stages[i].Plugin < stages[j].Plugin
	})

	return &Pipeline{Stages: stages}, nil
}

// GenerateStageID creates a deterministic stage ID
// Format: <priority>_<pluginName>[:<comment>]
func GenerateStageID(priority int, pluginName, comment string) string {
	sanitizedPlugin := sanitizeForFilename(pluginName)
	stageID := fmt.Sprintf("%d_%s", priority, sanitizedPlugin)

	if comment != "" {
		sanitizedComment := sanitizeForFilename(comment)
		stageID = fmt.Sprintf("%s:%s", stageID, sanitizedComment)
	}

	return stageID
}

// GetStageDir returns the directory path for a stage
func (s Stage) GetStageDir(transformDir string) string {
	return fmt.Sprintf("%s/stages/%s", transformDir, s.ID)
}

// GetRenderedPath returns the path to the rendered.yaml for this stage
func (s Stage) GetRenderedPath(transformDir string) string {
	return fmt.Sprintf("%s/rendered.yaml", s.GetStageDir(transformDir))
}

// SerializePipeline converts pipeline to YAML
func SerializePipeline(pipeline *Pipeline) ([]byte, error) {
	return yaml.Marshal(pipeline)
}

// DeserializePipeline loads pipeline from YAML
func DeserializePipeline(data []byte) (*Pipeline, error) {
	var pipeline Pipeline
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline: %w", err)
	}
	return &pipeline, nil
}

// FilterStages returns stages matching the selection criteria
func (p *Pipeline) FilterStages(stageIDs []string) ([]Stage, error) {
	if len(stageIDs) == 0 {
		return p.Stages, nil
	}

	filtered := []Stage{}
	stageMap := make(map[string]Stage)

	for _, stage := range p.Stages {
		stageMap[stage.ID] = stage
	}

	for _, id := range stageIDs {
		stage, ok := stageMap[id]
		if !ok {
			return nil, fmt.Errorf("stage not found: %s", id)
		}
		filtered = append(filtered, stage)
	}

	return filtered, nil
}

// GetStageByID returns a stage by its ID
func (p *Pipeline) GetStageByID(id string) (*Stage, error) {
	for _, stage := range p.Stages {
		if stage.ID == id {
			return &stage, nil
		}
	}
	return nil, fmt.Errorf("stage not found: %s", id)
}

// ListStages returns a formatted list of stages for display
func (p *Pipeline) ListStages() string {
	var sb strings.Builder
	sb.WriteString("Pipeline Stages:\n")
	sb.WriteString("================\n\n")

	for i, stage := range p.Stages {
		sb.WriteString(fmt.Sprintf("%d. Stage ID: %s\n", i+1, stage.ID))
		sb.WriteString(fmt.Sprintf("   Plugin: %s\n", stage.Plugin))
		sb.WriteString(fmt.Sprintf("   Priority: %d\n", stage.Priority))
		sb.WriteString(fmt.Sprintf("   Required: %t\n", stage.Required))
		sb.WriteString(fmt.Sprintf("   Enabled: %t\n", stage.Enabled))
		if stage.Comment != "" {
			sb.WriteString(fmt.Sprintf("   Comment: %s\n", stage.Comment))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
