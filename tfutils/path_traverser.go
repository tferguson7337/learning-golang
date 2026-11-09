package tfutils

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type TFResolvedSymlinkInfo struct {
	fileInfo fs.FileInfo
	absPath  string
}

type TFFileInfo struct {
	fileInfo            fs.FileInfo
	resolvedSymlinkInfo *TFResolvedSymlinkInfo
}

func NewTFFileInfo(fi fs.FileInfo, absPath *string, excludePattern *regexp.Regexp) *TFFileInfo {

	if excludePattern != nil && excludePattern.MatchString(*absPath) {
		// Skip directory elements that match the exclude regex pattern
		return nil
	}

	tffi := TFFileInfo{fileInfo: fi, resolvedSymlinkInfo: nil}
	if fi.Mode()&os.ModeSymlink != 0 {
		symlinkAbsPath, err := filepath.EvalSymlinks(*absPath)
		if err != nil {
			log.Printf("Failed to evaluate symlink for directory element - %v %v\n", err, *absPath)
			return nil
		}

		if excludePattern != nil && excludePattern.MatchString(symlinkAbsPath) {
			// Skip resolved symlink paths that match the exclude regex pattern
			return nil
		}

		resolvedSymlinkFileInfo, err := os.Stat(symlinkAbsPath)
		if err != nil {
			log.Printf("Failed to get info for evaluated symlink path - %v %v\n", err, *absPath)
			return nil
		}

		tffi.resolvedSymlinkInfo = &TFResolvedSymlinkInfo{
			fileInfo: resolvedSymlinkFileInfo,
			absPath:  symlinkAbsPath}
	}
	return &tffi
}

func (tffi TFFileInfo) String() string {
	var sizeString string
	if tffi.resolvedSymlinkInfo == nil {
		// Non-symlink: [drwx mask] [size] [mod date+time] [name]
		if !tffi.fileInfo.IsDir() {
			sizeString = NewScaledByteInfo(uint64(tffi.fileInfo.Size()), Byte).String()
		}
		return fmt.Sprintf(
			"%10v  %10v  %v  %v",
			tffi.fileInfo.Mode(),
			sizeString,
			tffi.fileInfo.ModTime().Format(time.DateTime),
			tffi.fileInfo.Name())
	}

	if !tffi.resolvedSymlinkInfo.fileInfo.IsDir() {
		sizeString = NewScaledByteInfo(uint64(tffi.resolvedSymlinkInfo.fileInfo.Size()), Byte).String()
	}

	// Symlink: [original drwx mask] [target size] [target mod date+time] [link name] [[target name]]
	return fmt.Sprintf(
		"%10v  %10v  %v  %v  [%v]",
		tffi.fileInfo.Mode(),
		sizeString,
		tffi.resolvedSymlinkInfo.fileInfo.ModTime().Format(time.DateTime),
		tffi.fileInfo.Name(),
		tffi.resolvedSymlinkInfo.absPath)
}

type VisitAbsRootFunc func(path string)
type VisitDirElementFunc func(info *TFFileInfo)

type PathTraverser struct {
	AbsRootPath      string              // Absolute path to the root directory for traversal
	RecursionDepth   uint                // Maximum depth to recurse into subdirectories, 0 means no recursion
	ExcludePattern   *regexp.Regexp      // nil if no exclude pattern is specified
	TraverseSymlinks bool                // Whether to traverse symlinks to directories
	VisitAbsRoot     VisitAbsRootFunc    // Invoked for the root directory before traversal begins
	VisitDirElement  VisitDirElementFunc // Invoked for each file or directory encountered during traversal
}

func NewPathTraverser(
	absRootPath string,
	recursionDepth uint,
	excludePattern *regexp.Regexp,
	traverseSymlinks bool,
	visitAbsRoot VisitAbsRootFunc,
	visitDirElement VisitDirElementFunc) *PathTraverser {

	return &PathTraverser{
		AbsRootPath:      absRootPath,
		RecursionDepth:   recursionDepth,
		ExcludePattern:   excludePattern,
		TraverseSymlinks: traverseSymlinks,
		VisitAbsRoot:     visitAbsRoot,
		VisitDirElement:  visitDirElement}
}

