package weconnect

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// extractJSONObject returns the first balanced { ... } object found at or after
// from. String contents (single or double quoted) and escapes are respected so
// braces inside strings do not affect the nesting count.
func extractJSONObject(s string, from int) (string, error) {
	start := strings.IndexByte(s[from:], '{')
	if start < 0 {
		return "", errors.New("no object literal found")
	}
	start += from

	var depth int
	var inStr bool
	var quote byte
	var escaped bool

	for i := start; i < len(s); i++ {
		c := s[i]

		if inStr {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == quote:
				inStr = false
			}
			continue
		}

		switch c {
		case '"', '\'':
			inStr = true
			quote = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], nil
			}
		}
	}

	return "", errors.New("unbalanced object literal")
}

var (
	// single-quoted string values -> double quoted (JS object literals)
	reBareKey       = regexp.MustCompile(`([{,]\s*)([A-Za-z_][A-Za-z0-9_]*)(\s*):`)
	reTrailingComma = regexp.MustCompile(`,(\s*[}\]])`)
)

// unmarshalRelaxed decodes a JSON object that may use JavaScript object-literal
// conventions (bare keys, single-quoted strings, trailing commas). Well-formed
// JSON is decoded directly; only on failure is a normalization pass applied.
func unmarshalRelaxed(obj string, v any) error {
	if err := json.Unmarshal([]byte(obj), v); err == nil {
		return nil
	}

	normalized := normalizeJSObject(obj)
	return json.Unmarshal([]byte(normalized), v)
}

// normalizeJSObject converts a JavaScript object literal into JSON: single
// quotes to double quotes, bare identifier keys to quoted keys and trailing
// commas removed. It is a best-effort transform for the VW IDKit payloads.
func normalizeJSObject(obj string) string {
	obj = strings.ReplaceAll(obj, `'`, `"`)
	obj = reBareKey.ReplaceAllString(obj, `$1"$2"$3:`)
	obj = reTrailingComma.ReplaceAllString(obj, `$1`)
	return obj
}
