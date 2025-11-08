# 🤝 Contributing to Hotspot

We warmly welcome contributions to Hotspot! Whether you're reporting a bug, suggesting a new feature, or submitting a code change, your input helps make Hotspot a better tool for everyone.

## Getting Started

Hotspot is a Go CLI tool. To get started with development, clone the repository and ensure you have a recent version of Go installed.

```bash
git clone https://github.com/huangsam/hotspot.git
cd hotspot
make
```

## How to Submit Feedback and Issues

Before submitting an issue, please check the existing issues to see if the problem has already been reported.

We have defined separate templates to make sure we get the necessary information to act quickly:

- **🐛 Bug Reports:** Use the Bug Report template if you encounter a crash, an unexpected error, or if the output is demonstrably incorrect (e.g., a file's score is wrong). We need the exact command you ran and your system details to reproduce the issue.
- **✨ Feature Requests & Feedback:** Use the Feature Request template to propose a new scoring mode, a new output format, or to offer general usability feedback. Explain the problem you are trying to solve and your suggested solution.

## Testing

Hotspot uses Go's standard testing framework. Run tests with:

```bash
# Run unit tests only
make test

# Run all tests
make test-all
```

Integration tests are tagged with `//go:build integration` and are excluded from the default test suite to prevent them from running in CI or during normal development. They verify that hotspot's output matches `git log` exactly for both internal and external repositories.

## Submitting Code Changes

We encourage contributions of code, documentation, and tests!

- Fork the repository and create your branch from main
- **Make Changes:** Write your code, following the existing Go style and ensuring good test coverage
- **Run Tests:** Use `make test` for unit tests and `make test-integration` for integration tests
- **Ensure Consistency:** Run `make check` before submitting to ensure formatting, linting, and tests pass
- **Submit a Pull Request (PR):** Target the main branch. Provide a clear title and description explaining the purpose of your change and referencing any related issues

Thank you for contributing to Hotspot!
