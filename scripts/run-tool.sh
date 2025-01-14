#!/bin/bash

# 📚 Documentation
# ===============
# This script runs a tool from the tools directory, with fallback to go run if not built
#
# Features:
# 🔍 Auto-discovers available tools
# 🔄 Fallback to go run if tool not built
# 🚫 Suppresses known warning messages
# 🛠️ Shell completion support
#
# Usage:
#   ./run-tool.sh <tool-name> [args...]
#   ./run-tool.sh --complete  # List available tools
#
# Environment Variables:
#   TOOLS_BIN_DIR : Directory containing built tools (default: ./out/tools)
#   GOTAG_DEBUG   : Enable debug output for gotag (optional)
#
# Example:
#   ./run-tool.sh mockery --all
#   ./run-tool.sh protoc --version

# 🎯 Default configuration
: ${TOOLS_BIN_DIR:=./out/tools}

# 🔄 Parse arguments
first_arg="$1"
shift

# 🛠️ Add scripts to PATH
export PATH="$(pwd)/scripts:$PATH"

# 📋 List available tools function
function _list_available_tools() {
	echo "🔍 Available tools:"
	{
		# List compiled tools
		if [ -d "$TOOLS_BIN_DIR" ]; then
			find "$TOOLS_BIN_DIR" -type f -executable -printf "├── 📦 %f\n"
		fi
		# List tools from tools.go
		if [ -f "./tools/tools.go" ]; then
			grep -o '"[^"]*"' ./tools/tools.go | tr -d '"' | awk -F'/' '{printf "├── 🔧 %s\n", $NF}'
		fi
	} | sort -u
	echo "└── Done"
}

# 🏃 Run tool with go run if not built
function try_run_tool_with_go_run() {
	tool_import_path=$(grep -r "$first_arg" ./tools/tools.go | head -n 1)
	tool_import_path=${tool_import_path#*_}
	tool_import_path=${tool_import_path#*\"}
	tool_import_path=${tool_import_path%\"*}
	echo "⚠️  $first_arg not found pre-built, using: go run $tool_import_path" >&2
	go run "$tool_import_path" "$@"
}

# 🔄 Handle shell completion
if [ "${1-}" = "--complete" ]; then
	_list_available_tools
	exit 0
fi

# 🔍 Check if tool exists and run it
if [ ! -x "$TOOLS_BIN_DIR/$first_arg" ]; then
	try_run_tool_with_go_run "$@"
	exit $?
fi

# 🛠️ Helper function to escape regex
escape_regex() {
	printf '%s\n' "$1" | sed 's/[][(){}.*+?^$|\\]/\\&/g'
}

# 🚫 Messages to suppress
errors_to_suppress=(
	# https://github.com/protocolbuffers/protobuf-javascript/issues/148
	"reference https://github.com/protocolbuffers/protobuf/blob/95e6c5b4746dd7474d540ce4fb375e3f79a086f8/src/google/protobuf/compiler/plugin.proto#L122"
)

# 🔧 Build regex for suppressing errors
errors_to_suppress_regex=""
for phrase in "${errors_to_suppress[@]}"; do
	escaped_phrase=$(escape_regex "$phrase")
	if [[ -n "$errors_to_suppress_regex" ]]; then
		errors_to_suppress_regex+="|"
	fi
	errors_to_suppress_regex+="$escaped_phrase"
done

# 🚀 Run the tool
"$TOOLS_BIN_DIR/$first_arg" "$@" <&0 >&1 2> >(grep -Ev "$errors_to_suppress_regex" >&2)
