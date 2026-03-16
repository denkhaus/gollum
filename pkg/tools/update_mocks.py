#!/usr/bin/env python3
"""Update mock imports in test files."""

import re
from pathlib import Path

def update_test_file(file_path: Path):
    """Update mock imports in a test file."""
    content = file_path.read_text()

    # Check if file needs updating
    if 'hooks.NewMockHookManager' not in content:
        return False

    # Check if hooks import already exists
    if '"github.com/denkhaus/gollum/pkg/hooks"' in content:
        return False

    # Find the import block
    import_match = re.search(r'import \((.*?)\)', content, re.DOTALL)
    if not import_match:
        return False

    import_block = import_match.group(1)
    # Keep original line structure
    lines = import_block.split('\n')
    new_lines = []

    for line in lines:
        new_lines.append(line)
        # Add hooks import after mocks import
        if '"github.com/denkhaus/gollum/pkg/mocks"' in line:
            # Find indentation from mocks line
            indent = len(line) - len(line.lstrip())
            new_lines.append(' ' * indent + '\t"github.com/denkhaus/gollum/pkg/hooks"')

    # Reconstruct import block
    new_import_block = 'import (\n' + '\n'.join(new_lines) + '\n)'

    # Replace import block
    new_content = re.sub(r'import \(.*?\)', new_import_block, content, flags=re.DOTALL)

    # Replace mocks.NewMockHookManager with hooks.NewMockHookManager
    new_content = new_content.replace('mocks.NewMockHookManager', 'hooks.NewMockHookManager')

    file_path.write_text(new_content)
    return True

# Update all test files
tools_dir = Path('/home/denkhaus/dev/gomodules/gollum/pkg/tools')
count = 0
for test_file in tools_dir.glob('*_test.go'):
    if update_test_file(test_file):
        count += 1
        print(f'Updated: {test_file.name}')

print(f'\nTotal files updated: {count}')
