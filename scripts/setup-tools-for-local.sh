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
	--skip-build)
		SKIP_BUILD="true"
		shift
		;;
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

# 🧹 Clean and create tools directory if building
if [ "$SKIP_BUILD" = "false" ]; then
	echo "🔧 Setting up tools directory..."
	rm -rf "$TOOLS_OUTPUT_DIR"
	mkdir -p "$TOOLS_OUTPUT_DIR"
fi

# 📝 Generate taskfiles if requested
if [ "$GENERATE_TASKFILES" = "true" ]; then
	echo "📝 Generating taskfiles..."
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

# 🛠️ Build tool function
build_tool() {
	local import_path="$1"
	echo "🔨 Building tool from $import_path..."

	export TOOL_MODULE_PATH="$import_path"
	export OUTPUT_DIR="$TOOLS_OUTPUT_DIR"
	export GOOS=$(go env GOOS)
	export GOARCH=$(go env GOARCH)
	export SKIP_BUILD="$SKIP_BUILD"

	# Build tool (failures allowed)
	source "$ROOT_DIR/scripts/build-tool.sh" || true

	# Add task definition if generating taskfiles
	if [ "$GENERATE_TASKFILES" = "true" ]; then
		echo "📝 Adding task for $TOOL_NAME..."
		cat <<EOF >>$output_taskfile
  ${TOOL_NAME}:
    desc: run ${TOOL_NAME} - built from ${TOOL_MODULE_PATH}
    cmds:
      - ${SCRIPTS_DIR}/run-tool.sh ${TOOL_NAME} {{.CLI_ARGS}}
EOF
	fi
}

# 🔍 Parse and build tools
echo "🔍 Scanning tools.go for imports..."
while IFS= read -r line; do
	if [[ $line =~ ^[[:space:]]*_[[:space:]]*\"(.+)\" ]]; then
		import_path="${BASH_REMATCH[1]}"
		build_tool "$import_path"
	fi
done <"$ROOT_DIR/tools/tools.go"

# 📋 Generate scripts taskfile if requested
if [ "$GENERATE_TASKFILES" = "true" ]; then
	echo "📝 Generating scripts taskfile..."
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
				echo "🔧 Making ${script} executable..."
				chmod +x ${SCRIPTS_DIR}/${script}
			fi

			script_name=${script%.sh}
			echo "📝 Adding task for $script_name..."
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
