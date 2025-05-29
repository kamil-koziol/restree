package http2curl

import (
	"fmt"
	"os/exec"

	"github.com/kamil-koziol/restree/pkg/httpparser"
)

func ToCURL(req httpparser.HTTPRequest) (string, error) {
	args := []string{}

	args = append(args, "-X", req.Method)
	for k, v := range req.Headers {
		args = append(args, "-H", fmt.Sprintf("'%s: %s'", k, v))
	}
	if req.Body != "" {
		args = append(args, "-d", fmt.Sprintf("'%s'", req.Body))
	}
	args = append(args, req.URL.String())

	curl := exec.Command("curl", args...)
	return curl.String(), nil
}
