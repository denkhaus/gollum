// Package diff provides unit tests for diff generation and formatting services.
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	provider, err := NewProvider(nil)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.IsType(t, &providerImpl{}, provider)
}

func TestProvider_GenerateDiff(t *testing.T) {
	tests := []struct {
		name         string
		oldPath      string
		newPath      string
		oldContent   string
		newContent   string
		wantContains string
	}{
		{
			name:         "simple change",
			oldPath:      "old.txt",
			newPath:      "new.txt",
			oldContent:   "line1\nline2\nline3",
			newContent:   "line1\nline2 modified\nline3",
			wantContains: "line2 modified",
		},
		{
			name:         "addition",
			oldPath:      "a.txt",
			newPath:      "b.txt",
			oldContent:   "line1\nline2",
			newContent:   "line1\nline2\nline3",
			wantContains: "+line3",
		},
		{
			name:         "deletion",
			oldPath:      "before.txt",
			newPath:      "after.txt",
			oldContent:   "line1\nline2\nline3",
			newContent:   "line1\nline3",
			wantContains: "-line2",
		},
		{
			name:         "no changes",
			oldPath:      "same.txt",
			newPath:      "same.txt",
			oldContent:   "line1\nline2\nline3",
			newContent:   "line1\nline2\nline3",
			wantContains: "", // Empty diff when no changes
		},
		{
			name:         "complete replacement",
			oldPath:      "old.txt",
			newPath:      "new.txt",
			oldContent:   "old content",
			newContent:   "new content",
			wantContains: "new content",
		},
		{
			name:         "empty old content",
			oldPath:      "/dev/null",
			newPath:      "new.txt",
			oldContent:   "",
			newContent:   "new line",
			wantContains: "+new line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(nil)
			require.NoError(t, err)

			got, err := provider.GenerateDiff(tt.oldPath, tt.newPath, tt.oldContent, tt.newContent)
			require.NoError(t, err)

			if tt.wantContains == "" {
				// For no changes, diff might be empty or minimal
				assert.NotContains(t, got, "@@")
			} else {
				assert.Contains(t, got, tt.wantContains)
			}
		})
	}
}

func TestProvider_GenerateDiffForNewFile(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		content      string
		wantContains string
		expectEmpty  bool
	}{
		{
			name:         "new file with content",
			path:         "newfile.txt",
			content:      "line1\nline2\nline3",
			wantContains: "+line1",
			expectEmpty:  false,
		},
		{
			name:         "new empty file",
			path:         "empty.txt",
			content:      "",
			wantContains: "",
			expectEmpty:  true,
		},
		{
			name:         "new file with single line",
			path:         "single.txt",
			content:      "single line",
			wantContains: "+single line",
			expectEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(nil)
			require.NoError(t, err)

			got, err := provider.GenerateDiffForNewFile(tt.path, tt.content)
			require.NoError(t, err)

			if tt.expectEmpty {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
				assert.Contains(t, got, tt.wantContains)
				// Should reference /dev/null as old path when there's content
				assert.Contains(t, got, "/dev/null")
			}
		})
	}
}

func TestProvider_FormatForDisplay(t *testing.T) {
	tests := []struct {
		name  string
		diff  string
		empty bool
	}{
		{
			name:  "empty diff",
			diff:  "",
			empty: true,
		},
		{
			name: "simple diff",
			diff: `--- old.txt
+++ new.txt
@@ -1,3 +1,3 @@
 line1
-line2
+line2 modified
 line3`,
			empty: false,
		},
		{
			name: "diff with additions and deletions",
			diff: `--- a.go
+++ b.go
@@ -1,5 +1,5 @@
 package main

-func oldFunc() {}
+func newFunc() {}
 func helper() {}`,
			empty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(nil)
			require.NoError(t, err)

			got := provider.FormatForDisplay(tt.diff)

			if tt.empty {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestProvider_FormatCompact(t *testing.T) {
	tests := []struct {
		name  string
		diff  string
		empty bool
	}{
		{
			name:  "empty diff",
			diff:  "",
			empty: true,
		},
		{
			name: "simple diff for compact",
			diff: `--- old.txt
+++ new.txt
@@ -1,3 +1,3 @@
 line1
-line2
+line2 modified
 line3`,
			empty: false,
		},
		{
			name: "multi-hunk diff",
			diff: `--- old.txt
+++ new.txt
@@ -1,3 +1,3 @@
-old line 1
+new line 1
 context
@@ -5,7 +5,7 @@
-old line 2
+new line 2
 more context`,
			empty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(nil)
			require.NoError(t, err)

			got := provider.FormatCompact(tt.diff)

			if tt.empty {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
				// Compact should be shorter or equal to original
				assert.LessOrEqual(t, len(got), len(tt.diff)+100) // Allow some styling overhead
			}
		})
	}
}

func TestDiffStyler_styleDiff(t *testing.T) {
	styler := newDiffStyler()

	tests := []struct {
		name     string
		diff     string
		contains []string
	}{
		{
			name: "basic diff with all line types",
			diff: `--- old.txt
+++ new.txt
@@ -1,3 +1,3 @@
-line2
+line2 new
 context`,
			contains: nil,
		},
		{
			name:     "empty diff",
			diff:     "",
			contains: nil,
		},
		{
			name: "diff with metadata",
			diff: `diff --git a/file.txt b/file.txt
index 123..456 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
+new`,
			contains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := styler.styleDiff(tt.diff)

			if tt.diff == "" {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
				// Lipgloss styling is environment-dependent
				// Just verify content is returned, whether styled or not
				assert.NotEqual(t, got, "")
			}
		})
	}
}

func TestGetUnifiedDiff(t *testing.T) {
	tests := []struct {
		name       string
		oldPath    string
		newPath    string
		oldContent string
		newContent string
		wantEmpty  bool
	}{
		{
			name:       "identical content",
			oldPath:    "a.txt",
			newPath:    "b.txt",
			oldContent: "same content",
			newContent: "same content",
			wantEmpty:  true,
		},
		{
			name:       "different content",
			oldPath:    "a.txt",
			newPath:    "b.txt",
			oldContent: "old",
			newContent: "new",
			wantEmpty:  false,
		},
		{
			name:       "empty contents",
			oldPath:    "empty1.txt",
			newPath:    "empty2.txt",
			oldContent: "",
			newContent: "",
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getUnifiedDiff(tt.oldPath, tt.newPath, tt.oldContent, tt.newContent)

			if tt.wantEmpty {
				// Empty or minimal diff (no hunks)
				assert.NotContains(t, got, "@@")
			} else {
				assert.NotEmpty(t, got)
			}
		})
	}
}

// Benchmark tests
func BenchmarkProvider_GenerateDiff(b *testing.B) {
	oldContent := `line1
line2
line3
line4
line5`
	newContent := `line1
line2 modified
line3
line4
line5`

	provider, _ := NewProvider(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GenerateDiff("old.txt", "new.txt", oldContent, newContent)
	}
}

func BenchmarkProvider_FormatForDisplay(b *testing.B) {
	diff := `--- old.txt
+++ new.txt
@@ -1,5 +1,5 @@
 line1
-line2 old
+line2 new
 line3
-line4 old
+line4 new
 line5`

	provider, _ := NewProvider(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.FormatForDisplay(diff)
	}
}

func BenchmarkProvider_FormatCompact(b *testing.B) {
	diff := `--- old.txt
+++ new.txt
@@ -1,5 +1,5 @@
 line1
-line2 old
+line2 new
 line3
-line4 old
+line4 new
 line5`

	provider, _ := NewProvider(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.FormatCompact(diff)
	}
}
