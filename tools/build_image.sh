#!/bin/bash
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# script vars
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
PROJECT_DIR=$(dirname "${SCRIPT_DIR}")

# flags
MULTIPLATFORM_BUILD=0
MULTIPLATFORM_TARGETS="linux/amd64,linux/arm64,linux/s390x"
BUILD_WITH_PODMAN=0
BUILD_WITH_DOCKER=0
PUSH_IMAGE=0

# files
DOCKERFILE="${PROJECT_DIR}/tools/images/Dockerfile"
BUILD_ARGS=()

# Function to display help message
function display_help() {
    cat <<EOF
Usage: $0 [OPTIONS]

Options:
  -t, --tag         Name to give to the image being built.
  --multiplatform   Build the image for ${MULTIPLATFORM_TARGETS}.
  --podman          Build the image using podman.
  --docker          Build the image using docker.
  --build-arg       Pass a build argument for the Dockerfile.
  --build-args-file Set the path to a file containing key-value lines to use as Docker build args.
  --push            Push the image to the container registry.
  -f, --dockerfile  Set the path where the Dockerfile is located.
  -h, --help        Display this help message and exit.

Examples:
  $0 --tag docker.io/hyperledger/fabric-x-tools:latest
  $0 --tag docker.io/hyperledger/fabric-x-tools:latest --multiplatform
EOF
}

function parse_script_args() {
    while [[ "$#" -gt 0 ]]; do
        case "$1" in
        -t | --tag)
            if [[ -n "$2" && "$2" != "--"* ]]; then
                IMAGE_TAG="$2"
                IMAGE_NAME="${IMAGE_TAG%%:*}"
                CURRENT_TAG="${IMAGE_TAG##*:}"
                shift
            else
                echo "Error: $2 option requires a value."
                exit 1
            fi
            ;;
        -f | --dockerfile)
            if [[ -n "$2" && "$2" != "--"* ]]; then
                DOCKERFILE="$2"
                PROJECT_DIR=$(dirname "$2")
                shift
            else
                echo "Error: $2 option requires a value."
                exit 1
            fi
            ;;
        --build-arg)
            if [[ -n "$2" && "$2" != --* ]]; then
                BUILD_ARGS+=("--build-arg" "$2")
                shift 2 # Move to the next pair
            else
                echo "Error: $2 requires a value"
                exit 1
            fi
            ;;
        --build-args-file)
            if [[ -n "$2" && "$2" != "--"* ]]; then
                prepare_build_args "$2"
                shift
            else
                echo "Error: $2 option requires a value."
                exit 1
            fi
            ;;
        --multiplatform)
            MULTIPLATFORM_BUILD=1
            ;;
        --podman)
            BUILD_WITH_PODMAN=1
            ;;
        --docker)
            BUILD_WITH_DOCKER=1
            ;;
        --push)
            PUSH_IMAGE=1
            ;;
        -h | --help)
            display_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            ;;
        esac
        shift
    done
}

# Create a list of arguments for the image build from a file.
function prepare_build_args() {
    if [ -f "$1" ]; then
        while IFS='=' read -r key value; do
            BUILD_ARGS+=("--build-arg" "$key=$value")
        done <"$1"
    fi
}

# Build a docker image for the current platform (podman and docker have same API for this case)
function build_single_platform_image() {
    $1 build -f "${DOCKERFILE}" \
        "${BUILD_ARGS[@]}" \
        -t "${IMAGE_TAG}" \
        --load \
        "${PROJECT_DIR}"
    echo "Successfully built image ${IMAGE_TAG} with $1."

    # push image to container registry
    if [[ $PUSH_IMAGE -eq 1 ]]; then
        $1 push "${IMAGE_TAG}"
        echo "Successfully pushed image ${IMAGE_TAG}."

        # push also as latest
        if [[ $CURRENT_TAG != "latest" ]]; then
            $1 tag "$IMAGE_TAG" "${IMAGE_NAME}:latest"
            $1 push "${IMAGE_NAME}:latest"
            echo "Successfully pushed image ${IMAGE_NAME}:latest"
        fi
    fi
}

