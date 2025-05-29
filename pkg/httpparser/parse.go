package httpparser

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"net/url"
	"strings"
)

type HTTPRequest struct {
	Method  string
	URL     *url.URL
	Headers map[string]string
	Body    string
}

// ParsePartial partially parses .http file (can omit request line)
// .http file structure
//
// <HTTP_METHOD> <URL> <- This is optional
// <Header-Name>: <Header-Value>
// <Header-Name>: <Header-Value>
// ...
//
// <optional body in JSON, plain text, or form format>
func ParsePartial(body io.Reader) (*HTTPRequest, error) {
	scanner := bufio.NewScanner(body)

	req := &HTTPRequest{
		Headers: make(map[string]string),
	}

	state := "start"
	bodyLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()
	retry:
		switch state {
		case "start":
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.Fields(line)
			isRequestLine := len(parts) == 2 && isHTTPMethod(parts[0])
			if isRequestLine {
				req.Method = parts[0]
				u, err := url.Parse(parts[1])
				if err != nil {
					return nil, fmt.Errorf("invalid url %s in %s", parts[1], line)
				}
				req.URL = u
				state = "headers"
			} else {
				// Not a request line, reprocess as header
				state = "headers"
				goto retry
			}
		case "headers":
			if strings.TrimSpace(line) == "" {
				if len(req.Headers) != 0 {
					state = "body"
				}
				continue
			}

			colonIdx := strings.Index(line, ":")
			if colonIdx == -1 {
				return nil, fmt.Errorf("invalid line: %q", line)
			}

			key := strings.TrimSpace(line[:colonIdx])
			value := strings.TrimSpace(line[colonIdx+1:])
			req.Headers[key] = value
		case "body":
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	req.Body = strings.Join(bodyLines, "\n")

	return req, nil
}

func isHTTPMethod(m string) bool {
	switch strings.ToUpper(m) {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD":
		return true
	default:
		return false
	}
}

// Merge merges two http requests and returns new one
func Merge(dst HTTPRequest, src HTTPRequest) HTTPRequest {
	req := HTTPRequest{
		Method:  dst.Method,
		URL:     nil,
		Headers: maps.Clone(dst.Headers),
		Body:    dst.Body,
	}
	if dst.URL != nil {
		req.URL, _ = url.Parse(dst.URL.String())
	}

	if src.Body != "" {
		req.Body = src.Body
	}

	if src.Method != "" {
		req.Method = src.Method
	}

	if src.URL != nil {
		req.URL, _ = url.Parse(src.URL.String())
	}

	maps.Copy(req.Headers, src.Headers)

	return req
}
