package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/tferguson7337/learning-golang/tfutils"
)

func printRoot(path string) {
	fmt.Printf("\n%v\n", path)
	fmt.Println("----------------------------------------------------------------")
}

func hashElement(info *tfutils.TFFileInfo) {
	if info.FileInfo.IsDir() {
		// Skip hashing directories
		return
	}
	if info.FileInfo.Mode()&os.ModeSymlink != 0 {
		// Skip hashing symlinks
		return
	}

	file, err := os.OpenFile(*info.AbsPath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	readBuffer := make([]byte, 4096)
	hasher := sha256.New()

	// Read the file in chunks, feed to hasher
	for {
		n, err := file.Read(readBuffer)
		if n > 0 {
			hasher.Write(readBuffer[:n])
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("%x  %v\n", hasher.Sum(nil), info.FileInfo.Name())
}

func main() {
	recursionDepth := uint(0)
	var excludeRegex *regexp.Regexp
	traverseSymlinks := false
	flag.UintVar(&recursionDepth, "depth", 0, "Recursion depth for directory traversal. 0 means no recursion. Default is 0")
	flag.BoolVar(&traverseSymlinks, "symlinks", false, "Whether to traverse directory symlinks. Default is false")
	flag.Func("exclude", "Regex pattern for excluding files or directories. Default is none", func(s string) error {
		var err error
		excludeRegex, err = regexp.Compile(s)
		if err != nil {
			return err
		}
		return nil
	})

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("No arguments provided - please provide one or more paths")
		flag.Usage()
		return
	}

	for _, arg := range flag.Args() {
		absPath, err := filepath.Abs(arg)
		if err != nil {
			fmt.Printf("Failed to get absolute path for argument - err[%v] arg[%v]\n", err, arg)
			absPath = arg
		}

		tfutils.NewPathTraverser(
			absPath,
			recursionDepth,
			excludeRegex,
			traverseSymlinks,
			printRoot,
			hashElement).Traverse()
	}
}
