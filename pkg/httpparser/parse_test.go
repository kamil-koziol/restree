package httpparser

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kamil-koziol/restree/internal/assert"
)

func validMethod() string {
	return "POST"
}

func validURL() string {
	return "https://localhost:8080"
}

func validRequestLine() string {
	return fmt.Sprintf("%s %s\n", validMethod(), validURL())
}

func validHeaders() string {
	return "Content-Type: application/json\nAuthorization: test\n"
}

func validContent() string {
	return "{\"hello\": \"world\"}"
}

func validHTTPFile() string {
	return validRequestLine() + "\n" + validHeaders() + "\n" + validContent()
}

func validHTTPFileContentOnly() string {
	return validRequestLine() + "\n\n" + validContent()
}

func validHTTPFileHeadersOnly() string {
	return validRequestLine() + "\n" + validHeaders()
}

// Test Parse

func TestParseHTTPFile(t *testing.T) {
	b := validHTTPFile()
	req, err := Parse(bytes.NewBufferString(b))
	assert.Eq(t, nil, err)
	assert.Eq(t, req.Method, validMethod())
	assert.Eq(t, validURL(), req.URL)
	assert.Eq(t, 2, len(req.Headers))
	assert.Eq(t, "application/json", req.Headers["Content-Type"])
	assert.Eq(t, "test", req.Headers["Authorization"])
	assert.Eq(t, validContent(), req.Body)
}

func TestParseHTTPFileContentOnly(t *testing.T) {
	b := validHTTPFileContentOnly()
	req, err := Parse(bytes.NewBufferString(b))
	assert.Eq(t, nil, err)
	assert.Eq(t, validMethod(), req.Method)
	assert.Eq(t, validURL(), req.URL)
	assert.Eq(t, 0, len(req.Headers))
	assert.Eq(t, validContent(), req.Body)
}

func TestParseHTTPFileHeadersOnly(t *testing.T) {
	b := validHTTPFileHeadersOnly()
	req, err := Parse(bytes.NewBufferString(b))
	assert.Eq(t, err, nil)
	assert.Eq(t, validMethod(), req.Method)
	assert.Eq(t, validURL(), req.URL)
	assert.Eq(t, 2, len(req.Headers))
	assert.Eq(t, "application/json", req.Headers["Content-Type"])
	assert.Eq(t, "test", req.Headers["Authorization"])
	assert.Eq(t, "", req.Body)
}

func TestParseEmptyFails(t *testing.T) {
	b := ""
	_, err := Parse(bytes.NewBufferString(b))
	assert.Neq(t, nil, err)
}

func TestParseHTTPFileOnlyRequestLine(t *testing.T) {
	b := validRequestLine()
	req, err := Parse(bytes.NewBufferString(b))
	assert.Eq(t, nil, err)
	assert.Eq(t, validMethod(), req.Method)
	assert.Eq(t, validURL(), req.URL)
	assert.Eq(t, 0, len(req.Headers))
	assert.Eq(t, "", req.Body)
}

// Test ParseHeadersFile

func TestParseHeadersFile(t *testing.T) {
	b := validHeaders()
	headers, err := ParseHeadersFile(bytes.NewBufferString(b))
	assert.Eq(t, nil, err)
	assert.Eq(t, 2, len(headers))
	assert.Eq(t, "application/json", headers["Content-Type"])
	assert.Eq(t, "test", headers["Authorization"])
}
