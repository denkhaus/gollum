#!/usr/bin/env python3
"""Remove unused mocks import from test files."""

import re
from pathlib import Path

def clean_test_file(file_path: Path):
    """Remove unused mocks import."""
    content = file_path.read_text()

    # Check if mocks import exists
    if '"github.com/denkhaus/gollum/pkg/mocks"' not in content:
        return False

    # Check if mocks is still used
    mocks_usage = re.search(r'mocks\.Mock[A-Z]', content)
    if mocks_usage:
        return False  # Still using mocks, don't remove import

    # Remove mocks import line
    lines = content.split('\n')
    new_lines = []
    for line in lines:
        if '"github.com/denkhaus/gollum/pkg/mocks"' not in line:
            new_lines.append(line)

    new_content = '\n'.join(new_lines)
    file_path.write_text(new_content)
    return True

# Clean all test files
tools_dir = Path('/home/denkhaus/dev/gomodules/gollum/pkg/tools')
count = 0
for test_file in tools_dir.glob('*_test.go'):
    if clean_test_file(test_file):
        count += 1
        print(f'Cleaned: {test_file.name}')

print(f'\nTotal files cleaned: {count}')
