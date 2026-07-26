package convert

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const testYAML = `
name: ling-box
version: "1.0"
features:
  - url
  - base64
  - bcrypt
settings:
  timeout: 30
  verbose: true
`

const testJSON = `{
  "name": "ling-box",
  "version": "1.0",
  "features": ["url", "base64", "bcrypt"],
  "settings": {
    "timeout": 30,
    "verbose": true
  }
}`

func TestYAMLToJSON(t *testing.T) {
	out, err := YAMLToJSON([]byte(testYAML))
	if err != nil {
		t.Fatalf("YAMLToJSON returned error: %v", err)
	}

	var result interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(out))
	}

	m := result.(map[string]interface{})
	if m["name"] != "ling-box" {
		t.Errorf("expected name=ling-box, got %v", m["name"])
	}
	if m["version"] != "1.0" {
		t.Errorf("expected version=1.0, got %v", m["version"])
	}
}

func TestJSONToYAML(t *testing.T) {
	out, err := JSONToYAML([]byte(testJSON))
	if err != nil {
		t.Fatalf("JSONToYAML returned error: %v", err)
	}

	var result interface{}
	if err := yaml.Unmarshal(out, &result); err != nil {
		t.Fatalf("output is not valid YAML: %v\n%s", err, string(out))
	}

	m := result.(map[string]interface{})
	if m["name"] != "ling-box" {
		t.Errorf("expected name=ling-box, got %v", m["name"])
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{testJSON, "json"},
		{testYAML, "yaml"},
		{`{"a":1}`, "json"},
		{`a: 1`, "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, err := DetectFormat([]byte(tt.input))
			if err != nil {
				t.Fatalf("DetectFormat returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectFormat = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectFormatErrors(t *testing.T) {
	_, err := DetectFormat([]byte(""))
	if err == nil {
		t.Error("expected error for empty input")
	}

	_, err = DetectFormat([]byte("{{invalid}}"))
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestConvertAutoJSONtoYAML(t *testing.T) {
	out, from, to, err := ConvertAuto([]byte(testJSON))
	if err != nil {
		t.Fatalf("ConvertAuto returned error: %v", err)
	}
	if from != "json" {
		t.Errorf("expected from=json, got %s", from)
	}
	if to != "yaml" {
		t.Errorf("expected to=yaml, got %s", to)
	}
	if !strings.Contains(string(out), "ling-box") {
		t.Errorf("output should contain ling-box:\n%s", string(out))
	}
}

func TestConvertAutoYAMLtoJSON(t *testing.T) {
	out, from, to, err := ConvertAuto([]byte(testYAML))
	if err != nil {
		t.Fatalf("ConvertAuto returned error: %v", err)
	}
	if from != "yaml" {
		t.Errorf("expected from=yaml, got %s", from)
	}
	if to != "json" {
		t.Errorf("expected to=json, got %s", to)
	}
	if !strings.Contains(string(out), "ling-box") {
		t.Errorf("output should contain ling-box:\n%s", string(out))
	}
}

func TestRoundTrip(t *testing.T) {
	yamlOut, err := JSONToYAML([]byte(testJSON))
	if err != nil {
		t.Fatalf("JSONToYAML: %v", err)
	}

	jsonOut, err := YAMLToJSON(yamlOut)
	if err != nil {
		t.Fatalf("YAMLToJSON: %v", err)
	}

	var expected, result interface{}
	json.Unmarshal([]byte(testJSON), &expected)
	json.Unmarshal(jsonOut, &result)

	expMap := expected.(map[string]interface{})
	resMap := result.(map[string]interface{})

	if expMap["name"] != resMap["name"] {
		t.Errorf("round trip name mismatch: %v vs %v", expMap["name"], resMap["name"])
	}
}
