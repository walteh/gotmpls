#!/bin/bash

set -euo pipefail

SKIP_BUILD="false"
GENERATE_TASKFILES="false"

# parse flags for skip build and generate taskfile
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

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: ${SCRIPTS_DIR:="${ROOT_DIR}/scripts"}
: ${TASKFILE_OUTPUT_DIR:="./out/taskfiles"}
: ${TOOLS_OUTPUT_DIR:="./out/tools"}

if [ "$SKIP_BUILD" = "false" ]; then
	rm -rf "$TOOLS_OUTPUT_DIR"
	mkdir -p "$TOOLS_OUTPUT_DIR"
fi

# if TASKFILE_OUTPUT_DIR is set, then we need to build the taskfile
if [ "$GENERATE_TASKFILES" = "true" ]; then
	rm -rf "$TASKFILE_OUTPUT_DIR"
	mkdir -p "$TASKFILE_OUTPUT_DIR"

	output_taskfile="$TASKFILE_OUTPUT_DIR/Taskfile.tools.yml"
	rm -f "$output_taskfile"

	cat <<EOF >$output_taskfile
version: '3'

# vars:
#   TOOLS_OUTPUT_DIR: $TOOLS_OUTPUT_DIR
#   SCRIPTS_DIR: $SCRIPTS_DIR

# includes:
#   script:
#     taskfile: "./Taskfile.scripts.yml"
#     # vars:
#     #   SCRIPTS_DIR: $SCRIPTS_DIR
#     internal: true


tasks:
EOF
fi

# Extract tool imports from tools.go
build_tool() {
	export TOOL_MODULE_PATH="$1"
	export OUTPUT_DIR="$TOOLS_OUTPUT_DIR"
	export GOOS=$(go env GOOS)
	export GOARCH=$(go env GOARCH)
	export SKIP_BUILD="$SKIP_BUILD"
	# if it fails, that's okay - we don't want any of the local builds to cause their requirements to fail
	source "$ROOT_DIR/scripts/build-tool.sh"

	if [ "$GENERATE_TASKFILES" = "true" ]; then
		cat <<EOF >>$output_taskfile
  ${TOOL_NAME}:
    desc: run ${TOOL_NAME} - built from ${TOOL_MODULE_PATH}
    cmds:
      - ${SCRIPTS_DIR}/run-tool.sh ${TOOL_NAME} {{.CLI_ARGS}}
EOF
	fi
}

# Parse tools.go to get the tool imports
while IFS= read -r line; do
	if [[ $line =~ ^[[:space:]]*_[[:space:]]*\"(.+)\" ]]; then
		import_path="${BASH_REMATCH[1]}"
		build_tool "$import_path"
	fi
done <"$ROOT_DIR/tools/tools.go"

# if SCRIPTS_DIR is set, and TASKFILE_OUTPUT_DIR is set, then we need to build the taskfile
if [ "$GENERATE_TASKFILES" = "true" ]; then
	output_file="${TASKFILE_OUTPUT_DIR}/Taskfile.scripts.yml"
	rm -f "$output_file"

	cat <<EOF >$output_file
version: '3'

# vars:
#   SCRIPTS_DIR: $SCRIPTS_DIR

tasks:
EOF

	for script in $(ls scripts); do
		# if script is the same as the current file, skip it
		if [[ $script == $(basename "$0") ]]; then
			continue
		fi
		# if it ends with sh, dont skip it
		if [[ $script == *.sh ]]; then

			# if the script is not executable, make it executable
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
