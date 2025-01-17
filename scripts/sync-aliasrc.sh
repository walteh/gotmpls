#!/usr/bin/env bash

set -euo pipefail

# Check if script is being sourced (works in both bash and zsh)
SOURCED=0
if [ -n "${BASH_SOURCE[0]:-}" ] && [ "${BASH_SOURCE[0]}" != "${0}" ]; then
	SOURCED=1
elif [ -n "${ZSH_VERSION:-}" ] && [ "${ZSH_EVAL_CONTEXT:-}" = "toplevel" ]; then
	SOURCED=1
fi

# Parse args for shell override
SHELL_TYPE=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--shell)
		SHELL_TYPE="$2"
		shift 2
		;;
	*)
		echo "❌ Unknown option: $1"
		echo "Usage: $0 [--shell zsh|bash]"
		exit 1
		;;
	esac
done

# If no override, detect from $SHELL
if [ -z "$SHELL_TYPE" ]; then
	SHELL_TYPE=$(basename "$SHELL")
fi

# Validate shell type
case "$SHELL_TYPE" in
*zsh* | zsh)
	SHELL_TYPE="zsh"
	;;
*bash* | bash)
	SHELL_TYPE="bash"
	;;
*)
	echo "❌ Unsupported shell: $SHELL_TYPE (only zsh and bash are supported)"
	echo "Override with: $0 --shell zsh"
	exit 1
	;;
esac

echo "🔍 Using shell: $SHELL_TYPE"

# Get script directory (works in both bash and zsh)
if [ -n "${BASH_SOURCE:-}" ]; then
	SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
	SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
fi

PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_DIR="${HOME}/.config/aliasrc"

# Verify the shell config exists
SHELL_CONFIG="$PROJECT_ROOT/.aliasrc/shell-configs/aliasrc.$SHELL_TYPE"
if [ ! -f "$SHELL_CONFIG" ]; then
	echo "❌ Shell config not found: $SHELL_CONFIG"
	echo "Expected config for: $SHELL_TYPE"
	echo "Looking in: $PROJECT_ROOT/.aliasrc/shell-configs/"
	ls -la "$PROJECT_ROOT/.aliasrc/shell-configs/" 2>/dev/null || echo "Directory not found!"
	exit 1
fi

# Create config directory if it doesn't exist
mkdir -p "$CONFIG_DIR"

# Copy the shell config
echo "📝 Installing aliasrc for $SHELL_TYPE..."
cp "$SHELL_CONFIG" "$CONFIG_DIR/aliasrc.$SHELL_TYPE"
chmod +x "$CONFIG_DIR/aliasrc.$SHELL_TYPE"

# Check if config is already sourced
RC_FILE="${HOME}/.${SHELL_TYPE}rc"
SOURCE_LINE="source ~/.config/aliasrc/aliasrc.${SHELL_TYPE}"

if grep -q "aliasrc.${SHELL_TYPE}" "$RC_FILE" 2>/dev/null; then
	echo "✅ Config already sourced in $RC_FILE"
else
	echo ""
	echo "🔄 to enable on every shell start, add this line to your $RC_FILE:"
	echo "💪 to enable now, run this line in your current shell:"
	echo ""
	echo "    $SOURCE_LINE"

	echo ""
fi
