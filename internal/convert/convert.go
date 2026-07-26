package convert

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// YAMLToJSON converts YAML input to JSON output.
func YAMLToJSON(yamlInput []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	data = normalize(data)

	result, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("JSON marshal failed: %w", err)
	}
	return result, nil
}

// JSONToYAML converts JSON input to YAML output.
func JSONToYAML(jsonInput []byte) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(jsonInput, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	data = normalize(data)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("YAML marshal failed: %w", err)
	}
	encoder.Close()

	// Remove trailing newline for cleaner output
	result := bytes.TrimRight(buf.Bytes(), "\n")
	return result, nil
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
