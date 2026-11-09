package tfutils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

const testFileContent string = "test file content"

func createTempFile(
	t *testing.T,
	dir string,
	name string, /* relative to dir */
	target *string /* If non-nil, `name` will be a symlink */) (
	*fs.FileInfo, // opened file or symlink
	*string, // full path to new file
	*string, // full path to target file (if symlink)
) {
	newFilePath := filepath.Join(dir, name)
	if target == nil {
		// Non-symlink path
		file, err := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer file.Close()

		_, err = file.Write([]byte(testFileContent))
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		fileInfo, err := os.Stat(newFilePath)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}

		return &fileInfo, &newFilePath, nil
	}

	// Symlink path, requires elevated privileges
	targetFilePath := filepath.Join(dir, *target)
	err := os.Symlink(targetFilePath, newFilePath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, WIN_ERROR_PRIVILEGE_NOT_HELD) {
			// Signal non-fatal failure to caller
			return nil, nil, nil
		}
		t.Fatalf("Failed to create symlink: %v", err)
	}

	fileInfo, err := os.Lstat(newFilePath)
	if err != nil {
		t.Fatalf("Failed to open symlink: %v", err)
	}
	return &fileInfo, &newFilePath, &targetFilePath
}

func TestNewTFFileInfo(t *testing.T) {
	testDir := t.TempDir()
	testDirInfo, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat test dir: %v", err)
	}

	tfTestDirInfo := NewTFFileInfo(testDirInfo, &testDir, nil)
	if tfTestDirInfo == nil {
		t.Fatal("Received nil test dir info")
	}
	if tfTestDirInfo.fileInfo != testDirInfo {
		t.Fatalf(
			"tfTestDirInfo.fileInfo does not match os.FileInfo:\n`%v`\n`%v`",
			tfTestDirInfo.fileInfo,
			testDirInfo)
	}
	if tfTestDirInfo.resolvedSymlinkInfo != nil {
		t.Fatal("tfTestDirInfo contains non-nil symlink info")
	}

	testFileName := "testFile.txt"
	testFileInfo, testFileAbsPath, _ := createTempFile(t, testDir, testFileName, nil)
	if testFileInfo == nil {
		t.Fatal("Received nil test file info")
	}

	tfTestFileInfo := NewTFFileInfo(*testFileInfo, testFileAbsPath, nil)
	if tfTestFileInfo == nil {
		t.Fatal("Failed to get TFFileInfo for test file")
	}
	if !reflect.DeepEqual(tfTestFileInfo.fileInfo, *testFileInfo) {
		t.Fatalf(
			"tfTestFileInfo.fileInfo does not match fs.FileInfo:\n`%v`\n`%v`",
			tfTestFileInfo.fileInfo,
			*testFileInfo)
	}
	if tfTestFileInfo.resolvedSymlinkInfo != nil {
		t.Fatal("tfTestFileInfo contains non-nil symlink info")
	}

	testSymlinkName := "test-symlink"
	testSymlinkInfo, testSymlinkAbsPath, testSymlinkTargetAbsPath := createTempFile(t, testDir, testSymlinkName, &testFileName)
	if testSymlinkInfo == nil {
		t.Log("Symlink not created due to privilege requirements, skipping symlink checks")
		return
	}
	if testSymlinkAbsPath == nil || testSymlinkTargetAbsPath == nil {
		t.Fatal("Received nil symlink abs path or nil symlink target abs path")
	}

	tfTestSymlinkInfo := NewTFFileInfo(*testSymlinkInfo, testSymlinkAbsPath, nil)
	if tfTestSymlinkInfo == nil {
		t.Fatal("Failed to get TFFileInfo for symlink")
	}
	if !reflect.DeepEqual(tfTestSymlinkInfo.fileInfo, *testSymlinkInfo) {
		t.Fatalf(
			"tfTestSymlinkInfo.fileInfo does not match fs.FileInfo:\n`%v`\n`%v`",
			tfTestSymlinkInfo.fileInfo,
			*testSymlinkInfo)
	}
	if tfTestSymlinkInfo.resolvedSymlinkInfo == nil {
		t.Fatal("tfTestSymlinkInfo is missing target info")
	}
	if tfTestSymlinkInfo.resolvedSymlinkInfo.absPath != *testFileAbsPath {
		t.Fatalf(
			"Symlink's target path does not match original:\n`%v`\n`%v`",
			tfTestSymlinkInfo.resolvedSymlinkInfo.absPath,
			*testFileAbsPath)
	}
	if !reflect.DeepEqual(tfTestSymlinkInfo.resolvedSymlinkInfo.fileInfo, *testFileInfo) {
		t.Fatalf(
			"Symlink target's file info does not match original:\n`%v`\n`%v`",
			tfTestSymlinkInfo.resolvedSymlinkInfo.fileInfo,
			*testFileInfo)
	}
}

