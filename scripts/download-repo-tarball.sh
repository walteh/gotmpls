#!/bin/bash

# 📚 Documentation
# ===============
# This script downloads a GitHub repository as a tarball and sets up Go embedding
#
# Features:
# 🔍 Validates required arguments
# 🛠️ Handles optional arguments with defaults
# 📥 Downloads repository tarball
# ✍️ Creates embed.go file for Go embedding
# 🎨 Provides colorful status messages
#
# Usage:
#   ./download-repo-tarball.sh --repo REPO --org ORG --ref REF [--pkg PKG] [--path PATH]
#
# Arguments:
#   --repo  : Repository name (required)
#   --org   : GitHub organization (required)
#   --ref   : Git reference (tag/branch/commit) (required)
#   --pkg   : Go package name (optional, defaults to lowercase repo name without hyphens)
#   --path  : Output directory path (optional, defaults to gen/git-repo-tarballs)
#
# Example:
#   ./download-repo-tarball.sh \
#     --repo "my-repo" \
#     --org "my-org" \
#     --ref "tags/v1.0.0" \
#     --pkg "myrepo" \
#     --path "gen/tarballs"

set -euo pipefail

# 🎯 Default values
REPO=""
ORG=""
REF=""
PKG=""
OUTPUT_PATH="gen/git-repo-tarballs"

# 🔄 Parse command line arguments using flags for better readability
while (("$#")); do
    case "$1" in
    --repo)
        REPO="$2"
        shift 2
        ;;
    --org)
        ORG="$2"
        shift 2
        ;;
    --ref)
        REF="$2"
        shift 2
        ;;
    --pkg)
        PKG="$2"
        shift 2
        ;;
    --path)
        OUTPUT_PATH="$2"
        shift 2
        ;;
    *)
        echo "❌ Unknown argument: $1"
        echo "📖 Use --help for usage information"
        exit 1
        ;;
    esac
done

# 🔍 Validate required arguments
if [[ -z "$REPO" || -z "$ORG" || -z "$REF" ]]; then
    echo "❌ Missing required arguments"
    echo "📖 Usage: $0 --repo REPO --org ORG --ref REF [--pkg PKG] [--path PATH]"
    exit 1
fi

# 🎲 If PKG is not provided, generate it from REPO
if [[ -z "$PKG" ]]; then
    PKG=$(echo "$REPO" | tr '[:upper:]' '[:lower:]' | tr -d '-')
fi

# 📁 Create output directory
mkdir -p "$OUTPUT_PATH/$REPO"

# 📥 Download tarball
TARBALL_PATH="$OUTPUT_PATH/$REPO/$REPO.tar.gz"
rm -f "$TARBALL_PATH"

echo "📦 Downloading $ORG/$REPO@$REF..."
if ! curl -fL --progress-bar "https://github.com/$ORG/$REPO/archive/refs/$REF.tar.gz" -o "$TARBALL_PATH"; then
    echo "❌ Failed to download $REPO"
    rm -f "$TARBALL_PATH"
    exit 1
fi

if [[ ! -s "$TARBALL_PATH" ]]; then
    echo "❌ Downloaded file is empty"
    rm -f "$TARBALL_PATH"
    exit 1
fi

# Get file size in KB
SIZE=$(du -k "$TARBALL_PATH" | cut -f1)
echo "⬇️  Downloaded $(printf "%'d" $SIZE) KB"

# 📝 Create embed.go file
EMBED_PATH="$OUTPUT_PATH/$REPO/embed.go"
cat >"$EMBED_PATH" <<EOF
package $PKG

import _ "embed"

//go:embed $REPO.tar.gz
var Data []byte
var Ref string = "$REF"
EOF

echo "✅ Successfully downloaded $REPO and created embed.go"
