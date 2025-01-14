#!/bin/bash

set -euo pipefail

# Check if we have the required arguments
if [ "$#" -lt 2 ]; then
	echo "Usage: $0 <github-url> <destination-dir> [--string-to-replace <pattern>]"
	echo "Example: $0 github.com/golang/tools/blob/master/gopls/internal/protocol/generate ./pkg/lsp/generator --string-to-replace 'func main():func main_original()'"
	exit 1
fi

GITHUB_URL=$1
DEST_DIR=$2
shift 2

# Parse GitHub URL components
# github.com/ORG/REPO/blob/BRANCH/PATH -> ORG, REPO, BRANCH, PATH
if [[ ! "$GITHUB_URL" =~ ^github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$ ]]; then
	echo "Invalid GitHub URL format. Expected: github.com/ORG/REPO/blob/BRANCH/PATH"
	exit 1
fi

ORG="${BASH_REMATCH[1]}"
REPO="${BASH_REMATCH[2]}"
BRANCH="${BASH_REMATCH[3]}"
SOURCE_PATH="${BASH_REMATCH[4]}"

# Store string replacements
STRINGS_TO_REPLACE=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	--string-to-replace)
		if [[ $# -lt 2 ]]; then
			echo "Error: --string-to-replace requires a pattern"
			exit 1
		fi
		STRINGS_TO_REPLACE+=("$2")
		shift 2
		;;
	*)
		echo "Unknown argument: $1"
		exit 1
		;;
	esac
done

echo "Syncing files from github.com/$ORG/$REPO"
echo "Branch: $BRANCH"
echo "Source path: $SOURCE_PATH"
echo "Destination: $DEST_DIR"
echo "String replacements: ${STRINGS_TO_REPLACE[*]:-none}"

# Create destination directory
mkdir -p "$DEST_DIR"

# Get list of .go files from the directory
FILES=$(curl -s "https://api.github.com/repos/$ORG/$REPO/contents/$SOURCE_PATH?ref=$BRANCH" | grep "\"path\"" | grep "\.go\"" | cut -d '"' -f 4)

if [ -z "$FILES" ]; then
	echo "No .go files found in the specified directory"
	exit 1
fi

# Download each .go file
for file in $FILES; do
	filename=$(basename "$file")
	echo "Downloading $filename..."

	# Download file
	if ! curl -fL "https://raw.githubusercontent.com/$ORG/$REPO/$BRANCH/$file" -o "$DEST_DIR/$filename"; then
		echo "Failed to download $filename"
		rm -f "$DEST_DIR/$filename"
		exit 1
	fi

	# Check if file is empty
	if [ ! -s "$DEST_DIR/$filename" ]; then
		echo "Downloaded file $filename is empty"
		rm -f "$DEST_DIR/$filename"
		exit 1
	fi

	# Add package declaration if it doesn't exist
	if ! grep -q "^package" "$DEST_DIR/$filename"; then
		PACKAGE_NAME=$(basename "$DEST_DIR")
		sed -i.bak "1s;^;package ${PACKAGE_NAME}\n\n;" "$DEST_DIR/$filename"
		rm -f "$DEST_DIR/$filename.bak"
	fi

	# Apply string replacements
	for pattern in "${STRINGS_TO_REPLACE[@]}"; do
		if [[ "$pattern" =~ ^([^:]+):([^:]+)$ ]]; then
			old="${BASH_REMATCH[1]}"
			new="${BASH_REMATCH[2]}"
			echo "Replacing '$old' with '$new' in $filename"
			sed -i.bak "s|$old|$new|g" "$DEST_DIR/$filename"
			rm -f "$DEST_DIR/$filename.bak"
		else
			echo "Warning: Invalid replacement pattern '$pattern', skipping"
		fi
	done
done

echo "Successfully synced all Go files to $DEST_DIR"
