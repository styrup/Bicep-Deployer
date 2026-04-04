package bicep

import (
	"strings"
)

// ParamType describes the type of a Bicep parameter.
type ParamType string

const (
	TypeString ParamType = "string"
	TypeInt    ParamType = "int"
	TypeBool   ParamType = "bool"
	TypeObject ParamType = "object"
	TypeArray  ParamType = "array"
)

// Parameter represents a parsed Bicep `param` declaration.
type Parameter struct {
	Name           string    `json:"name"`
	Type           ParamType `json:"type"`
	DefaultValue   *string   `json:"defaultValue,omitempty"`
	Description    string    `json:"description,omitempty"`
	AllowedValues  []string  `json:"allowedValues,omitempty"`
	Required       bool      `json:"required"`
	ExpressionHint string    `json:"-"` // internal: hint for expression defaults
}

// ParseParameters extracts all parameter declarations from Bicep source.
func ParseParameters(source string) []Parameter {
	var params []Parameter
	var pendingDescription string
	var pendingAllowed []string

	lines := strings.Split(source, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// @description('...')
		if strings.HasPrefix(line, "@description(") {
			pendingDescription = extractQuoted(line)
			i++
			continue
		}

		// @allowed([...]) — may span multiple lines
		if strings.HasPrefix(line, "@allowed(") {
			pendingAllowed, i = parseAllowedBlock(lines, i)
			continue
		}

		// Skip other decorators
		if strings.HasPrefix(line, "@") {
			i++
			continue
		}

		// param <name> <type> [= <default>]
		if strings.HasPrefix(line, "param ") {
			p := parseParamLine(line)
			p.AllowedValues = pendingAllowed

			// Merge description: @description takes priority, append expression hint
			if pendingDescription != "" {
				p.Description = pendingDescription
			}
			// If default was an expression, the parser stored a hint in p.Description.
			// Prepend the @description if both exist.
			if pendingDescription != "" && p.ExpressionHint != "" {
				p.Description = pendingDescription + " (" + p.ExpressionHint + ")"
			} else if p.ExpressionHint != "" {
				p.Description = p.ExpressionHint
			}

			params = append(params, p)

			pendingDescription = ""
			pendingAllowed = nil
		} else if line != "" && !strings.HasPrefix(line, "//") {
			pendingDescription = ""
			pendingAllowed = nil
		}

		i++
	}

	return params
}

// parseAllowedBlock parses @allowed([...]) which may span multiple lines.
// Returns the extracted values and the next line index to process.
func parseAllowedBlock(lines []string, startIdx int) ([]string, int) {
	// Collect all text from @allowed( until the matching ])
	var buf strings.Builder
	i := startIdx
	for i < len(lines) {
		buf.WriteString(lines[i])
		buf.WriteString("\n")
		if strings.Contains(lines[i], "]") {
			i++
			break
		}
		i++
	}

	block := buf.String()
	start := strings.Index(block, "[")
	end := strings.LastIndex(block, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, i
	}

	inner := block[start+1 : end]

	// Split by newlines first, then extract quoted values from each line
	var values []string
	for _, line := range strings.Split(inner, "\n") {
		// Each line may contain one or more comma-separated values
		for _, part := range strings.Split(line, ",") {
			v := strings.TrimSpace(part)
			v = stripQuotes(v)
			if v != "" && !containsDuplicate(values, v) {
				values = append(values, v)
			}
		}
	}

	return values, i
}

// parseParamLine parses a single `param name type [= default]` line.
func parseParamLine(line string) Parameter {
	// Remove leading "param "
	rest := strings.TrimPrefix(line, "param ")
	parts := strings.Fields(rest)

	p := Parameter{}
	if len(parts) == 0 {
		return p
	}

	p.Name = parts[0]

	if len(parts) >= 2 {
		rawType := parts[1]
		// Handle union types like 'Standard_LRS' | 'Premium_LRS' — treat as string
		p.Type = normalizeType(rawType)
	}

	// Look for default value after '='
	if idx := strings.Index(rest, "="); idx != -1 {
		defaultRaw := strings.TrimSpace(rest[idx+1:])
		// Remove inline comments
		if ci := strings.Index(defaultRaw, "//"); ci != -1 {
			defaultRaw = strings.TrimSpace(defaultRaw[:ci])
		}
		defaultVal := stripQuotes(defaultRaw)

		// If the default is a Bicep expression (e.g. resourceGroup().location),
		// don't expose it as a literal default — mark as optional with hint.
		if isExpression(defaultVal) {
			p.Required = false
			p.ExpressionHint = "Default: " + defaultVal
		} else {
			p.DefaultValue = &defaultVal
			p.Required = false
		}
	} else {
		p.Required = true
	}

	return p
}

// normalizeType maps raw Bicep type strings to our ParamType enum.
func normalizeType(raw string) ParamType {
	switch strings.ToLower(raw) {
	case "int":
		return TypeInt
	case "bool":
		return TypeBool
	case "object":
		return TypeObject
	case "array":
		return TypeArray
	default:
		return TypeString
	}
}

// extractQuoted extracts the first single- or double-quoted string from a line.
func extractQuoted(line string) string {
	for _, quote := range []byte{'\'', '"'} {
		start := strings.IndexByte(line, quote)
		if start == -1 {
			continue
		}
		end := strings.IndexByte(line[start+1:], quote)
		if end == -1 {
			continue
		}
		return line[start+1 : start+1+end]
	}
	return ""
}

// stripQuotes removes surrounding single or double quotes from a value.
func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// isExpression returns true if the value looks like a Bicep expression
// rather than a literal (e.g. resourceGroup().location, subscription().id).
func isExpression(s string) bool {
	return strings.Contains(s, "(") || strings.Contains(s, ")")
}

func containsDuplicate(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
