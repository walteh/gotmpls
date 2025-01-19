#!/bin/bash

# 📚 Documentation
# ===============
# This script syncs Go files from a GitHub repository and applies custom transformations
#
# Features:
# 🔍 Downloads specific Go files from a GitHub repository
# 🛠️ Applies custom string replacements
# 📦 Adds package declarations if missing
# ✨ Handles nested directory structures
#
# Usage:
#   ./sync-go-files.sh <github-url> <destination-dir> [--string-to-replace <pattern>]
#
# Arguments:
#   github-url           : GitHub URL in format github.com/ORG/REPO/blob/BRANCH/PATH (required)
#   destination-dir      : Local directory to sync files to (required)
#   --string-to-replace : Pattern to replace in format 'old:new' (optional, multiple allowed)
#
# Example:
#   ./sync-go-files.sh \
#     github.com/golang/tools/blob/master/gopls/internal/protocol/generate \
#     ./pkg/lsp/generator \
#     --string-to-replace 'func main():func main_original()'

set -euo pipefail

# 🔍 Validate required arguments
if [ "$#" -lt 2 ]; then

	echo "❌ Missing required arguments"
	echo "📖 Usage: $0 <github-url> <destination-dir> [--string-to-replace <pattern>]"
	echo "   Example: $0 github.com/golang/tools/blob/master/gopls/internal/protocol/generate ./pkg/lsp/generator"
	exit 1
fi

# 🎯 Parse initial arguments
GITHUB_URL=$1
DEST_DIR=$2
shift 2

# 🔄 Parse GitHub URL components
if [[ ! "$GITHUB_URL" =~ ^github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$ ]]; then
	echo "❌ Invalid GitHub URL format"
	echo "📖 Expected: github.com/ORG/REPO/blob/BRANCH/PATH"
	exit 1
fi

ORG="${BASH_REMATCH[1]}"
REPO="${BASH_REMATCH[2]}"
BRANCH="${BASH_REMATCH[3]}"
SOURCE_PATH="${BASH_REMATCH[4]}"

# 📝 Store string replacements
STRINGS_TO_REPLACE=()
FILES_TO_IGNORE=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	--string-to-replace)
		if [[ $# -lt 2 ]]; then
			echo "❌ Error: --string-to-replace requires a pattern"
			exit 1
		fi
		STRINGS_TO_REPLACE+=("$2")
		shift 2
		;;
	--file-to-ignore)
		if [[ $# -lt 2 ]]; then
			echo "❌ Error: --file-to-ignore requires a pattern"
			exit 1
		fi
		FILES_TO_IGNORE+=("$2")
		shift 2
		;;
	*)
		echo "❌ Unknown argument: $1"
		exit 1
		;;
	esac
done

# 📋 Show configuration
echo "🔄 Syncing files from github.com/$ORG/$REPO"
echo "├── 🌿 Branch: $BRANCH"
echo "├── 📂 Source: $SOURCE_PATH"
echo "├── 🎯 Destination: $DEST_DIR"
echo "├── 🔧 Replacements: ${STRINGS_TO_REPLACE[*]:-none}"
echo "└── 🫥 Files to ignore: ${FILES_TO_IGNORE[*]:-none}"
# 📁 Create destination directory
mkdir -p "$DEST_DIR"

# 📥 Get list of .go files from the directory
echo "🔍 Fetching file list..."
FILES=$(curl -s "https://api.github.com/repos/$ORG/$REPO/contents/$SOURCE_PATH?ref=$BRANCH" | grep "\"path\"" | cut -d '"' -f 4)

# 🔍 Filter out files to ignore
echo "🔍 Filtering out files to ignore..."
for file in "${FILES_TO_IGNORE[@]}"; do
	FILES=$(echo "$FILES" | grep -vE "$file")
done

if [ -z "$FILES" ]; then
	echo "❌ No files found in the specified directory"
	exit 1
fi

# 📦 Download and process each file
for file in $FILES; do
	filename=$(basename "$file")
	echo "📥 Processing $filename..."

	# Download file
	if ! curl -fL --progress-bar "https://raw.githubusercontent.com/$ORG/$REPO/$BRANCH/$file" -o "$DEST_DIR/$filename"; then
		echo "❌ Failed to download $filename"
		rm -f "$DEST_DIR/$filename"
		exit 1
	fi

	# Verify file
	if [ ! -s "$DEST_DIR/$filename" ]; then
		echo "❌ Downloaded file $filename is empty"
		rm -f "$DEST_DIR/$filename"
		exit 1
	fi

	# Add package declaration if missing
	if ! grep -q "^package" "$DEST_DIR/$filename"; then
		PACKAGE_NAME=$(basename "$DEST_DIR")
		echo "📝 Adding package declaration: $PACKAGE_NAME"
		sed -i.bak "1s;^;package ${PACKAGE_NAME}\n\n;" "$DEST_DIR/$filename"
		rm -f "$DEST_DIR/$filename.bak"
	fi

	# Apply string replacements
	for pattern in "${STRINGS_TO_REPLACE[@]}"; do
		if [[ "$pattern" =~ ^([^:]+):([^:]*)$ ]]; then
			old="${BASH_REMATCH[1]}"
			new="${BASH_REMATCH[2]}"
			echo "🔧 Replacing '$old' with '$new'"
			sed -i.bak "s|$old|$new|g" "$DEST_DIR/$filename"
			rm -f "$DEST_DIR/$filename.bak"
		else
			echo "⚠️  Invalid replacement pattern: '$pattern', skipping"
		fi
	done
done

echo "✅ Successfully synced all Go files to $DEST_DIR"
