package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamil-koziol/restree/pkg/restree"
)

type InitCmdFlags struct {
	Directory string
}

func Init(base []string, args []string) int {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", strings.Join(base, " "))
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		initCmd.PrintDefaults()
	}

	flags := InitCmdFlags{}
	initCmd.StringVar(&flags.Directory, "D", "", "Specify the init directory")

	if err := initCmd.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse args")
		return 1
	}

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

	// Simple scaffold

	// .env
	envFile := `host=http://localhost`

	envFilePath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFilePath, []byte(envFile), 0o660); err != nil {
		fmt.Fprintf(os.Stderr, "Error unable to create %s: %s\n", envFilePath, err)
		return 1
	}

	// headers
	headers := `Accept: */*
Cache-Control: no-cache
Connection: keep-alive
# Content-Type: application/json
# Authorization: Bearer {{token}}`

	headersFilePath := filepath.Join(dir, restree.HeadersFileName)
	if err := os.WriteFile(headersFilePath, []byte(headers), 0o660); err != nil {
		fmt.Fprintf(os.Stderr, "Error unable to create %s: %s\n", headersFilePath, err)
		return 1
	}

	// before script
	beforeScript := `#!/bin/sh

cat .env`

	beforeScriptFilePath := filepath.Join(dir, restree.BeforeScriptFileName)
	if err := os.WriteFile(beforeScriptFilePath, []byte(beforeScript), 0o770); err != nil {
		fmt.Fprintf(os.Stderr, "Error unable to create %s: %s\n", beforeScriptFilePath, err)
		return 1
	}

	// simple http file
	httpFile := `POST {{host}}/hello

{
	"message": "Hello world!"
}`

	httpFileFilePath := filepath.Join(dir, "hello.http")
	if err := os.WriteFile(httpFileFilePath, []byte(httpFile), 0o770); err != nil {
		fmt.Fprintf(os.Stderr, "Error unable to create %s: %s\n", httpFileFilePath, err)
		return 1
	}

	return 0
}
