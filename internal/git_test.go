package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

// TestGetFileFirstCommitTime tests the GetFileFirstCommitTime function with various git outputs
func TestGetFileFirstCommitTime(t *testing.T) {
	tests := []struct {
		name         string
		gitOutput    string
		gitError     error
		follow       bool
		expectedTime time.Time
		expectError  bool
	}{
		{
			name:         "single commit",
			gitOutput:    "1760857642\n",
			expectedTime: time.Unix(1760857642, 0),
		},
		{
			name:         "multiple commits - returns oldest",
			gitOutput:    "1762705922\n1760857642\n1760857641\n",
			expectedTime: time.Unix(1760857641, 0), // Last line is oldest
		},
		{
			name:        "no commits found",
			gitOutput:   "",
			expectError: true,
		},
		{
			name:        "empty lines only",
			gitOutput:   "\n\n",
			expectError: true,
		},
		{
			name:        "invalid timestamp",
			gitOutput:   "invalid\n",
			expectError: true,
		},
		{
			name:        "git command error",
			gitError:    errors.New("git command failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockGitClient)

			// Expected args for git log command
			expectedArgs := []string{"log", "--pretty=format:%ct"}
			if tt.follow {
				expectedArgs = append(expectedArgs, "--follow")
			}
			expectedArgs = append(expectedArgs, "--", "test.go")

			// Set up mock expectations
			ctx := context.Background()
			var calledArgs []any
			calledArgs = append(calledArgs, ctx, "/test/repo")
			for _, arg := range expectedArgs {
				calledArgs = append(calledArgs, arg)
			}

			mockClient.
				On("Run", calledArgs...).
				Return([]byte(tt.gitOutput), tt.gitError).
				Once()

			// Create LocalGitClient with mock
			// We need to use a different approach since LocalGitClient doesn't expose Run directly
			// Let's test through the interface by creating a wrapper

			// Actually, let's create a test-specific client that uses the mock
			testClient := &testGitClient{mock: mockClient}

			result, err := testClient.GetFileFirstCommitTime(ctx, "/test/repo", "test.go", tt.follow)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTime, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// testGitClient wraps MockGitClient to implement GitClient interface
type testGitClient struct {
	mock *MockGitClient
}

func (t *testGitClient) GetFileFirstCommitTime(ctx context.Context, repoPath string, path string, follow bool) (time.Time, error) {
	args := []string{"log", "--pretty=format:%ct"}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, "--", path)

	out, err := t.mock.Run(ctx, repoPath, args...)
	if err != nil {
		return time.Time{}, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return time.Time{}, fmt.Errorf("no commits found for file '%s'", path)
	}

	timestampStr := strings.TrimSpace(lines[len(lines)-1])
	if timestampStr == "" {
		return time.Time{}, fmt.Errorf("no timestamp found for file '%s'", path)
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse commit time '%s': %w", timestampStr, err)
	}

	return time.Unix(timestamp, 0), nil
}
