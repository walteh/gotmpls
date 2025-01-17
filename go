#!/usr/bin/env bash
set -euo pipefail

# if first argument is "test", use gotestsum
if [ "${1:-}" == "test" ]; then
	shift
	./scripts/run-tool.sh gotestsum \
		--format testname \
		--format-icons hivis \
		--hide-summary=skipped \
		--raw-command -- go test -json -cover "$@"
	exit $?
fi

# otherwise run go directly with all arguments
exec go "$@"
