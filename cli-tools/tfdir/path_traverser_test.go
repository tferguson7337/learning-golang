package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
)

type TestFileInfo struct {
	name    string      // suffix of the filename - to be joined with t.TempDir
	mode    os.FileMode // file mode (permissions, potential symlink)
	content string      // content of file, or symlink target if applicable.
}

type ExpectedRootInfo struct {
	name string // name of the dir
	visitDueToSymlink bool // indicates if the root was visited due to a symlink, enables skipping check 
}

type ExpectedTestFileInfo struct {
	name string      // name of the file or directory
	mode os.FileMode // file mode (permissions, dir or symlink)
	size int         // size of the file, 0 for directories and symlinks
	target *ExpectedTestFileInfo // target of the symlink, empty for non-symlinks
}

const WIN_ERROR_PRIVILEGE_NOT_HELD syscall.Errno = 1314

// Builds out the test filesystem layout based on the provided directory structure
// Returns error if any file or directory creation fails, and a boolean indicating whether symlink checks should be skipped due to insufficient privileges.
func buildTestFSLayout(
	t *testing.T,
	tempDir *string,
	dirStructure *map[string][]TestFileInfo) (error, bool) {
	skipSymlinkChecks := false
	for subdir, files := range *dirStructure {
		subdirPath := filepath.Join(*tempDir, subdir)
		err := os.MkdirAll(subdirPath, 0777)
		if err != nil {
			t.Fatalf("Failed to create subdir %s: %v", subdirPath, err)
			return err, skipSymlinkChecks
		}
		for _, tfi := range files {
			filePath := filepath.Join(subdirPath, tfi.name)
			if tfi.mode&os.ModeSymlink != 0 {
				targetPath := filepath.Join(*tempDir, tfi.content)
				err := os.Symlink(targetPath, filePath)
				if err != nil {
					if errors.Is(err, os.ErrPermission) || errors.Is(err, WIN_ERROR_PRIVILEGE_NOT_HELD) {
						skipSymlinkChecks = true
						continue
					}
					t.Fatalf("Failed to create symlink %s -> %s: %v", filePath, tfi.content, err)
					return err, skipSymlinkChecks
				}
				continue
			}

			// Non-symlink: regular file or directory
			f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, tfi.mode)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", filePath, err)
				return err, skipSymlinkChecks
			}
			if len(tfi.content) > 0 {
				wlen, err := f.WriteString(tfi.content)
				if err != nil {
					t.Fatalf("Failed to write to file %s: %v", filePath, err)
					return err, skipSymlinkChecks
				}
				if wlen != len(tfi.content) {
					err = fmt.Errorf("%v: written length %d does not match expected length %d\n", tfi.name, wlen, len(tfi.content))
					t.Fatalf("%v", err.Error())
					return err, skipSymlinkChecks
				}
			}
			f.Close()
		}
	}

	return nil, skipSymlinkChecks
}

// Destroys the test filesystem layout created under the given temporary directory
func cleanupTestFSLayout(tempDir *string) error {
	return os.RemoveAll(*tempDir)
}

func filterExpectedRoots(
	expectedRoots []ExpectedRootInfo,
	skipSymlinkChecks bool) []ExpectedRootInfo {
	if !skipSymlinkChecks {
		return expectedRoots
	}

	filtered := []ExpectedRootInfo{}
	for _, root := range expectedRoots {
		if skipSymlinkChecks && root.visitDueToSymlink {
			continue
		}
		filtered = append(filtered, root)
	}
	return filtered
}

