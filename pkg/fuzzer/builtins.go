package fuzzer

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed lists/sqli-basic.txt
var sqliBasicList string

//go:embed lists/xss-basic.txt
var xssBasicList string

//go:embed lists/common-passwords.txt
var commonPasswordsList string

//go:embed lists/dir-names.txt
var dirNamesList string

// BuiltInPayloads returns the payload slice for the given built-in list. For
// NUMERIC_RANGE it generates the inclusive integer range [rangeMin, rangeMax].
func BuiltInPayloads(list BuiltInList, rangeMin, rangeMax int) []string {
	switch list {
	case BuiltInSQLiBasic:
		return splitLines(sqliBasicList)
	case BuiltInXSSBasic:
		return splitLines(xssBasicList)
	case BuiltInCommonPasswords:
		return splitLines(commonPasswordsList)
	case BuiltInDirNames:
		return splitLines(dirNamesList)
	case BuiltInNumericRange:
		if rangeMax < rangeMin {
			return nil
		}

		out := make([]string, 0, rangeMax-rangeMin+1)
		for i := rangeMin; i <= rangeMax; i++ {
			out = append(out, strconv.Itoa(i))
		}

		return out
	default:
		return nil
	}
}

// splitLines returns non-empty, non-comment lines.
func splitLines(s string) []string {
	var out []string

	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		out = append(out, line)
	}

	return out
}
