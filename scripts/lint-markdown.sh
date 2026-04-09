#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

MODE="${1:-check}"

cd "$PROJECT_ROOT"

# Install markdownlint-cli if not available
if ! command -v markdownlint &> /dev/null; then
    echo "Installing markdownlint-cli..."
    npm install -g markdownlint-cli
fi

# Install prettier if not available
if ! command -v npx &> /dev/null; then
    echo "Error: npx is required for prettier. Please install Node.js"
    exit 1
fi

# Get only git-tracked markdown files
MARKDOWN_FILES=$(git ls-files "*.md" | tr '\n' ' ')

if [ -z "$MARKDOWN_FILES" ]; then
    echo "No markdown files found in git repository"
    exit 0
fi

case "$MODE" in
    "check" | "ci")
        echo "🔍 Checking Markdown files..."
        echo "Files to check: $MARKDOWN_FILES"
        markdownlint $MARKDOWN_FILES
        echo "✅ Markdown files are properly formatted"
        ;;
    "fix")
        echo "🔧 Fixing Markdown files..."
        echo "Files to fix: $MARKDOWN_FILES"
        
        # Function to preserve <pre> sections during prettier formatting
        format_markdown_with_pre_protection() {
            local file="$1"
            local temp_file="${file}.tmp"
            local pre_markers_file="${file}.pre_markers"
            
            # Create a unique marker for each <pre> block
            python3 - "$file" "$temp_file" "$pre_markers_file" << 'EOF'
import sys
import re
import json

file_path = sys.argv[1]
temp_file = sys.argv[2]
markers_file = sys.argv[3]

with open(file_path, 'r') as f:
    content = f.read()

# Find all <pre>...</pre> blocks and replace with markers
pre_blocks = {}
counter = 0

def replace_pre(match):
    global counter
    marker = f"__PRETTIER_PRE_BLOCK_{counter}__"
    pre_blocks[marker] = match.group(0)
    counter += 1
    return marker

# Replace <pre> blocks with markers
modified_content = re.sub(r'<pre>.*?</pre>', replace_pre, content, flags=re.DOTALL)

# Write modified content to temp file
with open(temp_file, 'w') as f:
    f.write(modified_content)

# Save markers mapping
with open(markers_file, 'w') as f:
    json.dump(pre_blocks, f)
EOF
            
            # Run prettier on the modified file
            npx prettier --parser markdown --prose-wrap always --print-width 120 --write "$temp_file"
            
            # Restore <pre> blocks
            python3 - "$temp_file" "$pre_markers_file" "$file" << 'EOF'
import sys
import json

temp_file = sys.argv[1]
markers_file = sys.argv[2]
original_file = sys.argv[3]

with open(temp_file, 'r') as f:
    content = f.read()

with open(markers_file, 'r') as f:
    pre_blocks = json.load(f)

# Restore <pre> blocks
for marker, pre_block in pre_blocks.items():
    content = content.replace(marker, pre_block)

# Write back to original file
with open(original_file, 'w') as f:
    f.write(content)
EOF
            
            # Clean up temp files
            rm -f "$temp_file" "$pre_markers_file"
        }
        
        # Format each markdown file with <pre> protection
        for file in $MARKDOWN_FILES; do
            echo "  Formatting $file with prettier (preserving <pre> sections)..."
            format_markdown_with_pre_protection "$file"
        done
        
        # Then run markdownlint to fix any remaining issues
        echo "Running markdownlint --fix..."
        markdownlint $MARKDOWN_FILES --fix
        echo "✅ Markdown files have been fixed"
        ;;
    *)
        echo "Usage: $0 [check|fix|ci]"
        exit 1
        ;;
esac
