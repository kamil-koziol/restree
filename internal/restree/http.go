package restree

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamil-koziol/restree/pkg/httpparser"
)

const (
	HeadersFileName      = "_headers.http"
	BeforeScriptFileName = "_before.sh"
)

type Variables map[string]string

// ExpandHTTPRequest expands HTTP request with provided variables
func ExpandHTTPRequest(req *httpparser.HTTPRequest, variables Variables, expandBodyVariables bool) (*httpparser.HTTPRequest, error) {
	result := &httpparser.HTTPRequest{
		Method:  req.Method,
		Headers: httpparser.HTTPHeaders{},
	}

	// Expand URL
	u, err := expandVariables(req.URL, variables)
	if err != nil {
		return nil, fmt.Errorf("unable to expand url: %w", err)
	}
	result.URL = u

	// Expand headers
	for h, v := range req.Headers {
		eh, err := expandVariables(v, variables)
		if err != nil {
			return nil, fmt.Errorf("unable to expand header: %s: %s: %w", h, v, err)
		}
		result.Headers[h] = eh
	}

	// Expand body
	if expandBodyVariables {
		result.Body, err = expandVariables(req.Body, variables)
		if err != nil {
			return nil, fmt.Errorf("unable to expand body: %w", err)
		}
	} else {
		result.Body = req.Body
	}

	return result, nil
}

// ReadHTTPRequest reads [httpparser.HTTPRequest] from data and expands it with provided variables
func ReadHTTPRequest(data io.Reader, variables Variables, expandBodyVariables bool) (*httpparser.HTTPRequest, error) {
	httpRequest, err := httpparser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %s", err)
	}

	expandedHTTPRequest, err := ExpandHTTPRequest(httpRequest, variables, expandBodyVariables)
	if err != nil {
		return nil, fmt.Errorf("unable to expand http request: %w", err)
	}

	if !expandBodyVariables {
		expandedHTTPRequest.Body = httpRequest.Body
	}

	return expandedHTTPRequest, nil
}

// ReadHTTPRequest reads [httpparser.HTTPHeaders] from data and expands it with provided variables
func ReadHTTPHeaders(data io.Reader, variables Variables) (httpparser.HTTPHeaders, error) {
	b, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("unable to read from data: %s", err)
	}

	content, err := expandVariables(string(b), variables)
	if err != nil {
		return nil, fmt.Errorf("failed to fill template: %s", err)
	}

	parsed, err := httpparser.ParseHeadersFile(bytes.NewBufferString(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %s", err)
	}

	return parsed, nil
}

func runScriptFromFS(fsys fs.FS, scriptPath string) (string, string, error) {
	file, err := fsys.Open(scriptPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot open script: %w", err)
	}
	defer file.Close() //nolint:errcheck

	scriptData, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("cannot read script: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "script-*.sh")
	if err != nil {
		return "", "", fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck
	defer tmpFile.Close()           //nolint:errcheck

	if _, err := tmpFile.Write(scriptData); err != nil {
		return "", "", fmt.Errorf("cannot write temp script: %w", err)
	}

	if err := os.Chmod(tmpFile.Name(), 0o700); err != nil {
		return "", "", fmt.Errorf("cannot chmod temp script: %w", err)
	}

	cmd := exec.Command("/bin/sh", tmpFile.Name())
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("script execution error: %w", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}

// parseScriptEnvOutput parses the output as env
func parseScriptEnvOutput(out string) (Variables, error) {
	envMap := make(Variables)
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

func processDirectoryFS(fsys fs.FS, currentPath string, variables Variables) (httpparser.HTTPHeaders, error) {
	entries, err := fs.ReadDir(fsys, currentPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read dir %s: %w", currentPath, err)
	}

	var headersFile, beforeScriptFile fs.DirEntry

	// find the files in directory
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		switch entry.Name() {
		case HeadersFileName:
			headersFile = entry
		case BeforeScriptFileName:
			beforeScriptFile = entry
		}
	}

	// run the before script first
	// TODO: pass the current headers to the scripts as a variable
	if beforeScriptFile != nil {
		stdout, stderr, err := runScriptFromFS(fsys, filepath.Join(currentPath, beforeScriptFile.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to execute before script: %s\n%s", err, stderr)
		}
		exportedEnvs, err := parseScriptEnvOutput(stdout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse envs: %s", err)
		}

		// set the variables
		maps.Copy(variables, exportedEnvs)
	}

	// run parse the headers
	headers := httpparser.HTTPHeaders{}
	if headersFile != nil {
		headersPath := filepath.Join(currentPath, headersFile.Name())
		f, err := fsys.Open(headersPath)
		if err != nil {
			return nil, fmt.Errorf("unable to open headers file: %w", err)
		}
		defer f.Close() //nolint:errcheck

		headers, err = ReadHTTPHeaders(f, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %s", headersPath, err)
		}
	}

	return headers, nil
}

type RecursiveReadOpts struct {
	ExpandBodyVariables bool
}

func RecursiveReadFS(fsys fs.FS, from string, to string, variables Variables, opts RecursiveReadOpts) (*httpparser.HTTPRequest, error) {
	traversalStr, found := strings.CutPrefix(to, from)
	if !found {
		return nil, fmt.Errorf("%s must be under %s", to, from)
	}

	traversal := strings.Split(traversalStr, string(os.PathSeparator))

	dirs := traversal[:len(traversal)-1]

	headers := httpparser.HTTPHeaders{}

	currentPath := "."
	for _, dir := range dirs {
		currentPath = filepath.Join(currentPath, dir)
		directoryHeaders, err := processDirectoryFS(fsys, currentPath, variables)
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

	httpFile, err := ReadHTTPRequest(f, variables, opts.ExpandBodyVariables)
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
//	variables := Variables{
//	    "test": "world",
//	}
//	content := "hello {{test}}"
//	result, err := expandVariables(content, variables)
//	// result == "hello world"
func expandVariables(content string, variables Variables) (string, error) {
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
