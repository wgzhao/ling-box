package uuid

import (
	"fmt"

	guuid "github.com/google/uuid"
)

// Type represents a UUID version
type Type string

const (
	TypeV1 Type = "v1"
	TypeV4 Type = "v4"
	TypeV7 Type = "v7"
)

// DefaultType is the most commonly used UUID version
const DefaultType = TypeV4

// Generate creates one or more UUIDs of the specified type.
func Generate(n int, t Type) ([]string, error) {
	if n <= 0 {
		n = 1
	}

	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id, err := generateOne(t)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func generateOne(t Type) (string, error) {
	switch t {
	case TypeV1:
		id, err := guuid.NewUUID()
		if err != nil {
			return "", err
		}
		return id.String(), nil
	case TypeV7:
		id, err := guuid.NewV7()
		if err != nil {
			return "", err
		}
		return id.String(), nil
	case TypeV4:
		fallthrough
	default:
		id, err := guuid.NewRandom()
		if err != nil {
			return "", err
		}
		return id.String(), nil
	}
}

// IsValid checks whether a string is a valid UUID.
func IsValid(s string) bool {
	_, err := guuid.Parse(s)
	return err == nil
}

// Parse parses a UUID string into its standard format.
func Parse(s string) (string, error) {
	id, err := guuid.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid UUID: %w", err)
	}
	return id.String(), nil
}
