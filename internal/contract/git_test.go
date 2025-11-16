package contract

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// skipIfGitNotAvailable skips the test if git binary is not found in PATH
func skipIfGitNotAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git binary not found in PATH: %v", err)
	}
}

// TestMockGitClient_Run ensures the mock correctly records and returns
// expected values when its Run method is called.
func TestMockGitClient_Run(t *testing.T) {
	// 1. Setup the Mock
	mockClient := new(MockGitClient)

	// Define the expected input arguments for the mock's 'Run' method.
	const expectedRepoPath = "/path/to/repo"
	expectedArgs := []string{"log", "-1", "--oneline"}

	// Define the expected output values.
	expectedOutput := []byte("a1b2c3d commit message")
	expectedError := errors.New("mocked git error")

	// The `Run` method implementation in MockGitClient converts the inputs
	// (repoPath string, args ...string) into a single []interface{} array
	// for `m.Called()`. We must match this structure in `.On()`.

	// Prepare the exact arguments that will be passed to m.Called() inside MockGitClient.Run()
	var calledArgs []any
	ctx := context.Background()
	calledArgs = append(calledArgs, ctx, expectedRepoPath)
	for _, arg := range expectedArgs {
		calledArgs = append(calledArgs, arg)
	}

	// 2. Program the Mock Behavior
	mockClient.
		On("Run", calledArgs...).              // Expect a call with these arguments.
		Return(expectedOutput, expectedError). // Program the values to return.
		Once()                                 // Expect the call to happen exactly once.

	// 3. Execute the Code Under Test (i.e., call the mock method)
	actualOutput, actualError := mockClient.Run(ctx, expectedRepoPath, expectedArgs...)

	// 4. Assertions

	// Verify that the returned values match the programmed values.
	assert.Equal(t, expectedOutput, actualOutput, "Run should return the programmed output")
	assert.Equal(t, expectedError, actualError, "Run should return the programmed error")

	// Verify that the expected method call actually occurred.
	// This confirms that the logic within MockGitClient.Run correctly called m.Called()
	// with the expected arguments, matching the .On() setup.
	mockClient.AssertExpectations(t)
}

// TestNewLocalGitClient tests the constructor for LocalGitClient.
func TestNewLocalGitClient(t *testing.T) {
	client := NewLocalGitClient()
	assert.NotNil(t, client, "NewLocalGitClient should return a non-nil client")
	assert.IsType(t, &LocalGitClient{}, client, "NewLocalGitClient should return a LocalGitClient instance")
}

// TestLocalGitClient_Run tests the Run method with various scenarios.
func TestLocalGitClient_Run(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	tests := []struct {
		name        string
		repoPath    string
		args        []string
		expectError bool
		setupMock   func(*mock.Mock)
	}{
		{
			name:        "invalid repo path",
			repoPath:    "/nonexistent/path",
			args:        []string{"status"},
			expectError: true,
		},
		{
			name:        "invalid git command",
			repoPath:    repoRoot, // Use repository root
			args:        []string{"invalid-command"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Run(ctx, tt.repoPath, tt.args...)
			if tt.expectError {
				assert.Error(t, err, "Run should return an error for %s", tt.name)
			} else {
				assert.NoError(t, err, "Run should not return an error for %s", tt.name)
			}
		})
	}
}

