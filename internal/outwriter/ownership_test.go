package outwriter

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
)

func TestFormatOwnershipDiff(t *testing.T) {
	tests := []struct {
		name     string
		details  schema.ComparisonDetails
		expected string
	}{
		{
			name: "new file with single owner",
			details: schema.ComparisonDetails{
				Status:       schema.NewStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{"Alice"},
			},
			expected: "New: Alice",
		},
		{
			name: "new file with multiple owners",
			details: schema.ComparisonDetails{
				Status:       schema.NewStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{"Alice", "Bob"},
			},
			expected: "New: Alice, Bob",
		},
		{
			name: "new file with no owners",
			details: schema.ComparisonDetails{
				Status:       schema.NewStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{},
			},
			expected: "New",
		},
		{
			name: "removed file with single owner",
			details: schema.ComparisonDetails{
				Status:       schema.InactiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{},
			},
			expected: "Removed: Alice",
		},
		{
			name: "removed file with multiple owners",
			details: schema.ComparisonDetails{
				Status:       schema.InactiveStatus,
				BeforeOwners: []string{"Alice", "Bob"},
				AfterOwners:  []string{},
			},
			expected: "Removed: Alice, Bob",
		},
		{
			name: "removed file with no previous owners",
			details: schema.ComparisonDetails{
				Status:       schema.InactiveStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{},
			},
			expected: "Removed",
		},
		{
			name: "active file with stable single owner",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Alice"},
			},
			expected: "Alice (stable)",
		},
		{
			name: "active file with stable multiple owners same order",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice", "Bob"},
				AfterOwners:  []string{"Alice", "Bob"},
			},
			expected: "Alice, Bob (stable)",
		},
		{
			name: "active file with stable multiple owners different order",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice", "Bob"},
				AfterOwners:  []string{"Bob", "Alice"},
			},
			expected: "Bob, Alice (stable)",
		},
		{
			name: "active file with changed ownership",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Bob"},
			},
			expected: "Bob",
		},
		{
			name: "active file with no current owners but had previous",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{},
			},
			expected: "No owners (was: Alice)",
		},
		{
			name: "active file with current owners but no previous",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{"Bob"},
			},
			expected: "Bob",
		},
		{
			name: "active file with no owners before or after",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{},
			},
			expected: "No owners",
		},
		{
			name: "active file with multiple current owners",
			details: schema.ComparisonDetails{
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Bob", "Charlie"},
			},
			expected: "Bob, Charlie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatOwnershipDiff(tt.details)
			if result != tt.expected {
				t.Errorf("formatOwnershipDiff() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
