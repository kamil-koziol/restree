package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"flag"
	"github.com/kamil-koziol/restree/internal/restree"
)

type Flags struct {
	Output string
	Body   string
}

func Run() int {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <filename>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nPositional arguments:\n")
		fmt.Fprintf(os.Stderr, "  filename\tPath to the .http file\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	flags := Flags{}
	flag.StringVar(&flags.Output, "o", "-", "Output file, use '-' for stdout")
	flag.StringVar(&flags.Body, "b", "", "Specify the input for the final .http body. Use a file path to write to a file, or '-' to use stdin")

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

	var bodyReader io.Reader
	if flags.Body == "-" {
		bodyReader = os.Stdin
	} else if flags.Body != "" {
		file, err := os.Open(flags.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open file: %s", err)
			return 1
		}
		defer file.Close()
		bodyReader = file
	}

	if bodyReader != nil {
		// overwrite the body
		b, err := io.ReadAll(bodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read: %s", err)
			return 1
		}
		httpFile.Body = string(b)
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

	fmt.Fprintln(outWriter, httpFile.String())

	return 0
}