func (pt *PathTraverser) traverseImpl(visitedDirSet *map[string]struct{}) {
	absPathFileInfo, err := os.Lstat(pt.AbsRootPath)
	if err != nil {
		log.Printf("Failed to stat root path: %v\n", err)
		return
	}

	// Determine if this is a single-file traversal (i.e., the root path is a file, not a directory)
	// Need to also handle symlink-to-file case.
	tffi := NewTFFileInfo(absPathFileInfo, &pt.AbsRootPath, pt.ExcludePattern)
	if tffi == nil {
		// Error or excluded by pattern - skip further processing for this root path
		return
	}

	// We were able to get file info, so appears valid enough - mark root as visited.
	if pt.VisitAbsRoot != nil {
		pt.VisitAbsRoot(pt.AbsRootPath)
	}

	rootIsDir := tffi.fileInfo.IsDir()
	rootIsTargetDir := tffi.resolvedSymlinkInfo != nil && tffi.resolvedSymlinkInfo.fileInfo.IsDir()
	if !rootIsDir && (!rootIsTargetDir || !pt.TraverseSymlinks) {
		// Single-file traversal - process now and bail out
		if pt.VisitDirElement != nil {
			pt.VisitDirElement(tffi)
		}
		return
	}

	var entries []string
	absRootPath := &pt.AbsRootPath
	if rootIsDir {
		absRootFile, err := os.Open(*absRootPath)
		if err != nil {
			log.Printf("Failed to open directory: %v\n", err)
			return
		}

		entries, err = absRootFile.Readdirnames(-1)
		absRootFile.Close()
		if err != nil {
			log.Printf("Failed to read dir names: %v\n", err)
			return
		}
	} else {
		// The provided root path is a symlink-to-dir
		// We'll treat this as a bit of a hybrid case:
		// - We'll start treating the directory containing the symlink as the new root
		// - We'll add the symlink itself as the only entry to process within that directory
		// - We'll then recurse into this directory as usual, if TraverseSymlinks and recursion depth allows
		absRootPathStr, entry := filepath.Split(pt.AbsRootPath)
		absRootPath = &absRootPathStr
		entries = append(entries, entry)
	}

	subDirs := []string{}
	for _, entry := range entries {
		entryAbsPath := filepath.Join(*absRootPath, entry)

		entryFileInfo, err := os.Lstat(entryAbsPath)
		if err != nil {
			log.Printf("Failed to get info for directory element - %v %v\n", err, entryAbsPath)
			continue
		}

		tffi := NewTFFileInfo(entryFileInfo, &entryAbsPath, pt.ExcludePattern)
		if tffi == nil {
			// Error or excluded by pattern - skip further processing for this entry
			continue
		}

		if pt.VisitDirElement != nil {
			pt.VisitDirElement(tffi)
		}

		if !entryFileInfo.IsDir() &&
			!(tffi.resolvedSymlinkInfo != nil && tffi.resolvedSymlinkInfo.fileInfo.IsDir()) {
			// We're not dealing with a directory - skip further processing for this element
			continue
		}

		if pt.RecursionDepth == 0 {
			// If recursion depth is 0, don't recurse into subdirectories
			continue
		}

		// Add the directory (or resolved symlink) to the visited set to avoid revisiting it
		// Also add the directory to the subDirs list for further recursion
		if tffi.resolvedSymlinkInfo != nil {
			if !pt.TraverseSymlinks {
				// We're not traversing symlinks, so skip this entry
				continue
			}
			if _, visited := (*visitedDirSet)[tffi.resolvedSymlinkInfo.absPath]; visited {
				continue
			}
			(*visitedDirSet)[tffi.resolvedSymlinkInfo.absPath] = struct{}{}
			subDirs = append(subDirs, tffi.resolvedSymlinkInfo.absPath)
		} else {
			if _, visited := (*visitedDirSet)[entryAbsPath]; visited {
				continue
			}
			(*visitedDirSet)[entryAbsPath] = struct{}{}
			subDirs = append(subDirs, entryAbsPath)
		}

		// Track additional subdirectories we need to recurse into
	}

	if pt.RecursionDepth > 0 {
		for _, subDir := range subDirs {
			NewPathTraverser(
				subDir,
				pt.RecursionDepth-1,
				pt.ExcludePattern,
				pt.TraverseSymlinks,
				pt.VisitAbsRoot,
				pt.VisitDirElement).traverseImpl(visitedDirSet)
		}
	}
}

func (pt *PathTraverser) Traverse() {
	visitedDirSet := make(map[string]struct{})
	visitedDirSet[pt.AbsRootPath] = struct{}{}
	pt.traverseImpl(&visitedDirSet)
}
