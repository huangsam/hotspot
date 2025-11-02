package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRecentCommitsMapGlobal(t *testing.T) {
	globalMapRefOne := GetRecentCommitsMapGlobal()
	globalMapRefTwo := GetRecentCommitsMapGlobal()
	globalMapRefOne["a"]++
	globalMapRefTwo["b"]++
	assert.Equal(t, 1, globalMapRefTwo["a"], "both maps should have 'a' incremented")
	assert.Equal(t, 1, globalMapRefOne["b"], "both maps should have 'b' incremented")
}
