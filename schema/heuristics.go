package schema

import (
	_ "embed"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/shape_heuristics.yaml
var shapeHeuristicsRaw []byte

type shapeHeuristics struct {
	Thresholds struct {
		IaCRatio         float64 `yaml:"iac_ratio"`
		FileCount        int     `yaml:"file_count"`
		ContributorCount int     `yaml:"contributor_count"`
	} `yaml:"thresholds"`
	Templates struct {
		Infra             string `yaml:"infra"`
		LargeFiles        string `yaml:"large_files"`
		LargeContributors string `yaml:"large_contributors"`
		Compounding       string `yaml:"compounding"`
		Small             string `yaml:"small"`
	} `yaml:"templates"`
	IaC struct {
		Extensions           []string `yaml:"extensions"`
		Basenames            []string `yaml:"basenames"`
		PathPatterns         []string `yaml:"path_patterns"`
		StrongSuffixes       []string `yaml:"strong_suffixes"`
		ContextualExtensions []string `yaml:"contextual_extensions"`
		HeuristicSuffixes    []string `yaml:"heuristic_suffixes"`
	} `yaml:"iac"`
}

var heuristics shapeHeuristics

// Maps for fast lookup.
var (
	iacExtMap        = make(map[string]struct{})
	iacBaseMap       = make(map[string]struct{})
	iacContextExtMap = make(map[string]struct{})
)

func init() {
	if err := yaml.Unmarshal(shapeHeuristicsRaw, &heuristics); err != nil {
		// This should only fail during development if the YAML is malformed.
		panic("failed to unmarshal shape_heuristics.yaml: " + err.Error())
	}

	for _, e := range heuristics.IaC.Extensions {
		iacExtMap[strings.ToLower(e)] = struct{}{}
	}
	for _, b := range heuristics.IaC.Basenames {
		iacBaseMap[strings.ToLower(b)] = struct{}{}
	}
	for _, e := range heuristics.IaC.ContextualExtensions {
		iacContextExtMap[strings.ToLower(e)] = struct{}{}
	}
}

// IsIaCFile returns true when the path is likely an infrastructure-as-code file.
func IsIaCFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	if _, ok := iacExtMap[ext]; ok {
		return true
	}
	if _, ok := iacBaseMap[base]; ok {
		return true
	}

	for _, s := range heuristics.IaC.StrongSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}

	if _, ok := iacContextExtMap[ext]; ok {
		lowerPath := strings.ToLower(path)
		for _, pattern := range heuristics.IaC.PathPatterns {
			if strings.Contains(lowerPath, pattern) {
				if ext == ".rb" && (!strings.Contains(lowerPath, "chef/") && !strings.Contains(lowerPath, "recipes/")) {
					continue
				}
				return true
			}
		}

		if ext != ".rb" {
			noExt := strings.TrimSuffix(base, ext)
			for _, s := range heuristics.IaC.HeuristicSuffixes {
				if strings.HasSuffix(noExt, s) {
					return true
				}
			}
		}
	}

	return false
}

// ShapeThresholds returns the configuration thresholds for shape analysis.
type ShapeThresholds struct {
	IaCRatio         float64
	FileCount        int
	ContributorCount int
}

// GetShapeThresholds returns the thresholds defined in shape_heuristics.yaml.
func GetShapeThresholds() ShapeThresholds {
	return ShapeThresholds{
		IaCRatio:         heuristics.Thresholds.IaCRatio,
		FileCount:        heuristics.Thresholds.FileCount,
		ContributorCount: heuristics.Thresholds.ContributorCount,
	}
}

// ShapeTemplates returns the reasoning templates for shape analysis.
type ShapeTemplates struct {
	Infra             string
	LargeFiles        string
	LargeContributors string
	Compounding       string
	Small             string
}

// GetShapeTemplates returns the reasoning templates defined in shape_heuristics.yaml.
func GetShapeTemplates() ShapeTemplates {
	return ShapeTemplates{
		Infra:             heuristics.Templates.Infra,
		LargeFiles:        heuristics.Templates.LargeFiles,
		LargeContributors: heuristics.Templates.LargeContributors,
		Compounding:       heuristics.Templates.Compounding,
		Small:             heuristics.Templates.Small,
	}
}
