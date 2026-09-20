package validator

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const delimiter = "---"

// frontmatter is the parsed YAML header of a SKILL.md file.
type frontmatter struct {
	// fields maps each top-level key to its key and value nodes.
	fields map[string]field
	// lines is the total number of lines in the file.
	lines int
}

type field struct {
	key   *yaml.Node
	value *yaml.Node
	line  int
}

type parseError struct {
	line    int
	message string
}

// splitLines splits content on LF and strips a trailing CR from each line so
// that LF and CRLF files are handled identically.
func splitLines(content []byte) []string {
	raw := strings.Split(string(content), "\n")
	for i, l := range raw {
		raw[i] = strings.TrimSuffix(l, "\r")
	}
	return raw
}

// countLines returns the number of lines in content. A trailing newline does
// not start a new line.
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	n := bytes.Count(content, []byte("\n"))
	if content[len(content)-1] != '\n' {
		n++
	}
	return n
}

// parseFrontmatter validates the framing of the frontmatter and decodes it
// into typed YAML nodes. It never uses regular expressions.
func parseFrontmatter(content []byte) (*frontmatter, *parseError) {
	if !utf8.Valid(content) {
		return nil, &parseError{message: fmt.Sprintf(
			"SKILL.md is not valid UTF-8 (first invalid byte at offset %d)", firstInvalidUTF8(content))}
	}
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))

	fm := &frontmatter{fields: map[string]field{}, lines: countLines(content)}
	lines := splitLines(content)

	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != delimiter {
		return nil, &parseError{line: 1, message: "SKILL.md must begin with a YAML frontmatter delimiter '---' on line 1"}
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == delimiter {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, &parseError{line: 1, message: "frontmatter opened on line 1 has no closing '---' delimiter"}
	}

	body := strings.Join(lines[1:end], "\n")
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
		return nil, &parseError{line: yamlErrorLine(err.Error(), 1), message: "frontmatter is not valid YAML: " + yamlErrorText(err.Error())}
	}

	// An empty frontmatter block decodes to an empty document; treat it as an
	// empty mapping so the required-field rules report what is missing.
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return fm, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, &parseError{line: root.Line + 1, message: fmt.Sprintf("frontmatter must be a YAML mapping of fields, got %s", describeNode(root))}
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		name := k.Value
		if _, dup := fm.fields[name]; dup {
			return nil, &parseError{line: k.Line + 1, message: fmt.Sprintf("frontmatter key %q is defined more than once", name)}
		}
		fm.fields[name] = field{key: k, value: v, line: k.Line + 1}
	}
	return fm, nil
}

func firstInvalidUTF8(b []byte) int {
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			return i
		}
		i += size
	}
	return -1
}

// yamlErrorLine extracts the "line N" location from a yaml.v3 error message
// and converts it to a file line by adding offset. It returns 0 when unknown.
func yamlErrorLine(msg string, offset int) int {
	const marker = "line "
	idx := strings.Index(msg, marker)
	if idx == -1 {
		return 0
	}
	rest := msg[idx+len(marker):]
	endIdx := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if endIdx == -1 {
		endIdx = len(rest)
	}
	n, err := strconv.Atoi(rest[:endIdx])
	if err != nil {
		return 0
	}
	return n + offset
}

// yamlErrorText strips the "yaml: line N: " prefix from a yaml.v3 error so the
// message does not repeat a line number that the Line field already carries.
func yamlErrorText(msg string) string {
	msg = strings.TrimPrefix(msg, "yaml: ")
	if strings.HasPrefix(msg, "line ") {
		if idx := strings.Index(msg, ": "); idx != -1 {
			return msg[idx+2:]
		}
	}
	return msg
}

// isString reports whether the node is a YAML string scalar. Quoted numbers
// such as "1.0" resolve to strings; bare numbers, booleans, nulls, and
// timestamps do not.
func isString(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!str"
}

// describeNode returns a human-readable type name for error messages.
func describeNode(n *yaml.Node) string {
	if n == nil {
		return "nothing"
	}
	switch n.Kind {
	case yaml.SequenceNode:
		return "a sequence"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.AliasNode:
		return "an alias"
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return "a string"
		case "!!int":
			return "an integer"
		case "!!float":
			return "a float"
		case "!!bool":
			return "a boolean"
		case "!!null":
			return "null"
		case "!!timestamp":
			return "a timestamp"
		case "!!binary":
			return "binary data"
		default:
			return "a " + strings.TrimPrefix(n.Tag, "!!") + " scalar"
		}
	}
	return "an unknown value"
}
