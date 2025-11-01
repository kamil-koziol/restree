package restree

import (
	"testing"

	"github.com/kamil-koziol/restree/internal/assert"
)

func TestParseScriptEnvOutput(t *testing.T) {
	output := "FOO=bar\nBAZ=qux\nINVALID_LINE\nKEY=val=ue\n"
	envMap, err := parseScriptEnvOutput(output)
	assert.Eq(t, err, nil)

	expected := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
		"KEY": "val=ue", // Should support '=' in values after first '='
	}
	for k, v := range expected {
		assert.Eq(t, v, envMap[k])
	}

	_, ok := envMap["INVALID_LINE"]
	assert.Assert(t, !ok, "expected INVALID_LINE to be ignored")
}

func TestExpandVariables(t *testing.T) {
	variables := map[string]string{
		"name": "world",
		"foo":  "bar",
	}

	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"hello {{name}}", "hello world", false},
		{"foo {{foo}} baz", "foo bar baz", false},
		{"missing {{missing}}", "missing {{missing}}", true},
		{"no vars here", "no vars here", false},
	}

	for _, tt := range tests {
		got, err := expandVariables(tt.input, variables)
		assert.Eq(t, (err != nil), tt.wantErr)
		assert.Eq(t, got, tt.expected)
	}
}
