package httpparser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type HTTPRequest struct {
	Method  string
	URL     string
	Headers HTTPHeaders
	Body    string
}

func (req *HTTPRequest) String() string {
	s := ""

	// Request line
	s += fmt.Sprintf("%s %s\n", req.Method, req.URL)

	if len(req.Headers) == 0 && req.Body == "" {
		return s
	}

	// Headers
	s += "\n"
	for header, value := range req.Headers {
		s += fmt.Sprintf("%s: %s\n", header, value)
	}

	// Body
	if req.Body != "" {
		s += "\n"
		s += req.Body
	}
	return s
}

type HTTPHeaders map[string]string

// Parse parses .http file
// .http file structure
//
// <HTTP_METHOD> <URL>
// <Header-Name>: <Header-Value>
// <Header-Name>: <Header-Value>
// ...
//
// <optional body in JSON, plain text, or form format>
func Parse(body io.Reader) (*HTTPRequest, error) {
	scanner := bufio.NewScanner(body)

	req := &HTTPRequest{
		Headers: make(HTTPHeaders),
	}

	state := "start"
	bodyLines := []string{}
	scannedLines := 0

	for scanner.Scan() {
		line := scanner.Text()
		scannedLines += 1

		switch state {
		case "start":
			// skip first empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.Fields(line)

			if !isHTTPMethod(parts[0]) {
				return nil, fmt.Errorf("invalid HTTP method: %s", req.Method)
			}
			req.Method = parts[0]
			req.URL = parts[1]

			state = "headersStart"
		case "headersStart":
			if strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("has to be empty headers start")
			}

			state = "headers"
		case "headers":
			if strings.TrimSpace(line) == "" {
				state = "body"
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

	if scannedLines == 0 {
		return nil, fmt.Errorf("cannot parse empty")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	req.Body = strings.Join(bodyLines, "\n")

	return req, nil
}

// ParseHeadersFile parses .http file that contains only headers
// .http file structure (headers only)
//
// <Header-Name>: <Header-Value>
// <Header-Name>: <Header-Value>
// ...
func ParseHeadersFile(body io.Reader) (HTTPHeaders, error) {
	scanner := bufio.NewScanner(body)

	headers := make(HTTPHeaders)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			return headers, nil
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			return nil, fmt.Errorf("invalid header: %q", line)
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		headers[key] = value
	}

	return headers, nil
}

func isHTTPMethod(m string) bool {
	switch strings.ToUpper(m) {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD":
		return true
	default:
		return false
	}
}
