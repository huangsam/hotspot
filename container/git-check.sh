#!/usr/bin/env bash
set -euo pipefail

# git-check.sh
# Check that git is installed and that its version is greater than or equal to 2.2.0
# Exit codes:
#   0 - OK (git installed and version >= 2.2.0)
#   1 - git not found
#   2 - git version is not greater than 2.2.0
#   3 - unable to parse git version
#
# Usage: ./git-check.sh

MIN_VERSION='2.2.0'

if ! command -v git >/dev/null 2>&1; then
	echo "ERROR: git is not installed or not on PATH"
	exit 1
fi

raw_ver=$(git --version 2>/dev/null || true)
# Extract first numeric version-like token, e.g. 2.25.1 from "git version 2.25.1"
ver=$(printf '%s' "$raw_ver" | grep -oE '[0-9]+(\.[0-9]+)*' | head -n1 || true)

if [ -z "$ver" ]; then
	echo "ERROR: unable to parse git version from: $raw_ver"
	exit 3
fi

# Compare two dot-separated versions. Return 0 if first >= second, 1 otherwise.
version_ge() {
	local IFS=.
	local -a a b
	local i ai bi
	IFS=. read -r -a a <<< "$1"
	IFS=. read -r -a b <<< "$2"
	for i in 0 1 2; do
		ai=${a[i]:-0}
		bi=${b[i]:-0}
		# Use base-10 to avoid issues with leading zeros
		if ((10#$ai > 10#$bi)); then
			return 0
		elif ((10#$ai < 10#$bi)); then
			return 1
		fi
	done
	# equal -> greater-or-equal
	return 0
}

if version_ge "$ver" "$MIN_VERSION"; then
	echo "OK: git $ver is greater than or equal to $MIN_VERSION"
	exit 0
else
	echo "ERROR: git $ver is less than $MIN_VERSION"
	exit 2
fi

