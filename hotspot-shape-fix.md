# Objective
Improve the `isIaCFile` detection logic in the `hotspot` tool to better recognize Infrastructure-as-Code (IaC) repositories (such as CloudFormation, Helm, Puppet, Chef) and fix a bug where `PathFilter` is not respected during shape analysis.

# Key Files & Context
- `core/shape.go`: Contains the `isIaCFile` logic and the `GetHotspotShapeResults` function which orchestrates the shape analysis.

# Implementation Steps

## 1. Improve IaC Detection (`core/shape.go`)
Update the `iacExtensions`, `iacBaseNames`, and `iacPathPatterns` to include more IaC tools. Also, add suffix matching for CloudFormation templates.

```go
// Add to iacExtensions:
var iacExtensions = map[string]struct{}{
	".tf":      {},
	".tfvars":  {},
	".hcl":     {},
	".tfstate": {},
	".tfplan":  {},
	".pp":      {}, // Puppet
}

// Add to iacBaseNames:
var iacBaseNames = map[string]struct{}{
	// ... existing ...
	"cheffile":            {}, // Chef
	"berksfile":           {}, // Chef
	"puppetfile":          {}, // Puppet
	"hiera.yaml":          {}, // Puppet
	"helmfile.yaml":       {}, // Helm
}

// Add to iacPathPatterns:
var iacPathPatterns = []string{
	// ... existing ...
	"puppet/", "chef/", "recipes/", "cloudformation/", "cfn/",
}
```

In `isIaCFile`, add logic to match common CloudFormation file suffixes:
```go
// 3. Container variants (e.g. api.dockerfile, web.containerfile)
if strings.HasSuffix(base, ".dockerfile") || strings.HasSuffix(base, ".containerfile") {
	return true
}

// 4. CloudFormation templates (e.g., master.cf.yml)
if strings.HasSuffix(base, ".cf.yml") || strings.HasSuffix(base, ".cf.yaml") || strings.HasSuffix(base, ".cf.json") {
	return true
}

// 5. YAML/JSON files inside well-known IaC or infrastructure directories
if ext == ".yml" || ext == ".yaml" || ext == ".json" {
    // ...
```

## 2. Fix PathFilter in Shape Analysis (`core/shape.go`)
In `GetHotspotShapeResults`, filter the `files` slice based on `cfg.Git.PathFilter` before computing the shape metrics.

```go
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
```

# Verification & Testing
Apply these changes in the `hotspot` repository, rebuild the binary, and run `get_repo_shape` on an infrastructure repo like `aws-cf-custom-templates` or a subdirectory with IaC files to ensure the `iac_file_ratio` and `RecommendedPreset` compute correctly.
