package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/tferguson7337/learning-golang/tfutils"
)


func printHeader(path string) {
	fmt.Printf("\n%v\n", path)
	fmt.Println("-------------------------------------------")
}

func printFileInfo(fi *tfutils.TFFileInfo) {
	fmt.Printf("%v\n", *fi)
}

func main() {
	log.Default().SetOutput(os.Stderr)

	// Setup flags
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
			log.Printf("Failed to get absolute path for argument - err[%v] arg[%v]\n", err, arg)
			absPath = arg
		}

		tfutils.NewPathTraverser(
			absPath,
			recursionDepth,
			excludeRegex,
			traverseSymlinks,
			printHeader,
			printFileInfo).Traverse()
	}
}
