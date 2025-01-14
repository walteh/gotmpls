#! /bin/bash

set -euo pipefail

: ${TOOL_MODULE_PATH:?missing tool module path}
: ${OUTPUT_DIR:?missing output dir}
: ${GOOS:?missing goos}
: ${GOARCH:?missing goarch}
: ${GOPROXY:=https://proxy.golang.org,direct} # Add default GOPROXY if not set
: ${SKIP_BUILD:="false"}

tool_name=$(basename "$TOOL_MODULE_PATH")

# if tool_name is v*, then we need to use the name of the directory
if [[ $tool_name == v* ]]; then
	tool_name=$(basename "$(dirname "$TOOL_MODULE_PATH")")
fi

mymodname=$(go list -m | head -n 1)
# if the tool import path starts with mymodname, then we remove the prefix
if [[ $TOOL_MODULE_PATH == $mymodname* ]]; then
	TOOL_MODULE_PATH="./${TOOL_MODULE_PATH#$mymodname/}"
fi

if [[ $SKIP_BUILD == "false" || $SKIP_BUILD == "0" ]]; then
	echo "building tool [$tool_name] from [$TOOL_MODULE_PATH]"
	GOPROXY=$GOPROXY CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -mod=readonly -ldflags="-s -w" -o "$OUTPUT_DIR/$tool_name" "$TOOL_MODULE_PATH"
	sha256sum "$OUTPUT_DIR/$tool_name" >"$OUTPUT_DIR/$tool_name.sha256"
fi

export TOOL_NAME="$tool_name"
export TOOL_PATH="$OUTPUT_DIR/$tool_name"
export TOOL_SHA256="$OUTPUT_DIR/$tool_name.sha256"
