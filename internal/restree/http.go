package restree

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamil-koziol/restree/internal/envutil"
	"github.com/kamil-koziol/restree/pkg/httpparser"
)

const HeadersFileName = "_headers.http"

func LoadHTTPRequest(path string) (*httpparser.HTTPRequest, error) {
	template, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read request %s: %s\n", path, err)
	}
	return HandleHTTPRequest(bytes.NewBuffer(template))
}

func HandleHTTPRequest(data io.Reader) (*httpparser.HTTPRequest, error) {
	// parse it
	template, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("unable to read from data: %s", err)
	}

	// fill the placeholders
	content, err := fillPlaceholders(string(template), envutil.All())
	if err != nil {
		return nil, fmt.Errorf("Failed to fill template: %s\n", err)
	}

	// now we can parse it as .http
	parsed, err := httpparser.Parse(bytes.NewBufferString(content))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %s: \n%s\n", err, content)
	}

	return parsed, nil
}

func LoadHTTPHeaders(path string) (httpparser.HTTPHeaders, error) {
	template, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read headers %s: %s\n", path, err)
	}
	return HandleHTTPHeaders(bytes.NewBuffer(template))
}

func HandleHTTPHeaders(data io.Reader) (httpparser.HTTPHeaders, error) {
	b, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("unable to read from data: %s", err)
	}

	content, err := fillPlaceholders(string(b), envutil.All())
	if err != nil {
		return nil, fmt.Errorf("Failed to fill template: %s\n", err)
	}

	parsed, err := httpparser.ParseHeadersFile(bytes.NewBufferString(content))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %s: \n%s\n", err, content)
	}

	return parsed, nil
}

func RecursiveRead(from string, to string) (*httpparser.HTTPRequest, error) {
	traversalStr, found := strings.CutPrefix(to, from)
	if !found {
		return nil, fmt.Errorf("%s must be under %s", to, from)
	}

	traversal := strings.Split(traversalStr, string(os.PathSeparator))

	dirs := traversal[:len(traversal)-1]

	headers := httpparser.HTTPHeaders{}

	currentPath := from
	for _, dir := range dirs {
		currentPath = filepath.Join(currentPath, dir)

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read dir: %s\n", currentPath)
		}

		for _, entry := range entries {
			name := entry.Name()
			if name == HeadersFileName {
				headersPath := filepath.Join(currentPath, name)
				currentHeaders, err := LoadHTTPHeaders(headersPath)
				if err != nil {
					return nil, fmt.Errorf("Failed to load template %s: %s\n", headersPath, err)
				}

				maps.Copy(headers, currentHeaders)
			}
		}
	}

	httpFile, err := LoadHTTPRequest(to)
	if err != nil {
		return nil, fmt.Errorf("Failed to load file %s: %s\n", to, err)
	}

	maps.Copy(httpFile.Headers, headers)
	return httpFile, nil
}

func fillPlaceholders(content string, values map[string]string) (string, error) {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)

	var missingVariables []string
	output := re.ReplaceAllStringFunc(content, func(match string) string {
		key := re.FindStringSubmatch(match)[1]
		if val, ok := values[key]; ok {
			return val
		}

		missingVariables = append(missingVariables, key)
		return match
	})

	if len(missingVariables) != 0 {
		return output, fmt.Errorf("missing variables: %s\n%s\n", missingVariables, output)
	}

	return output, nil
}
