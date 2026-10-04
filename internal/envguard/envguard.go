// Package envguard decides whether a Claude Code tool call touches .env* files.
package envguard

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ToolCall is the subset of the PreToolUse hook payload we inspect.
type ToolCall struct {
	ToolInput map[string]any `json:"tool_input"`
}

// Fields of tool_input that may reference a path, pattern or shell command.
var fields = []string{"file_path", "path", "notebook_path", "pattern", "glob", "command"}

const (
	boundary = "/\\s\"'=:<>|;&(`"
	// Characters that, directly after ".env.example", mean it is not the
	// whole segment (e.g. .env.example.local, .env.example*).
	wordish = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.*/-"
	example = ".env.example"
)

// ".env" at the start of a path segment / shell word, e.g.
// .env, .env.local, ./.env.example, src/.env, cat .env*, --env-file=.env
var envRe = regexp.MustCompile(`(?m)(^|[` + boundary + `])\.env([^A-Za-z0-9_/-]|$|[.*])`)

var boundaryRe = regexp.MustCompile("[" + boundary + "]")

// Targets extracts the strings to inspect from a raw hook payload.
func Targets(payload []byte) ([]string, error) {
	var call ToolCall
	if err := json.Unmarshal(payload, &call); err != nil {
		return nil, err
	}
	var out []string
	for _, f := range fields {
		if s, ok := call.ToolInput[f].(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// stripExample removes example occurrences that are a whole path segment,
// so they are allowed.
func stripExample(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, example)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := i + len(example)
		startOK := i == 0 || boundaryRe.MatchString(s[i-1:i])
		endOK := end == len(s) || !strings.ContainsRune(wordish, rune(s[end]))
		if startOK && endOK {
			b.WriteString(s[:i])
		} else {
			b.WriteString(s[:end])
		}
		s = s[end:]
	}
}

// Blocked reports whether any target references a .env* file.
func Blocked(targets []string) bool {
	joined := stripExample(strings.Join(targets, "\n"))
	return envRe.MatchString(joined)
}
