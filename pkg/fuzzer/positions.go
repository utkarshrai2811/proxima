package fuzzer

import (
	"fmt"
	"strings"
)

const marker = "§"

// Position represents a §name§ marker in a request template.
type Position struct {
	Name  string
	Start int // byte offset of the opening §
	End   int // byte offset after the closing §
}

// ParsePositions finds all §name§ markers in the template, in order.
func ParsePositions(template string) ([]Position, error) {
	var positions []Position

	i := 0
	for i < len(template) {
		rel := strings.Index(template[i:], marker)
		if rel == -1 {
			break
		}

		start := i + rel

		rel2 := strings.Index(template[start+len(marker):], marker)
		if rel2 == -1 {
			return nil, fmt.Errorf("fuzzer: unclosed § marker at byte %d", start)
		}

		end := start + len(marker) + rel2
		name := template[start+len(marker) : end]

		positions = append(positions, Position{
			Name:  name,
			Start: start,
			End:   end + len(marker),
		})

		i = end + len(marker)
	}

	return positions, nil
}

// Substitute replaces each §name§ marker with the provided value. Markers
// without a value are replaced with the empty string.
func Substitute(template string, values map[string]string) string {
	result := template

	for name, val := range values {
		result = strings.ReplaceAll(result, marker+name+marker, val)
	}

	return result
}

// PositionNames returns the distinct position names in order of first appearance.
func PositionNames(template string) ([]string, error) {
	positions, err := ParsePositions(template)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)

	var names []string

	for _, p := range positions {
		if !seen[p.Name] {
			seen[p.Name] = true

			names = append(names, p.Name)
		}
	}

	return names, nil
}
