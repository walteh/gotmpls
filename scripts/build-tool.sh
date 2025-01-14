#!/bin/bash

# 📚 Documentation
# ===============
# This script builds a Go tool with specific configuration and creates a SHA256 checksum
#
# Features:
# 🔧 Cross-platform build support
# 🔒 CGO disabled for better portability
# 📦 Handles versioned module paths
# 🔍 Generates SHA256 checksums
#
# Environment Variables:
#   TOOL_MODULE_PATH : Path to the Go module to build (required)
#   OUTPUT_DIR      : Directory to output the built binary (required)
#   GOOS           : Target operating system (required)
#   GOARCH         : Target architecture (required)
#   GOPROXY        : Go module proxy (default: https://proxy.golang.org,direct)
#   SKIP_BUILD     : Skip the build step (default: false)
#
# Example:
#   TOOL_MODULE_PATH="./tools/mytool" \
#   OUTPUT_DIR="./out/tools" \
#   GOOS="linux" \
#   GOARCH="amd64" \
#   ./build-tool.sh

set -euo pipefail

# 🔍 Validate required environment variables
: ${TOOL_MODULE_PATH:?❌ Missing TOOL_MODULE_PATH}
: ${OUTPUT_DIR:?❌ Missing OUTPUT_DIR}
: ${GOOS:?❌ Missing GOOS}
: ${GOARCH:?❌ Missing GOARCH}
: ${GOPROXY:=https://proxy.golang.org,direct}
: ${SKIP_BUILD:="false"}

# 📝 Extract tool name from path
tool_name=$(basename "$TOOL_MODULE_PATH")

# 🔄 Handle versioned modules (v* directories)
if [[ $tool_name == v* ]]; then
	tool_name=$(basename "$(dirname "$TOOL_MODULE_PATH")")
fi

# 🔍 Get current module name
mymodname=$(go list -m | head -n 1)

# 🛠️ Handle local module paths
if [[ $TOOL_MODULE_PATH == $mymodname* ]]; then
	TOOL_MODULE_PATH="./${TOOL_MODULE_PATH#$mymodname/}"
fi

# 📦 Build the tool
if [[ $SKIP_BUILD == "false" || $SKIP_BUILD == "0" ]]; then
	echo "🔨 Building $tool_name"
	echo "├── 📂 Source: $TOOL_MODULE_PATH"
	echo "├── 🎯 Target: $GOOS/$GOARCH"
	echo "└── 📤 Output: $OUTPUT_DIR/$tool_name"

	GOPROXY=$GOPROXY CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
		go build -mod=readonly -ldflags="-s -w" \
		-o "$OUTPUT_DIR/$tool_name" "$TOOL_MODULE_PATH"

	echo "📝 Generating SHA256 checksum..."
	sha256sum "$OUTPUT_DIR/$tool_name" >"$OUTPUT_DIR/$tool_name.sha256"
fi

# 📤 Export variables for other scripts
export TOOL_NAME="$tool_name"
export TOOL_PATH="$OUTPUT_DIR/$tool_name"
export TOOL_SHA256="$OUTPUT_DIR/$tool_name.sha256"

echo "✅ Build complete: $tool_name"
