package cmd

import (
	"io"
	"os"
)

func ResolveOutput(val string) (io.WriteCloser, error) {
	if val == "-" {
		return os.Stdout, nil
	} else {
		return os.Create(val)
	}
}