func TestPathTraverser(t *testing.T) {
	tests := []struct {
		name string

		// Traverser input parameters
		traverserFileName string // Only used in single file tests, otherwise t.TempDir() is used as root
		recursionDepth    uint
		exclusionPattern  *regexp.Regexp
		traverseSymlinks  bool

		// Test directory structure
		dirStructure map[string][]TestFileInfo // (t.TempDir subdirectory -> list of filenames in that subdirectory)

		// Expected elements visited
		expectedRootsVisited    []ExpectedRootInfo
		expectedElementsVisitedNoSymlinkChecks *[]ExpectedTestFileInfo
		expectedElementsVisitedWithSymlinkChecks *[]ExpectedTestFileInfo
	}{
		// TODO: Define test cases
		// - Empty dir
		{
			name:                    "Empty dir",
			traverserFileName:       "",
			recursionDepth:          0,
			exclusionPattern:        nil,
			traverseSymlinks:        false,
			dirStructure:            map[string][]TestFileInfo{},
			expectedRootsVisited:    []ExpectedRootInfo{{name: ""}}, // "" for t.TempDir
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{},
		},

		// - Direct file
		{
			name:              "Direct file",
			traverserFileName: "file.txt",
			recursionDepth:    0,
			exclusionPattern:  nil,
			traverseSymlinks:  false,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name: "file.txt",
						mode: 0666,
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: "file.txt"},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file.txt",
					mode: 0666,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file.txt",
					mode: 0666,
				},
			},
		},

		// - Single dir, with files
		{
			name:              "Single dir with files",
			traverserFileName: "",
			recursionDepth:    0,
			exclusionPattern:  nil,
			traverseSymlinks:  false,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
			},
		},

		// - Nested dirs, with files, no recursion
		{
			name:              "Nested dirs with files, no recursion",
			traverserFileName: "",
			recursionDepth:    0,
			exclusionPattern:  nil,
			traverseSymlinks:  false,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
				},
				"subdir": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					// We visit subdir as an element of the root dir, but we don't traverse it
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					// We visit subdir as an element of the root dir, but we don't traverse it
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
		},

		// - Nested dirs, with files, with recursion
		{
			name:              "Nested dirs with files, with recursion",
			traverserFileName: "",
			recursionDepth:    1,
			exclusionPattern:  nil,
			traverseSymlinks:  false,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
				},
				"subdir1": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir1\\subdir2": {},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
				{name: "subdir1"},
				// no subdir1\subdir2, since it's beyond the recursion depth
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
		},

		// - Symlinks, without follow-flag, no recursion
		{
			name:              "Symlinks, without follow-flag, no recursion",
			traverserFileName: "",
			recursionDepth:    0,
			exclusionPattern:  nil,
			traverseSymlinks:  false,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\subdir2",
					},
				},
				"subdir": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir\\subdir2": {},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "symlink1",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "file2.txt",
						mode: 0666,
						size: len("file2 content"),
					},
				},
				{
					name: "symlink2",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "subdir2",
						mode: 0777 | os.ModeDir,
						size: 0,
					},
				},
			},
		},

		// - Symlinks, with follow-flag, no recursion (same outcome)
		{
			name:              "Symlinks, with follow-flag, no recursion",
			traverserFileName: "",
			recursionDepth:    0,
			exclusionPattern:  nil,
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\subdir2",
					},
				},
				"subdir": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir\\subdir2": {
					{
						name:    "file3.txt",
						mode:    0666,
						content: "file3 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "symlink1",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "file2.txt",
						mode: 0666,
						size: len("file2 content"),
					},
				},
				{
					name: "symlink2",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "subdir2",
						mode: 0777 | os.ModeDir,
						size: 0,
					},
				},
			},
		},

		// - Symlinks, with follow-flag, with recursion
		{
			name:              "Symlinks, with follow-flag, with recursion",
			traverserFileName: "",
			recursionDepth:    1,
			exclusionPattern:  nil,
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\subdir2",
					},
				},
				"subdir": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir\\subdir2": {},
				"subdir\\subdir2\\subdir3": {
					{
						name:    "file3.txt",
						mode:    0666,
						content: "file3 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
				{name: "subdir"},
				{name: "subdir\\subdir2", visitDueToSymlink: true}, // due to symlink2 pointing to it
				// no subdir3 since its past recursion limit
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "symlink1",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "file2.txt",
						mode: 0666,
						size: len("file2 content"),
					},
				},
				{
					name: "symlink2",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "subdir2",
						mode: 0777 | os.ModeDir,
						size: 0,
					},
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "subdir3",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
		},

		// Infinite loop due to symlink-to-dir pointing back to original root directory
		// Note: case ensures we don't "infinitely" recurse due to symlink loops
		{
			name:	"Infinite loop due to symlink-to-dir",
			traverserFileName: "",
			recursionDepth:    10,
			exclusionPattern:  nil,
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name: "file1.txt",
						mode: 0444,
						content: "file1 content",
					},
				},
				"subdir1": { 
					{
						name: "file2.txt",
						mode: 0666,
						content: "file2 content",
					},
				},
				"subdir1\\subdir2": {
					{
						name: "symlink-to-subdir1",
						mode: 0666 | os.ModeSymlink,
						content: "subdir1", // link back to root parent directory
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
				{name: "subdir1"},
				{name: "subdir1\\subdir2"},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "file2.txt",
					mode: 0666,
					size: len("file2 content"),
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "symlink-to-subdir1",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "subdir1",
						mode: 0777 | os.ModeDir,
						size: 0,
					},
				},
			},
		},

		// - Exclusion pattern: all (.*)
		{
			name:              "Exclusion pattern: all",
			traverserFileName: "",
			recursionDepth:    1,
			exclusionPattern:  regexp.MustCompile(".*"),
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir\\subdir2",
					},
				},
				"subdir": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir\\subdir2": {},
				"subdir\\subdir2\\subdir3": {
					{
						name:    "file3.txt",
						mode:    0666,
						content: "file3 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{},
		},

		// - Exclusion pattern: dirs only (.*subdir[1-9]$)
		{
			name:              "Exclusion pattern: dirs only",
			traverserFileName: "",
			recursionDepth:    3,
			exclusionPattern:  regexp.MustCompile(".*subdir[1-9]$"),
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir1\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir1\\subdir2",
					},
				},
				"subdir1": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir1\\subdir2": {},
				"subdir1\\subdir2\\subdir3": {
					{
						name:    "file3.txt",
						mode:    0666,
						content: "file3 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "file1.txt",
					mode: 0444,
					size: len("file1 content"),
				},
				{
					name: "symlink1",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "file2.txt",
						mode: 0666,
						size: len("file2 content"),
					},
				},
				// symlink2 is excluded by the exclusion pattern (the target is checked against the pattern as well).
			},
		},

		// - Exclusion pattern: files only
		{
			name:              "Exclusion pattern: files only",
			traverserFileName: "",
			recursionDepth:    3,
			exclusionPattern:  regexp.MustCompile(`.*file[1-9]\.txt$`),
			traverseSymlinks:  true,
			dirStructure: map[string][]TestFileInfo{
				"": {
					{
						name:    "file1.txt",
						mode:    0444,
						content: "file1 content",
					},
					{
						name:    "symlink1",
						mode:    0666 | os.ModeSymlink,
						content: "subdir1\\file2.txt",
					},
					{
						name:    "symlink2",
						mode:    0666 | os.ModeSymlink,
						content: "subdir1\\subdir2",
					},
				},
				"subdir1": {
					{
						name:    "file2.txt",
						mode:    0666,
						content: "file2 content",
					},
				},
				"subdir1\\subdir2": {},
				"subdir1\\subdir2\\subdir3": {
					{
						name:    "file3.txt",
						mode:    0666,
						content: "file3 content",
					},
				},
			},
			expectedRootsVisited: []ExpectedRootInfo{
				{name: ""},
				{name: "subdir1"},
				{name: "subdir1\\subdir2"},
				{name: "subdir1\\subdir2\\subdir3"},
			},
			expectedElementsVisitedNoSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "subdir3",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
			expectedElementsVisitedWithSymlinkChecks: &[]ExpectedTestFileInfo{
				{
					name: "subdir1",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				// symlink1 is excluded by the exclusion pattern (the target is checked against the pattern as well).
				{
					name: "symlink2",
					mode: 0666 | os.ModeSymlink,
					size: 0,
					target: &ExpectedTestFileInfo{
						name: "subdir2",
						mode: 0777 | os.ModeDir,
						size: 0,
					},
				},
				{
					name: "subdir2",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
				{
					name: "subdir3",
					mode: 0777 | os.ModeDir,
					size: 0,
				},
			},
		},

	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Cleanup(func() { cleanupTestFSLayout(&tempDir) })

			err, skipSymlinkChecks := buildTestFSLayout(t, &tempDir, &test.dirStructure)
			if err != nil {
				t.Fatalf("Failed to build test FS layout: %v", err)
			}

			rootsVisited := []string{}
			visitAbsRoot := func(root string) {
				rootsVisited = append(rootsVisited, root)
			}
			elementsVisited := []TFFileInfo{}
			visitElement := func(element *TFFileInfo) {
				elementsVisited = append(elementsVisited, *element)
			}

			traverserRoot := filepath.Join(tempDir, test.traverserFileName)
			NewPathTraverser(
				traverserRoot,
				test.recursionDepth,
				test.exclusionPattern,
				test.traverseSymlinks,
				visitAbsRoot,
				visitElement).Traverse()

			filteredExpectedRoots := filterExpectedRoots(test.expectedRootsVisited, skipSymlinkChecks)
			if len(rootsVisited) != len(filteredExpectedRoots) {
				t.Fatalf("Expected %d roots visited, got %d", len(filteredExpectedRoots), len(rootsVisited))
			}
			// Assumption: The order of roots visited matches the order in expectedRootsVisited
			for i, root := range rootsVisited {
				expectedAbs := filepath.Join(tempDir, filteredExpectedRoots[i].name)
				if root != expectedAbs {
					t.Fatalf("Expected root visited %s, got %s", expectedAbs, root)
				}
			}

			expectedElements := test.expectedElementsVisitedWithSymlinkChecks
			if skipSymlinkChecks {
				expectedElements = test.expectedElementsVisitedNoSymlinkChecks
			}
			if len(elementsVisited) != len(*expectedElements) {
				t.Fatalf("Expected %d elements visited, got %d", len(*expectedElements), len(elementsVisited))
			}
			// Assumption: The order of elements visited matches the order in expectedElementsVisited
			for i, element := range elementsVisited {
				expected := (*expectedElements)[i]
				if element.resolvedSymlinkInfo != nil {
					// We're processing a symlink, validate the symlink info itself first
					if expected.target == nil {
						t.Fatalf("Expected symlink target info is nil")
					}
					if element.fileInfo.Name() != expected.name {
						t.Fatalf("Expected symlink name %s, got %s", expected.name, element.fileInfo.Name())
					}
					if element.fileInfo.Mode() != expected.mode {
						t.Fatalf("Expected symlink mode %v, got %v", expected.mode, element.fileInfo.Mode())
					}
					if element.fileInfo.Size() != int64(expected.size) {
						t.Fatalf("Expected symlink size %d, got %d", expected.size, element.fileInfo.Size())
					}

					// Next, validate the target file info of the symlink
					if expected.target == nil {
						t.Fatalf("Expected symlink target info is nil")
					}
					if element.resolvedSymlinkInfo.fileInfo.Name() != expected.target.name {
						t.Fatalf("Expected symlink target name %s, got %s", expected.target.name, element.resolvedSymlinkInfo.fileInfo.Name())
					}
					if element.resolvedSymlinkInfo.fileInfo.Mode() != expected.target.mode {
						t.Fatalf("Expected symlink target file mode %v, got %v", expected.target.mode, element.resolvedSymlinkInfo.fileInfo.Mode())
					}
					if element.resolvedSymlinkInfo.fileInfo.Size() != int64(expected.target.size) {
						t.Fatalf("Expected symlink target size %d, got %d", expected.target.size, element.resolvedSymlinkInfo.fileInfo.Size())
					}
				} else {
					if element.fileInfo.Name() != expected.name {
						t.Fatalf("Expected element name %s, got %s", expected.name, element.fileInfo.Name())
					}
					if element.fileInfo.Mode() != expected.mode {
						t.Fatalf("Expected element file mode %v, got %v", expected.mode, element.fileInfo.Mode())
					}
					if element.fileInfo.Size() != int64(expected.size) {
						t.Fatalf("Expected element size %d, got %d", expected.size, element.fileInfo.Size())
					}
				}
			}
		})
	}
}
