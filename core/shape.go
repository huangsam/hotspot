package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// iacExtensions are file extensions strongly associated with IaC tooling.
var iacExtensions = map[string]struct{}{
	".tf":      {},
	".tfvars":  {},
	".hcl":     {},
	".tfstate": {},
	".tfplan":  {},
}

// iacBaseNames are filenames that are strong indicators of IaC or infrastructure configuration.
var iacBaseNames = map[string]struct{}{
	"ansible.cfg":         {},
	"site.yml":            {},
	"site.yaml":           {},
	"playbook.yml":        {},
	"playbook.yaml":       {},
	"chart.yaml":          {},
	"values.yaml":         {},
	"dockerfile":          {},
	"containerfile":       {},
	"docker-compose.yml":  {},
	"docker-compose.yaml": {},
	"pulumi.yaml":         {},
	"pulumi.yml":          {},
	"vagrantfile":         {},
	"backend.tf":          {},
	"provider.tf":         {},
	".terraform.lock.hcl": {},
	"cloudformation.json": {},
	"cloudformation.yaml": {},
	"cloudformation.yml":  {},
}

// iacPathPatterns are directory substrings whose YAML/JSON children are likely IaC.
var iacPathPatterns = []string{
	// Tool-specific patterns
	"terraform/", "ansible/", "helm/", "k8s/", "kubernetes/",
	"kustomize/", "playbooks/", "roles/", "charts/",
	"manifests/", "deploy/", "deployments/", "kube/",
	"group_vars/", "host_vars/", "inventory/", "molecule/", "vars/",
	// Generic infrastructure patterns
	"infra/", "infrastructure/", "ops/", "provision/", "provisioning/",
	"setup/", "env/", "environments/",
}

// isIaCFile returns true when the path is likely an infrastructure-as-code file.
func isIaCFile(path string) bool {
	lower := strings.ToLower(path)
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	// 1. Strong indicators: Extensions
	if _, ok := iacExtensions[ext]; ok {
		return true
	}

	// 2. Strong indicators: Specific filenames
	if _, ok := iacBaseNames[base]; ok {
		return true
	}

	// 3. Container variants (e.g. api.dockerfile, web.containerfile)
	if strings.HasSuffix(base, ".dockerfile") || strings.HasSuffix(base, ".containerfile") {
		return true
	}

	// 4. YAML/JSON files inside well-known IaC or infrastructure directories
	if ext == ".yml" || ext == ".yaml" || ext == ".json" {
		for _, pattern := range iacPathPatterns {
			if strings.Contains(lower, pattern) {
				return true
			}
		}

		// 5. Generic suffixes for YAML/JSON files (e.g., app-deployment.yaml)
		noExt := strings.TrimSuffix(base, ext)
		if strings.HasSuffix(noExt, "-deployment") ||
			strings.HasSuffix(noExt, "-stack") ||
			strings.HasSuffix(noExt, "-provision") {
			return true
		}
	}

	return false
}

// recommendPreset selects the best preset based on key shape metrics.
func recommendPreset(fileCount, uniqueContributors int, iacFileRatio float64) schema.PresetName {
	if iacFileRatio >= 0.25 {
		return schema.PresetInfra
	}
	if fileCount > 300 || uniqueContributors > 20 {
		return schema.PresetLarge
	}
	return schema.PresetSmall
}

// ComputeRepoShape derives shape metrics from a file list and aggregate output.
func ComputeRepoShape(files []string, output *schema.AggregateOutput) schema.RepoShape {
	fileCount := len(files)

	// Total commits across all active files
	// Unique contributors across all files
	// Total churn for average calculation
	var totalCommits float64
	var totalChurn float64
	allContribs := make(map[string]struct{})
	activeFiles := 0

	for _, stat := range output.FileStats {
		totalCommits += float64(stat.Commits)
		totalChurn += float64(stat.Churn)
		activeFiles++

		for author := range stat.Contributors {
			allContribs[author] = struct{}{}
		}
	}

	avgChurnPerFile := 0.0
	if activeFiles > 0 {
		avgChurnPerFile = totalChurn / float64(activeFiles)
	}

	// IaC file ratio based on current HEAD file list
	iacCount := 0
	for _, f := range files {
		if isIaCFile(f) {
			iacCount++
		}
	}
	iacFileRatio := 0.0
	if fileCount > 0 {
		iacFileRatio = float64(iacCount) / float64(fileCount)
	}

	preset := recommendPreset(fileCount, len(allContribs), iacFileRatio)

	return schema.RepoShape{
		FileCount:          fileCount,
		TotalCommits:       totalCommits,
		UniqueContributors: len(allContribs),
		AvgChurnPerFile:    avgChurnPerFile,
		IaCFileRatio:       iacFileRatio,
		RecommendedPreset:  preset,
		Preset:             schema.GetPreset(preset),
		AnalyzedAt:         time.Now().UTC(),
	}
}

// GetHotspotShapeResults runs an aggregation pass and computes the repo shape.
func GetHotspotShapeResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (schema.RepoShape, time.Duration, error) {
	start := time.Now()

	files, err := client.ListFilesAtRef(ctx, cfg.Git.RepoPath, "HEAD")
	if err != nil {
		return schema.RepoShape{}, 0, fmt.Errorf("failed to list files: %w", err)
	}

	urn := git.ResolveURN(ctx, client, cfg.Git.RepoPath)
	output, err := agg.CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, client, mgr, urn)
	if err != nil {
		return schema.RepoShape{}, 0, fmt.Errorf("aggregation failed: %w", err)
	}

	shape := ComputeRepoShape(files, output)
	return shape, time.Since(start), nil
}

// ExecuteHotspotShape runs shape analysis and writes the result.
// It prints the full shape metrics as JSON to stdout.
func ExecuteHotspotShape(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) error {
	shape, duration, err := GetHotspotShapeResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Shape analysis complete in %s", duration))

	data, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shape: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
