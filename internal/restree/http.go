package restree

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamil-koziol/restree/internal/envutil"
	"github.com/kamil-koziol/restree/pkg/httpparser"
)

const (
	HeadersFileName      = "_headers.http"
	BeforeScriptFileName = "_before.sh"
)

// ExpandHTTPRequest expands HTTP request with provided variables
func ExpandHTTPRequest(req *httpparser.HTTPRequest, vars map[string]string, expandBodyVariables bool) (*httpparser.HTTPRequest, error) {
	result := &httpparser.HTTPRequest{
		Method:  req.Method,
		Headers: httpparser.HTTPHeaders{},
	}

	// Expand URL
	u, err := expandVariables(req.URL, vars)
	if err != nil {
		return nil, fmt.Errorf("unable to expand url: %w", err)
	}
	result.URL = u

	// Expand headers
	for h, v := range req.Headers {
		eh, err := expandVariables(v, vars)
		if err != nil {
			return nil, fmt.Errorf("unable to expand header: %s: %s: %w", h, v, err)
		}
		result.Headers[h] = eh
	}

	// Expand body
	if expandBodyVariables {
		result.Body, err = expandVariables(req.Body, vars)
		if err != nil {
			return nil, fmt.Errorf("unable to expand body: %w", err)
		}
	} else {
		result.Body = req.Body
	}

	return result, nil
}

// ReadHTTPRequest reads [httpparser.HTTPRequest] from data and expands it with provided variables
func ReadHTTPRequest(data io.Reader, vars map[string]string, expandBodyVariables bool) (*httpparser.HTTPRequest, error) {
	httpRequest, err := httpparser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %s", err)
	}

	expandedHTTPRequest, err := ExpandHTTPRequest(httpRequest, vars, expandBodyVariables)
	if err != nil {
		return nil, fmt.Errorf("unable to expand http request: %w", err)
	}

	if !expandBodyVariables {
		expandedHTTPRequest.Body = httpRequest.Body
	}

	return expandedHTTPRequest, nil
}

func LoadHTTPHeaders(path string) (httpparser.HTTPHeaders, error) {
	template, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read headers %s: %s", path, err)
	}
	return HandleHTTPHeaders(bytes.NewBuffer(template))
}

func HandleHTTPHeaders(data io.Reader) (httpparser.HTTPHeaders, error) {
	b, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("unable to read from data: %s", err)
	}

	content, err := expandVariables(string(b), envutil.All())
	if err != nil {
		return nil, fmt.Errorf("failed to fill template: %s", err)
	}

	parsed, err := httpparser.ParseHeadersFile(bytes.NewBufferString(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %s", err)
	}

	return parsed, nil
}

func runScript(scriptPath string) (string, string, error) {
	cmd := exec.Command("/bin/sh", scriptPath)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("error happened during script execution: %s", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), err
}

// parseScriptEnvOutput parses the output as env
func parseScriptEnvOutput(out string) (map[string]string, error) {
	envMap := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewBufferString(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				envMap[key] = value
			}
		}
	}

	return envMap, nil
}

func processDirectory(currentPath string) (httpparser.HTTPHeaders, error) {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %s", currentPath)
	}

	headers := httpparser.HTTPHeaders{}

	var headersPath, beforePath string

	// find the files in directory
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		entryPath := filepath.Join(currentPath, name)

		switch entry.Name() {
		case HeadersFileName:
			headersPath = entryPath
		case BeforeScriptFileName:
			beforePath = entryPath
		}

	}

	// run the before script first
	// TODO: pass the current headers to the scripts as a variable
	if beforePath != "" {
		stdout, stderr, err := runScript(beforePath)
		if err != nil {
			return nil, fmt.Errorf("failed to execute before script: %s\n%s", err, stderr)
		}
		exportedEnvs, err := parseScriptEnvOutput(stdout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse envs: %s", err)
		}

		// set the variables
		for key, value := range exportedEnvs {
			if err := os.Setenv(key, value); err != nil {
				return nil, fmt.Errorf("unable to set env variable: %s", err)
			}
		}
	}

	// run parse the headers
	if headersPath != "" {
		headers, err = LoadHTTPHeaders(headersPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %s", headersPath, err)
		}
	}

	return headers, nil
}

type RecursiveReadOpts struct {
	ExpandBodyVariables bool
}

func RecursiveRead(from string, to string, opts RecursiveReadOpts) (*httpparser.HTTPRequest, error) {
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
		directoryHeaders, err := processDirectory(currentPath)
		if err != nil {
			return nil, fmt.Errorf("unable to process dir: %s", err)
		}
		maps.Copy(headers, directoryHeaders)
	}

	f, err := os.Open(to)
	if err != nil {
		return nil, fmt.Errorf("unable to open target file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	httpFile, err := ReadHTTPRequest(f, envutil.All(), opts.ExpandBodyVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to load file %s: %s", to, err)
	}

	maps.Copy(httpFile.Headers, headers)
	return httpFile, nil
}

// expandVariables replaces all occurrences of variables in the `{{var}}` format
//
// Each variable placeholder in the content (e.g., "{{name}}") is replaced with
// the corresponding value from the `variables` map (e.g., variables["name"]).
//
// Example:
//
//	variables := map[string]string{
//	    "test": "world",
//	}
//	content := "hello {{test}}"
//	result, err := expandVariables(content, variables)
//	// result == "hello world"
func expandVariables(content string, variables map[string]string) (string, error) {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)

	var missingVariables []string
	output := re.ReplaceAllStringFunc(content, func(match string) string {
		key := re.FindStringSubmatch(match)[1]
		if val, ok := variables[key]; ok {
			return val
		}

		missingVariables = append(missingVariables, key)
		return match
	})

	if len(missingVariables) != 0 {
		return output, fmt.Errorf("missing variables: %s", missingVariables)
	}

	return output, nil
}