# Build a docker image for multi-platform using docker
function build_multiplatform_image_with_docker() {
    local push_flag
    local image_tags=("-t" "${IMAGE_TAG}")

    # handle the push for docker buildx
    push_flag=$([ $PUSH_IMAGE -eq 1 ] && echo "true" || echo "false")
    if [[ $PUSH_IMAGE -eq 1 && $CURRENT_TAG != "latest" ]]; then
        image_tags+=("-t" "${IMAGE_NAME}:latest")
    fi

    docker buildx build \
        -f "${DOCKERFILE}" \
        --platform "${MULTIPLATFORM_TARGETS}" \
        --output "type=image,push=${push_flag}" \
        "${BUILD_ARGS[@]}" \
        "${image_tags[@]}" \
        "${PROJECT_DIR}"
    echo "Successfully built and pushed OCI manifest ${IMAGE_TAG} and ${IMAGE_NAME}:latest with docker."
}

# Build a docker image for multi-platform with podman CLI
function build_multiplatform_image_with_podman() {
    # delete existing manifest
    if podman manifest exists "${IMAGE_TAG}"; then
        echo "Deleting existing manifest: ${IMAGE_TAG}"
        podman manifest rm "${IMAGE_TAG}"
    fi

    # create new manifest
    podman manifest create "${IMAGE_TAG}"
    podman build \
        -f "${DOCKERFILE}" \
        --platform "${MULTIPLATFORM_TARGETS}" \
        "${BUILD_ARGS[@]}" \
        --manifest "${IMAGE_TAG}" \
        "${PROJECT_DIR}"
    echo "Successfully built OCI manifest ${IMAGE_TAG} with podman."

    # push image to container registry
    if [[ $PUSH_IMAGE -eq 1 ]]; then
        podman manifest push "${IMAGE_TAG}"
        echo "Successfully pushed OCI manifest ${IMAGE_TAG} with podman."

        # push also as latest
        if [[ $CURRENT_TAG != "latest" ]]; then
            podman tag "$IMAGE_TAG" "${IMAGE_NAME}:latest"
            podman manifest push "${IMAGE_NAME}:latest"
            echo "Successfully pushed OCI manifest ${IMAGE_NAME}:latest with podman."
        fi
    fi
}

# Build a docker image using podman CLI
function build_image_with_podman() {
    if [[ $MULTIPLATFORM_BUILD -eq 0 ]]; then
        build_single_platform_image podman
    else
        build_multiplatform_image_with_podman
    fi
}

# Build the docker image with docker CLI
function build_image_with_docker() {
    if [[ $MULTIPLATFORM_BUILD -eq 0 ]]; then
        build_single_platform_image docker
    else
        build_multiplatform_image_with_docker
    fi
}

# Build a docker image
function build_docker_image() {
    if [[ BUILD_WITH_PODMAN -eq 1 ]]; then
        if command -v podman &>/dev/null; then
            build_image_with_podman
        else
            echo "Error: Podman is not installed."
            exit 1
        fi
    elif [[ BUILD_WITH_DOCKER -eq 1 ]]; then
        if command -v docker &>/dev/null; then
            build_image_with_docker
        else
            echo "Error: Docker is not installed."
            exit 1
        fi
        build_image_with_docker
    elif command -v podman &>/dev/null; then
        build_image_with_podman
    elif command -v docker &>/dev/null; then
        build_image_with_docker
    else
        echo "Error: Neither Docker nor Podman is installed."
        exit 1
    fi
}

# prepare script
set -eo pipefail
trap exit 1 INT
parse_script_args "$@"

# Check if target image name is provided
if [ -z "${IMAGE_TAG}" ]; then
    display_help
    exit 1
fi

build_docker_image