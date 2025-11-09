package cmd

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamil-koziol/restree/internal/envutil"
	"github.com/kamil-koziol/restree/pkg/restree"
	restree_client "github.com/kamil-koziol/restree/pkg/restree/client"
)

type RunCmdFlags struct {
	Output              io.WriteCloser
	Directory           string
	ExpandBodyVariables bool
	InsecureSkipVerify  bool
	Verbose             bool
}

func Run(base []string, args []string) int {
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <filename>\n", strings.Join(base, " "))
		fmt.Fprintf(os.Stderr, "\nPositional arguments:\n")
		fmt.Fprintf(os.Stderr, "  filename\tPath to the .http file\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		runCmd.PrintDefaults()
	}

	flags := RunCmdFlags{
		Output: os.Stdout,
	}
	defer flags.Output.Close() // nolint

	runCmd.Func("o", "Output file", func(s string) (err error) {
		flags.Output, err = ResolveOutput(s)
		return err
	})

	runCmd.StringVar(&flags.Directory, "D", "", "Specify the starting directory")
	runCmd.BoolVar(&flags.ExpandBodyVariables, "expand-body-variables", false, "Expand body variables")
	runCmd.BoolVar(&flags.InsecureSkipVerify, "k", false, "Allow insecure server connections")
	runCmd.BoolVar(&flags.Verbose, "v", false, "Increase the verbosity")

	if err := runCmd.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse args")
		return 1
	}

	if runCmd.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing required <file> argument.")
		flag.Usage()
		return 1
	}

	filePath := runCmd.Arg(0)

	dir := ""
	if flags.Directory == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not get current working directory: %s\n", err)
			return 1
		}
	} else {
		dir = flags.Directory
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with file abs path: %s\n", err)
		return 1
	}

	filePath, err = filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with file abs path: %s\n", err)
		return 1
	}

	httpFile, err := restree.RecursiveReadFS(os.DirFS(dir), dir, filePath, envutil.All(), restree.RecursiveReadOpts{
		ExpandBodyVariables: flags.ExpandBodyVariables,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client := restree_client.New(&http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: flags.InsecureSkipVerify,
		},
	})

	var bodyReader io.Reader
	if httpFile.Body != "" {
		bodyReader = strings.NewReader(httpFile.Body)
	}
	req, err := http.NewRequest(httpFile.Method, httpFile.URL, bodyReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create request: %s", err)
		return 1
	}
	for header, value := range httpFile.Headers {
		req.Header.Add(header, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error occured during request: %s", err)
		return 1
	}

	_, _ = fmt.Fprintf(os.Stderr, "%s %s %s\n", resp.Status, resp.Request.Method, resp.Request.URL.String())

	if flags.Verbose {
		for k, v := range resp.Header {
			_, _ = fmt.Fprintf(os.Stderr, "%s: %s\n", k, strings.Join(v, ","))
		}
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read response body: %s", err)
		return 1
	}

	_, _ = fmt.Fprintln(flags.Output, string(b))

	return 0
}
