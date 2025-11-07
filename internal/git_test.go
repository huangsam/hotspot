package internal

import (
	"context"
	"errors"
	"testing"

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
