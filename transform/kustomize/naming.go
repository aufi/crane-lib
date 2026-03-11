package kustomize

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxPatchFileNameLength is the maximum length for a patch file name
	// This accounts for filesystem limits and readability
	MaxPatchFileNameLength = 200
	// HashSuffixLength is the length of hash suffix when name is too long
	HashSuffixLength = 8
)

var (
	// invalidFileCharsRegex matches characters that are not safe for filenames
	// Allow only alphanumeric, underscore, and dash (not dot to avoid confusion)
	invalidFileCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	// multiDashRegex matches multiple consecutive dashes
	multiDashRegex = regexp.MustCompile(`-+`)
)

// GeneratePatchFileName creates a deterministic, collision-safe filename for a patch file
// Pattern: <namespace-or-_cluster>--<group-or-core>-<version>--<kind>--<name>.patch.yaml
func GeneratePatchFileName(target PatchTarget) string {
	parts := []string{}

	// Namespace or _cluster
	if target.Namespace != "" {
		parts = append(parts, sanitizeForFilename(target.Namespace))
	} else {
		parts = append(parts, "_cluster")
	}

	// Group (or "core" for core resources)
	group := target.Group
	if group == "" {
		group = "core"
	}
	groupVersion := fmt.Sprintf("%s-%s", sanitizeForFilename(group), sanitizeForFilename(target.Version))
	parts = append(parts, groupVersion)

	// Kind
	parts = append(parts, sanitizeForFilename(target.Kind))

	// Name
	parts = append(parts, sanitizeForFilename(target.Name))

	// Join with -- separator
	baseName := strings.Join(parts, "--")

	// Handle length limits
	fileName := baseName + ".patch.yaml"
	if len(fileName) > MaxPatchFileNameLength {
		// Truncate and add hash suffix for uniqueness
		hash := generateShortHash(baseName)
		maxBaseLen := MaxPatchFileNameLength - len(".patch.yaml") - HashSuffixLength - 1 // -1 for dash
		if maxBaseLen < 0 {
			maxBaseLen = 0
		}
		truncated := baseName
		if len(baseName) > maxBaseLen {
			truncated = baseName[:maxBaseLen]
		}
		fileName = fmt.Sprintf("%s-%s.patch.yaml", truncated, hash)
	}

	return fileName
}

// sanitizeForFilename converts a string to be filesystem-safe
func sanitizeForFilename(s string) string {
	// Replace invalid characters with dash
	sanitized := invalidFileCharsRegex.ReplaceAllString(s, "-")
	// Collapse multiple dashes
	sanitized = multiDashRegex.ReplaceAllString(sanitized, "-")
	// Trim leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")
	// Handle empty result
	if sanitized == "" {
		sanitized = "unnamed"
	}
	return sanitized
}

// generateShortHash creates a short hash of the input string
func generateShortHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash[:4]) // Use first 4 bytes (8 hex chars)
}
