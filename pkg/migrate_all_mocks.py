#!/usr/bin/env python3
"""Migrate all test files to use new local mocks."""

import re
from pathlib import Path

def update_test_file(file_path: Path):
    """Update mock imports and usages in a test file."""
    content = file_path.read_text()
    original_content = content
    changes_made = False

    # Replace mock usages
    replacements = [
        ('mocks.NewMockLoggerService', 'logger.NewMockLoggerService'),
        ('mocks.NewMockHookManager', 'hooks.NewMockHookManager'),
        ('mocks.NewMockConfigService', 'config.NewMockConfigService'),
        ('mocks.NewMockAgentRegistry', 'registry.NewMockAgentRegistry'),
        ('mocks.NewMockAgentFactory', 'shared.NewMockAgentFactory'),
        ('mocks.NewMockAgent', 'shared.NewMockAgent'),
        ('mocks.NewMockPromptManager', 'manager.NewMockPromptManager'),
        ('mocks.NewMockFileStateManager', 'state.NewMockFileStateManager'),
        ('mocks.NewMockSkillService', 'skills.NewMockSkillService'),
        ('mocks.NewMockService', 'workspace.NewMockService'),
        ('mocks.NewMockBus', 'events.NewMockBus'),
        ('*mocks.MockAgent', '*shared.MockAgent'),
        ('*mocks.MockAgentRegistry', '*registry.MockAgentRegistry'),
        ('*mocks.MockAgentFactory', '*shared.MockAgentFactory'),
        ('*mocks.MockConfigService', '*config.MockConfigService'),
        ('*mocks.MockHookManager', '*hooks.MockHookManager'),
        ('*mocks.MockLoggerService', '*logger.MockLoggerService'),
        ('*mocks.MockPromptManager', '*manager.MockPromptManager'),
        ('*mocks.MockFileStateManager', '*state.MockFileStateManager'),
        ('*mocks.MockSkillService', '*skills.MockSkillService'),
        ('*mocks.MockService', '*workspace.MockService'),
        ('*mocks.MockBus', '*events.MockBus'),
    ]

    for old, new in replacements:
        if old in content:
            content = content.replace(old, new)
            changes_made = True

    if not changes_made:
        return False

    # Add required imports
    import_map = {
        'logger.NewMockLoggerService': 'github.com/denkhaus/gollum/pkg/logger',
        'hooks.NewMockHookManager': 'github.com/denkhaus/gollum/pkg/hooks',
        'config.NewMockConfigService': 'github.com/denkhaus/gollum/pkg/config',
        'registry.NewMockAgentRegistry': 'github.com/denkhaus/gollum/pkg/registry',
        'shared.NewMockAgentFactory': 'github.com/denkhaus/gollum/pkg/shared',
        'shared.NewMockAgent': 'github.com/denkhaus/gollum/pkg/shared',
        'manager.NewMockPromptManager': 'github.com/denkhaus/gollum/pkg/prompt/manager',
        'state.NewMockFileStateManager': 'github.com/denkhaus/gollum/pkg/state',
        'skills.NewMockSkillService': 'github.com/denkhaus/gollum/pkg/skills',
        'workspace.NewMockService': 'github.com/denkhaus/gollum/pkg/workspace',
        'events.NewMockBus': 'github.com/denkhaus/gollum/pkg/events',
    }

    # Find existing imports
    imports_to_add = []
    for mock, import_path in import_map.items():
        if mock in content and import_path not in content:
            # Check if we need an alias
            if 'manager.NewMockPromptManager' in mock:
                imports_to_add.append(('\t"github.com/denkhaus/gollum/pkg/prompt/manager"', 'manager'))
            else:
                parts = import_path.split('/')
                alias = parts[-1]
                imports_to_add.append((f'\t"{import_path}"', alias))

    if not imports_to_add:
        file_path.write_text(content)
        return True

    # Find import block
    import_match = re.search(r'import \((.*?)\)', content, re.DOTALL)
    if not import_match:
        file_path.write_text(content)
        return True

    import_block = import_match.group(1)
    lines = import_block.split('\n')

    # Add new imports
    for imp, alias in imports_to_add:
        # Find insertion point (alphabetically)
        inserted = False
        for i, line in enumerate(lines):
            if line.strip() and line.strip() > imp.strip():
                lines.insert(i, imp)
                inserted = True
                break
        if not inserted:
            lines.append(imp)

    # Reconstruct import block
    new_import_block = 'import (\n' + '\n'.join(lines) + '\n)'
    content = re.sub(r'import \(.*?\)', new_import_block, content, flags=re.DOTALL)

    # Remove unused mocks import
    if '"github.com/denkhaus/gollum/pkg/mocks"' in content and 'mocks.' not in content and '*mocks.Mock' not in content:
        lines = content.split('\n')
        new_lines = []
        for line in lines:
            if '"github.com/denkhaus/gollum/pkg/mocks"' not in line:
                new_lines.append(line)
        content = '\n'.join(new_lines)

    file_path.write_text(content)
    return True

# Update all test files
pkg_dir = Path('/home/denkhaus/dev/gomodules/gollum/pkg')
count = 0
for test_file in pkg_dir.glob('**/*_test.go'):
    if update_test_file(test_file):
        count += 1
        print(f'Updated: {test_file.relative_to(pkg_dir.parent)}')

print(f'\nTotal files updated: {count}')
