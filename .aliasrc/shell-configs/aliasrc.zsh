#!/usr/bin/env zsh

# Prevent output when being sourced
if [[ $- != *i* ]]; then
	exec 1>/dev/null 2>&1
fi

{ # Start of redirected block

	# Function to find and execute local alias
	find_local_alias() {
		local cmd=$1
		shift
		local dir=$PWD

		while [[ "$dir" != "/" ]]; do
			if [[ -x "$dir/.aliasrc/$cmd" ]]; then
				"$dir/.aliasrc/$cmd" "$@"
				return $?
			fi
			dir=$(dirname "$dir")
		done

		# If ALIASRC_BYPASS is set, return 1 to indicate no alias found
		if [ -n "${ALIASRC_BYPASS:-}" ]; then
			return 1
		fi

		"$cmd" "$@"
	}

	# Enhanced which command that shows both aliasrc and system commands
	which() {
		local cmd=$1
		local found=false
		local dir=$PWD

		# Look for aliasrc version
		while [[ "$dir" != "/" ]]; do
			if [[ -x "$dir/.aliasrc/$cmd" ]]; then
				echo "🔄 aliasrc: $dir/.aliasrc/$cmd"
				found=true
				break
			fi
			dir=$(dirname "$dir")
		done

		# Look for system version
		if system_cmd=$(command which "$cmd" 2>/dev/null); then
			echo "💻  system: $system_cmd (set ALIASRC_BYPASS=1 to use system command)"
			found=true
		fi

		if [[ "$found" == "false" ]]; then
			# fallback to system command
			command which "$cmd"
		fi
	}

	# Hook function that runs before each command
	__aliasrc_hook() {
		# Skip if no command
		[ -z "$1" ] && return

		# Get the first word (command) without using arrays
		local cmd="${1%% *}"

		# Skip for complex commands or if ALIASRC_BYPASS is set
		if [[ -n "${ALIASRC_BYPASS:-}" ]] ||
			[[ "$1" == *"|"* || "$1" == *">"* || "$1" == *"<"* || "$1" == *"&"* ]] ||
			type "$cmd" 2>/dev/null | grep -q "builtin\|alias\|function"; then
			return
		fi

		# Get arguments (everything after the first word)
		local args=""
		if [[ "$1" == *" "* ]]; then
			args="${1#* }"
		fi

		find_local_alias "$cmd" $args
	}

	# Export the which function so it's available in subshells
	export -f which find_local_alias __aliasrc_hook

	# ZSH-specific setup
	if typeset -f add-zsh-hook >/dev/null; then
		autoload -Uz add-zsh-hook
		add-zsh-hook preexec __aliasrc_hook
	else
		# Fallback for ZSH without add-zsh-hook
		if [[ -z "${preexec_functions}" ]]; then
			preexec_functions=()
		fi
		preexec_functions+=(__aliasrc_hook)
	fi

} >/dev/null 2>&1 # End of redirected block
