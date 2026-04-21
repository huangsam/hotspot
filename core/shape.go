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
	".tf":      {}, // Terraform
	".tfvars":  {}, // Terraform
	".hcl":     {}, // Terraform
	".tfstate": {}, // Terraform
	".tfplan":  {}, // Terraform
	".pp":      {}, // Puppet
	".bicep":   {}, // Azure Bicep
	".jinja":   {}, // Jinja templates (GCP/Ansible)
}

// iacBaseNames are filenames that are strong indicators of IaC or infrastructure configuration.
var iacBaseNames = map[string]struct{}{
	"ansible.cfg":         {}, // Ansible
	"site.yml":            {}, // Ansible
	"site.yaml":           {}, // Ansible
	"playbook.yml":        {}, // Ansible
	"playbook.yaml":       {}, // Ansible
	"chart.yaml":          {}, // Helm
	"values.yaml":         {}, // Helm
	"dockerfile":          {}, // Docker
	"containerfile":       {}, // Docker
	"docker-compose.yml":  {}, // Docker
	"docker-compose.yaml": {}, // Docker
	"pulumi.yaml":         {}, // Pulumi
	"pulumi.yml":          {}, // Pulumi
	"vagrantfile":         {}, // Vagrant
	"backend.tf":          {}, // Terraform
	"provider.tf":         {}, // Terraform
	".terraform.lock.hcl": {}, // Terraform
	"cloudformation.json": {}, // CloudFormation
	"cloudformation.yaml": {}, // CloudFormation
	"cloudformation.yml":  {}, // CloudFormation
	"cheffile":            {}, // Chef
	"berksfile":           {}, // Chef
	"puppetfile":          {}, // Puppet
	"hiera.yaml":          {}, // Puppet
	"helmfile.yaml":       {}, // Helm
	"serverless.yml":      {}, // Serverless
	"serverless.yaml":     {}, // Serverless
	"azuredeploy.json":    {}, // Azure ARM
	"terragrunt.hcl":      {}, // Terragrunt
	"samconfig.toml":      {}, // AWS SAM
	"main.bicep":          {}, // Azure Bicep
	"packer.json":         {}, // Packer
}

// iacPathPatterns are directory substrings whose YAML/JSON children are likely IaC.
var iacPathPatterns = []string{
	// Tool-specific patterns
	"terraform/", "ansible/", "helm/", "k8s/", "kubernetes/",
	"kustomize/", "playbooks/", "roles/", "charts/",
	"manifests/", "deploy/", "deployments/", "kube/",
	"group_vars/", "host_vars/", "inventory/", "molecule/", "vars/",
	"puppet/", "chef/", "recipes/", "cloudformation/", "cfn/",
	"sam/", "cdk/", "terragrunt/", "packer/", "bicep/", "flux/", "argo/",
	"serverless/", "gitops/",
	// Generic infrastructure patterns
	"infra/", "infrastructure/", "ops/", "provision/", "provisioning/",
	"setup/", "env/", "environments/",
}

// iacStrongSuffixes are multi-part extensions or tool-specific suffixes.
var iacStrongSuffixes = []string{
	".dockerfile", ".containerfile",
	".cf.yml", ".cf.yaml", ".cf.json",
}

// iacContextualExts are extensions that require path context to be considered IaC.
var iacContextualExts = map[string]struct{}{
	".yml":  {},
	".yaml": {},
	".json": {},
	".rb":   {},
	".toml": {},
}

// iacHeuristicSuffixes are generic filename patterns often used for infrastructure.
var iacHeuristicSuffixes = []string{"-deployment", "-stack", "-provision"}

// isIaCFile returns true when the path is likely an infrastructure-as-code file.
func isIaCFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	// 1. Fast Map Matches: Fixed extensions or specific filenames (O(1))
	if _, ok := iacExtensions[ext]; ok {
		return true
	}
	if _, ok := iacBaseNames[base]; ok {
		return true
	}

	// 2. Suffix Matches: Known tool suffixes (O(N) small loop)
	for _, s := range iacStrongSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}

	// 3. Path-dependent matches: Require path context
	if _, ok := iacContextualExts[ext]; ok {
		lowerPath := strings.ToLower(path)
		for _, pattern := range iacPathPatterns {
			if strings.Contains(lowerPath, pattern) {
				// Special Case: Ruby files are ONLY infra if in chef/ or recipes/
				if ext == ".rb" && (!strings.Contains(lowerPath, "chef/") && !strings.Contains(lowerPath, "recipes/")) {
					continue
				}
				return true
			}
		}

		// 4. Heuristic matches: Generic patterns (only for YAML/JSON)
		if ext != ".rb" && ext != ".toml" {
			noExt := strings.TrimSuffix(base, ext)
			for _, s := range iacHeuristicSuffixes {
				if strings.HasSuffix(noExt, s) {
					return true
				}
			}
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

	// FIX: Apply PathFilter so subdirectory shape analysis only looks at relevant files
	if cfg.Git.PathFilter != "" {
		var filtered []string
		for _, f := range files {
			if strings.HasPrefix(f, cfg.Git.PathFilter) {
				filtered = append(filtered, f)
			}
		}
		files = filtered
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
