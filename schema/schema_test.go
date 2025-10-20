package schema

import "testing"

func TestGetRecentCommitsMapGlobal(t *testing.T) {
	globalMapRefOne := GetRecentCommitsMapGlobal()
	globalMapRefTwo := GetRecentCommitsMapGlobal()
	globalMapRefOne["a"]++
	globalMapRefTwo["b"]++
	if globalMapRefTwo["a"] != 1 {
		t.Error("Expected globalMapRefTwo to be updated correctly")
	}
	if globalMapRefOne["b"] != 1 {
		t.Error("Expected globalMapRefOne to be updated correctly")
	}
}
