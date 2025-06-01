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

const HeadersFileName = "_headers.http"
const BeforeScriptFileName = "_before.sh"

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
	content, err := expandEnv(string(template), envutil.All())
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

	content, err := expandEnv(string(b), envutil.All())
	if err != nil {
		return nil, fmt.Errorf("Failed to fill template: %s\n", err)
	}

	parsed, err := httpparser.ParseHeadersFile(bytes.NewBufferString(content))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %s: \n%s\n", err, content)
	}

	return parsed, nil
}

func runScript(scriptPath string) (string, string, error) {
	cmd := exec.Command("bash", scriptPath) // Use "sh" if preferred

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return "", "", err
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
			os.Setenv(key, value)
		}
	}

	// run parse the headers
	if headersPath != "" {
		headers, err = LoadHTTPHeaders(headersPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to load template %s: %s", headersPath, err)
		}
	}

	return headers, nil
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
		directoryHeaders, err := processDirectory(currentPath)
		if err != nil {
			return nil, fmt.Errorf("unable to process dir: %s", err)
		}
		maps.Copy(headers, directoryHeaders)

	}

	httpFile, err := LoadHTTPRequest(to)
	if err != nil {
		return nil, fmt.Errorf("Failed to load file %s: %s\n", to, err)
	}

	maps.Copy(httpFile.Headers, headers)
	return httpFile, nil
}

func expandEnv(content string, values map[string]string) (string, error) {
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
