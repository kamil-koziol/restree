package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kamil-koziol/restree/internal/restree"
	"github.com/kamil-koziol/restree/pkg/http2curl"
	flag "github.com/spf13/pflag"
)

type Flags struct {
	Output string
}

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

	flags := Flags{}
	flag.StringVarP(&flags.Output, "output", "o", "-", "Output file, use '-' for stdout")

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

	// Determine the output outWriter
	var outWriter io.Writer
	if flags.Output == "-" {
		outWriter = os.Stdout
	} else {
		file, err := os.Create(flags.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file: %v\n", err)
			return 1
		}
		defer file.Close()
		outWriter = file
	}

	fmt.Fprintln(outWriter, curl)

	return 0
}