func TestTFFileInfoString(t *testing.T) {
	testDir := t.TempDir()
	testFileName := "testFile.txt"
	testFileInfo, testFileAbsPath, _ := createTempFile(t, testDir, testFileName, nil)
	if testFileInfo == nil {
		t.Fatal("Received nil test file info")
	}

	tfTestFileInfo := NewTFFileInfo(*testFileInfo, testFileAbsPath, nil)
	if tfTestFileInfo == nil {
		t.Fatal("Failed to get TFFileInfo for test file")
	}

	testFileRegex := regexp.MustCompile(
		// [drwx mask] [size] [mod date+time] [name]
		fmt.Sprintf(
			`%v\s+%v\s+%v\s+%v`,
			tfTestFileInfo.fileInfo.Mode(),
			NewScaledByteInfo(uint64(tfTestFileInfo.fileInfo.Size()), Byte).String(),
			tfTestFileInfo.fileInfo.ModTime().Format(time.DateTime),
			testFileName))
	tfTestFileInfoStr := tfTestFileInfo.String()
	if !testFileRegex.MatchString(tfTestFileInfoStr) {
		t.Fatalf(
			"String output did not match regex:\n'%v'\n'%v'",
			testFileRegex.String(),
			tfTestFileInfoStr)
	}

	// Test symlink to testFile.txt
	testSymlinkName := "test-symlink"
	testSymlinkFileInfo, testSymlinkAbsPath, testTargetAbsPath := createTempFile(t, testDir, testSymlinkName, &testFileName)
	if testSymlinkFileInfo == nil {
		t.Log("Could not create symlink - skipping symlink part of test")
		return
	}
	if testSymlinkAbsPath == nil || testTargetAbsPath == nil {
		t.Fatal("Received nil abs path(s)")
	}
	if *testTargetAbsPath != *testFileAbsPath {
		t.Fatalf(
			"Target abs path mismatch with original file abs path:\n'%v'\n'%v'",
			*testTargetAbsPath,
			*testFileAbsPath)
	}

	tfTestSymlinkFileInfo := NewTFFileInfo(*testSymlinkFileInfo, testSymlinkAbsPath, nil)
	if tfTestSymlinkFileInfo == nil {
		t.Fatal("Received nil TFFileInfo for symlink")
	}
	if tfTestSymlinkFileInfo.resolvedSymlinkInfo == nil {
		t.Fatal("Received nil TFFileInfo.resolvedSymlinkInfo")
	}

	// [original drwx mask] [target size] [target mod date+time] [name] [target abs path]
	testSymlinkRegexStr := fmt.Sprintf(
		`%v\s+%v\s+%v\s+%v\s+\[%v\]`,
		tfTestSymlinkFileInfo.fileInfo.Mode(),
		NewScaledByteInfo(uint64(tfTestSymlinkFileInfo.resolvedSymlinkInfo.fileInfo.Size()), Byte).String(),
		tfTestSymlinkFileInfo.resolvedSymlinkInfo.fileInfo.ModTime().Format(time.DateTime),
		testSymlinkName,
		strings.ReplaceAll(*testTargetAbsPath, `\`, `\\`))
	testSymlinkRegex := regexp.MustCompile(testSymlinkRegexStr)
	tfTestSymlinkFileInfoStr := tfTestSymlinkFileInfo.String()
	if !testSymlinkRegex.MatchString(tfTestSymlinkFileInfoStr) {
		t.Fatalf(
			"String output did not match regex:\n'%v'\n'%v'",
			testSymlinkRegex.String(),
			tfTestSymlinkFileInfoStr)
	}
}
