package core

import (
	"strconv"
	"strings"
	"testing"

	"github.com/huangsam/hotspot/schema"
)

// FuzzComputeScore fuzzes the computeScore function with random FileResult inputs.
func FuzzComputeScore(f *testing.F) {
	seeds := []struct {
		result schema.FileResult
		mode   string
	}{
		{
			result: schema.FileResult{
				Path:               "main.go",
				SizeBytes:          1000,
				LinesOfCode:        50,
				Commits:            10,
				Churn:              100,
				AgeDays:            365,
				UniqueContributors: 2,
				Gini:               0.5,
				RecentCommits:      5,
				Mode:               "hot",
			},
			mode: "hot",
		},
		{
			result: schema.FileResult{
				Path:               "test.go",
				SizeBytes:          0, // edge case
				LinesOfCode:        0,
				Commits:            0,
				Churn:              0,
				AgeDays:            0,
				UniqueContributors: 0,
				Gini:               0,
				RecentCommits:      0,
				Mode:               "risk",
			},
			mode: "risk",
		},
	}
	for _, seed := range seeds {
		f.Add(seed.result.Path, seed.result.SizeBytes, seed.result.LinesOfCode,
			seed.result.Commits, seed.result.Churn, seed.result.AgeDays,
			seed.result.UniqueContributors, seed.result.Gini, seed.result.RecentCommits,
			seed.mode)
	}

	f.Fuzz(func(_ *testing.T,
		path string,
		sizeBytes int64,
		linesOfCode int,
		commits int,
		churn int,
		ageDays int,
		uniqueContributors int,
		gini float64,
		recentCommits int,
		mode string,
	) {
		result := schema.FileResult{
			Path:               path,
			SizeBytes:          sizeBytes,
			LinesOfCode:        linesOfCode,
			Commits:            commits,
			Churn:              churn,
			AgeDays:            ageDays,
			UniqueContributors: uniqueContributors,
			Gini:               gini,
			RecentCommits:      recentCommits,
			Mode:               mode,
		}
		_ = computeScore(&result, mode, nil)
	})
}

// FuzzGini fuzzes the gini function with random value arrays.
func FuzzGini(f *testing.F) {
	seeds := []string{
		"[1,2,3]",
		"[0,0,0]",
		"[100]",
		"[]",
		"[1,1,1,1]",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, valuesJSON string) {
		// Simple parsing, may fail but that's ok for fuzzing
		values := []float64{}
		if valuesJSON != "" && valuesJSON[0] == '[' && valuesJSON[len(valuesJSON)-1] == ']' {
			// Very basic parsing, just for fuzzing
			inner := valuesJSON[1 : len(valuesJSON)-1]
			if inner != "" {
				parts := strings.SplitSeq(inner, ",")
				for p := range parts {
					// Skip parsing errors, just try
					if f, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
						values = append(values, f)
					}
				}
			}
		}
		_ = gini(values)
	})
}
