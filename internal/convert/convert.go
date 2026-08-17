package convert

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Parse parses input in the given format (json, yaml, csv) into a
// normalized value. JSON and YAML parse into their native Go shapes
// (maps of string keys); CSV parses as an array of objects with string
// values (header row + data rows).
func Parse(format string, input []byte) (interface{}, error) {
	switch format {
	case "json":
		var data interface{}
		if err := json.Unmarshal(input, &data); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return normalize(data), nil
	case "yaml":
		var data interface{}
		if err := yaml.Unmarshal(input, &data); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		return normalize(data), nil
	case "csv":
		return parseCSV(input)
	default:
		return nil, fmt.Errorf("unsupported input format %q (supported: json, yaml, csv)", format)
	}
}

// Render serializes a parsed value in the given format (json, yaml,
// csv, markdown). CSV and markdown only accept arrays of objects (or,
// for markdown, also scalar arrays and a top-level object).
func Render(format string, data interface{}) ([]byte, error) {
	switch format {
	case "json":
		out, err := json.MarshalIndent(normalize(data), "", "  ")
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return out, nil
	case "yaml":
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		if err := encoder.Encode(normalize(data)); err != nil {
			return nil, fmt.Errorf("YAML marshal failed: %w", err)
		}
		encoder.Close()

		// Remove trailing newline for cleaner output
		return bytes.TrimRight(buf.Bytes(), "\n"), nil
	case "csv":
		return renderCSV(data)
	case "markdown":
		return renderMarkdown(data)
	default:
		return nil, fmt.Errorf("unsupported output format %q (supported: json, yaml, csv, markdown)", format)
	}
}

// YAMLToJSON converts YAML input to JSON output.
func YAMLToJSON(yamlInput []byte) ([]byte, error) {
	data, err := Parse("yaml", yamlInput)
	if err != nil {
		return nil, err
	}
	return Render("json", data)
}

// JSONToYAML converts JSON input to YAML output.
func JSONToYAML(jsonInput []byte) ([]byte, error) {
	data, err := Parse("json", jsonInput)
	if err != nil {
		return nil, err
	}
	return Render("yaml", data)
}

// DetectFormat detects whether the input is JSON or YAML by trying to parse it.
// Returns "json", "yaml", or an error if neither.
func DetectFormat(input []byte) (string, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("empty input")
	}

	// Try JSON first
	var js interface{}
	if json.Unmarshal(trimmed, &js) == nil {
		return "json", nil
	}

	// Try YAML
	var ys interface{}
	if yaml.Unmarshal(trimmed, &ys) == nil {
		return "yaml", nil
	}

	return "", fmt.Errorf("unable to detect format: not valid JSON or YAML")
}

// ConvertAuto detects the input format and converts to the other.
func ConvertAuto(input []byte) ([]byte, string, string, error) {
	format, err := DetectFormat(input)
	if err != nil {
		return nil, "", "", err
	}

	switch format {
	case "json":
		out, err := JSONToYAML(input)
		return out, "json", "yaml", err
	case "yaml":
		out, err := YAMLToJSON(input)
		return out, "yaml", "json", err
	default:
		return nil, "", "", fmt.Errorf("unsupported format: %s", format)
	}
}

// normalize converts map[interface{}]interface{} (from YAML) to map[string]interface{} (for JSON)
// and handles other type differences between YAML and JSON.
func normalize(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			s, ok := key.(string)
			if !ok {
				s = fmt.Sprintf("%v", key)
			}
			m[s] = normalize(val)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			m[key] = normalize(val)
		}
		return m
	case []interface{}:
		for i, val := range v {
			v[i] = normalize(val)
		}
		return v
	default:
		return data
	}
}
