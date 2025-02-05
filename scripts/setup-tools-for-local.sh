#!/bin/bash

# 📚 Documentation
# ===============
# This script sets up development tools and generates taskfiles for local development
#
# Features:
# 🔧 Builds development tools from tools.go
# 📝 Generates task definitions for Task
# 🔄 Handles script permissions
# 🎯 Supports incremental builds
#
# Usage:
#   ./setup-tools-for-local.sh [flags]
#
# Flags:
#   --skip-build         : Skip building tools (default: false)
#   --generate-taskfiles : Generate taskfiles (default: false)
#
# Environment Variables:
#   SCRIPTS_DIR         : Directory containing scripts (default: ./scripts)
#   TASKFILE_OUTPUT_DIR : Directory for generated taskfiles (default: ./out/taskfiles)
#   TOOLS_OUTPUT_DIR    : Directory for built tools (default: ./out/tools)

set -euo pipefail

# 🎯 Default values
SKIP_BUILD="false"
GENERATE_TASKFILES="false"

# 🔄 Parse command line flags
while [[ "$#" -gt 0 ]]; do
	case $1 in
	--generate-taskfiles)
		GENERATE_TASKFILES="true"
		shift
		;;
	*) shift ;;
	esac
done

# 📂 Setup directories
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 🔧 Configure paths
: ${SCRIPTS_DIR:="${ROOT_DIR}/scripts"}
: ${TASKFILE_OUTPUT_DIR:="./out/taskfiles"}
: ${TOOLS_OUTPUT_DIR:="./out/tools"}

# 📝 Generate taskfiles if requested
if [ "$GENERATE_TASKFILES" = "true" ]; then
	rm -rf "$TASKFILE_OUTPUT_DIR"
	mkdir -p "$TASKFILE_OUTPUT_DIR"

	output_taskfile="$TASKFILE_OUTPUT_DIR/Taskfile.tools.yml"
	rm -f "$output_taskfile"

	# Create tools taskfile header
	cat <<EOF >$output_taskfile
version: '3'

tasks:
EOF
fi

get_tool_module_name() {
	# 📝 Extract tool name from path
	tool_name=$(basename "$1")

	resetr=$(cat ./tools/go.mod | grep "$1" | grep -o -E 'name:\s*(\S*)' | cut -d ":" -f 2 | xargs || true)

	if [ -n "$resetr" ]; then
		tool_name=$resetr
	elif [[ $tool_name == v* ]]; then
		tool_name=$(basename "$(dirname "$1")")
	fi

	echo "$tool_name"
}

build_tool() {
	local tool_module_path="$1"
	tool_name=$(get_tool_module_name "$tool_module_path")

	# Add task definition if generating taskfiles
	if [ "$GENERATE_TASKFILES" = "true" ]; then
		cat <<EOF >>$output_taskfile

    ${tool_name}:
        desc: run ${tool_name} - built from ${tool_module_path}
        cmds:
            - ./go tool ${tool_module_path} {{.CLI_ARGS}}
EOF
	fi
}

for tool in $(go list tool); do
	build_tool "$tool"
done

# 📋 Generate scripts taskfile if requested
if [ "$GENERATE_TASKFILES" = "true" ]; then
	output_file="${TASKFILE_OUTPUT_DIR}/Taskfile.scripts.yml"
	rm -f "$output_file"

	# Create scripts taskfile header
	cat <<EOF >$output_file
version: '3'

tasks:
EOF

	# Add task for each script
	for script in $(ls scripts); do
		# Skip self
		if [[ $script == $(basename "$0") ]]; then
			continue
		fi

		# Process shell scripts
		if [[ $script == *.sh ]]; then
			# Ensure script is executable
			if [[ ! -x ${SCRIPTS_DIR}/${script} ]]; then
				chmod +x ${SCRIPTS_DIR}/${script}
			fi

			script_name=${script%.sh}
			cat <<EOF >>$output_file

    ${script_name}:
        desc: run $SCRIPTS_DIR/${script_name}.sh
        cmds:
            - $SCRIPTS_DIR/${script_name}.sh {{.CLI_ARGS}}
EOF
		fi
	done
fi

echo "✅ Setup complete!"