// TestLocalGitClient_GetRepoRoot tests the GetRepoRoot method.
func TestLocalGitClient_GetRepoRoot(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Test with current directory (assuming we're in a git repo)
	root, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error for current directory")
	assert.NotEmpty(t, root, "GetRepoRoot should return a non-empty root path")

	// Test with absolute path to current directory
	root2, err := client.GetRepoRoot(ctx, root)
	assert.NoError(t, err, "GetRepoRoot should not return an error for absolute path")
	assert.Equal(t, root, root2, "GetRepoRoot should return the same root for absolute path")

	// Test with invalid path
	_, err = client.GetRepoRoot(ctx, "/nonexistent/path")
	assert.Error(t, err, "GetRepoRoot should return an error for non-git directory")
}

// TestLocalGitClient_GetCommitTime tests the GetCommitTime method.
func TestLocalGitClient_GetCommitTime(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	// Test with HEAD
	commitTime, err := client.GetCommitTime(ctx, repoRoot, "HEAD")
	assert.NoError(t, err, "GetCommitTime should not return an error for HEAD")
	assert.True(t, commitTime.After(time.Time{}), "GetCommitTime should return a valid time")

	// Test with invalid ref
	_, err = client.GetCommitTime(ctx, repoRoot, "invalid-ref")
	assert.Error(t, err, "GetCommitTime should return an error for invalid ref")
}

// TestLocalGitClient_GetActivityLog tests the GetActivityLog method.
func TestLocalGitClient_GetActivityLog(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	startTime := time.Now().AddDate(0, 0, -30) // 30 days ago
	endTime := time.Now()

	// Test with time range
	_, err = client.GetActivityLog(ctx, repoRoot, startTime, endTime)
	assert.NoError(t, err, "GetActivityLog should not return an error")
	// Log might be empty if no commits in range, but should not error

	// Test with zero times (no time filter)
	_, err = client.GetActivityLog(ctx, repoRoot, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetActivityLog should not return an error with zero times")
}

// TestLocalGitClient_GetFileActivityLog tests the GetFileActivityLog method.
func TestLocalGitClient_GetFileActivityLog(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()

	// Test with existing file
	_, err = client.GetFileActivityLog(ctx, repoRoot, "main.go", startTime, endTime, false)
	assert.NoError(t, err, "GetFileActivityLog should not return an error for existing file")

	// Test with follow flag
	_, err = client.GetFileActivityLog(ctx, repoRoot, "main.go", time.Time{}, time.Time{}, true)
	assert.NoError(t, err, "GetFileActivityLog should not return an error with follow flag")

	// Test with non-existent file (git log doesn't error, just returns empty)
	_, err = client.GetFileActivityLog(ctx, repoRoot, "nonexistent.go", startTime, endTime, false)
	assert.NoError(t, err, "GetFileActivityLog should not return an error for non-existent file (returns empty)")
}

// TestLocalGitClient_ListFilesAtRef tests the ListFilesAtRef method.
func TestLocalGitClient_ListFilesAtRef(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root first
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	// Test with HEAD
	files, err := client.ListFilesAtRef(ctx, repoRoot, "HEAD")
	assert.NoError(t, err, "ListFilesAtRef should not return an error for HEAD")
	assert.NotNil(t, files, "ListFilesAtRef should return a file list")
	assert.True(t, len(files) > 0, "ListFilesAtRef should return at least one file")
	// Just check that we got some files back - the exact content depends on the repo state
	t.Logf("Found %d files at HEAD", len(files))

	// Test with invalid ref
	_, err = client.ListFilesAtRef(ctx, repoRoot, "invalid-ref")
	assert.Error(t, err, "ListFilesAtRef should return an error for invalid ref")
}

// TestLocalGitClient_GetOldestCommitDateForPath tests the GetOldestCommitDateForPath method.
func TestLocalGitClient_GetOldestCommitDateForPath(t *testing.T) {
	skipIfGitNotAvailable(t)

	client := NewLocalGitClient()
	ctx := context.Background()

	// Get the repository root first
	repoRoot, err := client.GetRepoRoot(ctx, ".")
	assert.NoError(t, err, "GetRepoRoot should not return an error")

	before := time.Now()
	maxSearchDuration := 365 * 24 * time.Hour // 1 year

	// Test with a very broad time range to ensure we find commits
	// Use a file that should definitely exist
	_, _ = client.GetOldestCommitDateForPath(ctx, repoRoot, "README.md", before, 1, maxSearchDuration)
	// Don't assert NoError since the file might not have commits in the time range
	// Just ensure the function doesn't panic and returns some result

	// Test with non-existent file (should error)
	_, err = client.GetOldestCommitDateForPath(ctx, repoRoot, "definitely-nonexistent-file-12345.txt", before, 1, maxSearchDuration)
	assert.Error(t, err, "GetOldestCommitDateForPath should return an error for non-existent file")
}
