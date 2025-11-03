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
	assert.NotEqual(t, 0, globalMapRefTwo["a"], "both maps should have 'a' incremented")
	assert.NotEqual(t, 0, globalMapRefOne["b"], "both maps should have 'b' incremented")
}
