package restree

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamil-koziol/restree/internal/envutil"
	"github.com/kamil-koziol/restree/pkg/httpparser"
)

const TemplateFileName = "template.http"

func LoadTemplateFromFile(path string) (*httpparser.HTTPRequest, error) {
	template, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s: %s\n", path, err)
	}
	return LoadTemplate(bytes.NewBuffer(template))
}

func LoadTemplate(data io.Reader) (*httpparser.HTTPRequest, error) {
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
	parsed, err := httpparser.ParsePartial(bytes.NewBufferString(content))
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

	builder := &httpparser.HTTPRequest{
		Headers: make(map[string]string),
	}

	currentPath := from
	for _, dir := range dirs {
		currentPath = filepath.Join(currentPath, dir)

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read dir: %s\n", currentPath)
		}

		for _, entry := range entries {
			name := entry.Name()
			if name == TemplateFileName {
				templatePath := filepath.Join(currentPath, name)
				parsed, err := LoadTemplateFromFile(templatePath)
				if err != nil {
					return nil, fmt.Errorf("Failed to load template %s: %s\n", templatePath, err)
				}

				merged := httpparser.Merge(*builder, *parsed)
				builder = &merged
			}
		}
	}

	httpFile, err := LoadTemplateFromFile(to)
	if err != nil {
		return nil, fmt.Errorf("Failed to load file %s: %s\n", httpFile, err)
	}

	final := httpparser.Merge(*builder, *httpFile)
	return &final, nil
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
