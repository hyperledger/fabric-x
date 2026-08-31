#!/usr/bin/env bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# Cross-compiles the Fabric-X CLI tools for a single OS/arch and packages
# them into a versioned tarball under release/<TARGET>/.
#
# Required env vars:
#   TARGET   <os>-<arch>, e.g. linux-amd64, darwin-arm64
#   RELEASE  git tag, e.g. v1.0.0
# Optional env vars:
#   REVISION commit SHA (defaults to `git rev-parse HEAD`)
#
# Usage:
#   TARGET=darwin-arm64 RELEASE=v1.0.0 ./scripts/create-binary-package.sh

# Exit on error/unset var/pipeline failure.
set -euo pipefail

# Fail fast with a clear message if TARGET or RELEASE aren't set.
: "${TARGET:?TARGET is required, e.g. TARGET=linux-amd64}"
: "${RELEASE:?RELEASE is required, e.g. RELEASE=v1.0.0}"

# Default REVISION to the current commit SHA if not passed in.
REVISION="${REVISION:-$(git rev-parse HEAD)}"

# Split TARGET (e.g. "linux-amd64") into GOOS and GOARCH.
if [[ ! "$TARGET" =~ ^([a-z0-9]+)-([a-z0-9]+)$ ]]; then
  echo "ERROR: TARGET must be in <os>-<arch> form, got: ${TARGET}" >&2
  exit 1
fi
GOOS="${BASH_REMATCH[1]}"
GOARCH="${BASH_REMATCH[2]}"

# Trim the leading 'v' from the tag, matching build-image.yml's version extraction.
VERSION="${RELEASE#v}"

# Cross-compile via the Makefile, stamping version/commit into the binaries.
RELEASE_DIR="release/${TARGET}"
echo "Building ${TARGET} binaries for ${RELEASE} (${REVISION})..."
make release-bins GOOS="${GOOS}" GOARCH="${GOARCH}" RELEASE_DIR=release \
  METADATA_VAR="Version=${VERSION} CommitSHA=${REVISION}"

# Ship the license alongside the binaries.
cp LICENSE "${RELEASE_DIR}/"

# -C cds into RELEASE_DIR first, so the archive holds relative paths (bin/, LICENSE).
ARCHIVE="fabric-x-tools-${TARGET}-${VERSION}.tar.gz"
tar -czf "${RELEASE_DIR}/${ARCHIVE}" -C "${RELEASE_DIR}" bin LICENSE

echo "${RELEASE_DIR}/${ARCHIVE}"
