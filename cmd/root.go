package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamil-koziol/restree/internal/restree"
	"github.com/kamil-koziol/restree/pkg/http2curl"
	flag "github.com/spf13/pflag"
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <filename>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nPositional arguments:\n")
		fmt.Fprintf(os.Stderr, "  filename\tPath to the .http file\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.CommandLine.SortFlags = false

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing required <file> argument.")
		flag.Usage()
		return 1
	}

	filePath := flag.Arg(0)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not get current working directory: %s\n", err)
		return 1
	}

	filePath, err = filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with file abs path: %s\n", err)
		return 1
	}

	httpFile, err := restree.RecursiveRead(cwd, filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	curl, err := http2curl.ToCURL(*httpFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert .http to curl: %s", err)
		return 1
	}

	fmt.Printf("%s", curl)

	return 0
}
