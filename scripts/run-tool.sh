#! /bin/bash

: ${TOOLS_BIN_DIR:=./out/tools}

# set -e pipefail

first_arg="$1"
shift

# export GOTAG_DEBUG=true

export PATH="$(pwd)/scripts:$PATH"

function _list_available_tools() {
	# List tools from both the binary directory and tools.go
	{
		# List compiled tools
		if [ -d "$TOOLS_BIN_DIR" ]; then
			find "$TOOLS_BIN_DIR" -type f -executable -printf "%f\n"
		fi
		# List tools from tools.go
		if [ -f "./tools/tools.go" ]; then
			grep -o '"[^"]*"' ./tools/tools.go | tr -d '"' | awk -F'/' '{print $NF}'
		fi
	} | sort -u
}

function try_run_tool_with_go_run() {
	tool_import_path=$(grep -r "$first_arg" ./tools/tools.go | head -n 1)
	tool_import_path=${tool_import_path#*_}
	tool_import_path=${tool_import_path#*\"}
	tool_import_path=${tool_import_path%\"*}
	echo "WARNING: $first_arg was not found pre-built, running go run $tool_import_path $@" >&2
	go run "$tool_import_path" "$@"
}

# Add completion support
if [ "${1-}" = "--complete" ]; then
	_list_available_tools
	exit 0
fi

if [ ! -x "$TOOLS_BIN_DIR/$first_arg" ]; then
	try_run_tool_with_go_run "$@"
	exit $?
fi

escape_regex() {
	printf '%s\n' "$1" | sed 's/[][(){}.*+?^$|\\]/\\&/g'
}

errors_to_suppress=(
	# https://github.com/protocolbuffers/protobuf-javascript/issues/148
	"reference https://github.com/protocolbuffers/protobuf/blob/95e6c5b4746dd7474d540ce4fb375e3f79a086f8/src/google/protobuf/compiler/plugin.proto#L122"
)

errors_to_suppress_regex=""
for phrase in "${errors_to_suppress[@]}"; do
	escaped_phrase=$(escape_regex "$phrase")
	if [[ -n "$errors_to_suppress_regex" ]]; then
		errors_to_suppress_regex+="|"
	fi
	errors_to_suppress_regex+="$escaped_phrase"
done

# pass stdin to the tool and write to stdout
"$TOOLS_BIN_DIR/$first_arg" "$@" <&0 >&1 2> >(grep -Ev "$errors_to_suppress_regex" >&2)
