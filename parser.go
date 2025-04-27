// Package ai_parser provides tools for parsing stochastic LLM outputs into structured data for agentic use.
// It is designed to be imported as a Go module in other AI projects, supporting flexible label definitions,
// multi-line and JSON fields, required/dependency validation, and robust handling of LLM output quirks.
package arkaineparser

import (
	"encoding/json" // For JSON field parsing
	"errors"
	"regexp"
	"strings"
)

// Label defines a label for parsing with options for required, data type, dependencies, JSON, and block start.
type Label struct {
	Name         string   // Name of the label (case-insensitive)
	Required     bool     // Whether this label is required
	DataType     string   // Data type (e.g. "text", "json")
	RequiredWith []string // List of other label names required with this one
	IsJSON       bool     // Whether this label should be parsed as JSON
	IsBlockStart bool     // Whether this label starts a new block
}

// Parser parses labeled sections from text input.
type Parser struct {
	labels   []Label
	patterns []labelPattern
	labelMap map[string]Label
}

type labelPattern struct {
	// Name of the label
	Name string
	// Regex pattern for the label
	Pattern *regexp.Regexp
}

// NewParser creates a new Parser with the given labels.
// Returns error if more than one block start label is defined.
func NewParser(labels []Label) (*Parser, error) {
	// Create a map of label names to label definitions
	labelMap := make(map[string]Label)
	// Count the number of block start labels
	blockStartCount := 0
	for i := range labels {
		// Convert label name to lowercase
		labels[i].Name = strings.ToLower(labels[i].Name)
		// Add label to map
		labelMap[labels[i].Name] = labels[i]
		// Increment block start count if label is a block start
		if labels[i].IsBlockStart {
			blockStartCount++
		}
	}
	// Check if more than one block start label is defined
	if blockStartCount > 1 {
		return nil, errors.New("Only one block start label is allowed")
	}
	// Build regex patterns for each label
	patterns := buildPatterns(labels)
	// Create a new Parser
	return &Parser{labels: labels, patterns: patterns, labelMap: labelMap}, nil
}

// buildPatterns constructs regex patterns for each label.
func buildPatterns(labels []Label) []labelPattern {
	// Create a list of regex patterns
	var patterns []labelPattern
	for _, label := range labels {
		// Create a regex pattern for the label
		labelRegex := strings.Join(strings.Fields(label.Name), `\\s+`)
		pattern := regexp.MustCompile(`(?i)^\\s*` + labelRegex + `\\s*[:~\-]+\\s*`)
		// Add pattern to list
		patterns = append(patterns, labelPattern{Name: label.Name, Pattern: pattern})
	}
	return patterns
}

// Parse parses the text into a map of label names (all lowercase) to their values. Each label can have a single value or a slice of values.
//   - Detects labels using regex patterns (case-insensitive, multiple separators)
//   - Collects multi-line values for labels
//   - Parses JSON fields if specified
//   - Validates required fields and dependencies
//   - Returns a map of results and a slice of error strings
func (p *Parser) Parse(text string) (map[string]interface{}, []string) {
	// Step 1: Clean the input text (remove markdown/code blocks, inline code)
	cleaned := cleanText(text)
	lines := splitAndTrimLines(cleaned)

	// Step 2: Initialize data structures
	// Map of label name (lowercase) to list of captured values
	data := make(map[string][]string)
	for _, label := range p.labels {
		data[label.Name] = []string{}
	}
	var (
		currentLabel string          // The label currently being populated
		currentEntry strings.Builder // Accumulates multiline values
	)

	// Step 3: Iterate over each line to parse labels and values
	for _, line := range lines {
		labelName, value := p.parseLine(line)
		if labelName != "" {
			// If we were collecting a previous entry, finalize it
			if currentLabel != "" {
				finalizeEntry(data, currentLabel, currentEntry.String())
				currentEntry.Reset()
			}
			currentLabel = strings.ToLower(labelName)
			currentEntry.WriteString(value)
		} else if currentLabel != "" {
			// Only treat as continuation if the line does not start with any known label
			isLabelLine := false
			for _, lbl := range p.labels {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), strings.ToLower(lbl.Name)+":") {
					isLabelLine = true
					break
				}
			}
			if !isLabelLine {
				if currentEntry.Len() > 0 {
					currentEntry.WriteString("\n")
				}
				currentEntry.WriteString(line)
			}
		}
	}
	// Finalize last entry if present
	if currentLabel != "" {
		finalizeEntry(data, currentLabel, currentEntry.String())
	}

	// Step 4: Process results: parse JSON fields, flatten single-value lists, collect errors
	results, errList := p.processResults(data)
	return results, errList
}

// cleanText removes markdown code blocks (```...```) and inline code (`...`) from the input text.
func cleanText(text string) string {
	// Remove markdown code blocks (```...```)
	codeBlock := regexp.MustCompile("(?s)```(?:\\w+)?\\s*(.*?)\\s*```")
	text = codeBlock.ReplaceAllStringFunc(text, func(match string) string {
		sub := codeBlock.FindStringSubmatch(match)
		if len(sub) > 1 {
			return sub[1]
		}
		return ""
	})
	// Remove inline code (`...`)
	inlineCode := regexp.MustCompile("`([^`]+)`")
	text = inlineCode.ReplaceAllString(text, "$1")
	return strings.TrimSpace(text)
}

// splitAndTrimLines splits text into lines and trims right whitespace.
func splitAndTrimLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return lines
}

// parseLine tries to match a label at the start of the line. Returns label name and value (if matched), else empty string.
func (p *Parser) parseLine(line string) (string, string) {
	// Try regex patterns for each label (case-insensitive)
	for _, pat := range p.patterns {
		if loc := pat.Pattern.FindStringIndex(line); loc != nil {
			value := strings.TrimSpace(line[loc[1]:])
			return pat.Name, value
		}
	}
	// Fallback: check for label prefix with separator
	for labelName := range p.labelMap {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), labelName) {
			remain := trimmed[len(labelName):]
			if sep, _ := regexp.MatchString(`^\s*[:~\-]+`, remain); sep {
				content := regexp.MustCompile(`^\s*[:~\-]+`).ReplaceAllString(remain, "")
				return labelName, strings.TrimSpace(content)
			} else {
				// treat as continuation
				return "", trimmed
			}
		}
	}
	// No match; treat as continuation
	return "", ""
}

// finalizeEntry appends a non-empty entry to the data map for a label.
func finalizeEntry(data map[string][]string, labelName, entry string) {
	content := strings.TrimSpace(entry)
	if content != "" {
		data[labelName] = append(data[labelName], content)
	}
}

// processResults parses JSON fields, flattens single-value lists, and collects errors.
func (p *Parser) processResults(rawData map[string][]string) (map[string]interface{}, []string) {
	results := make(map[string]interface{})
	errList := []string{}
	for labelName, entries := range rawData {
		labelDef := p.labelMap[labelName]
		parsedEntries := []interface{}{}
		for _, entry := range entries {
			if labelDef.IsJSON {
				// If entry is empty, treat as empty object
				if strings.TrimSpace(entry) == "" {
					parsedEntries = append(parsedEntries, map[string]interface{}{})
					continue
				}
				var obj interface{}
				if err := importJSONUnmarshal([]byte(entry), &obj); err != nil {
					parsedEntries = append(parsedEntries, entry)
					errList = append(errList, "JSON error in '"+labelDef.Name+"': "+err.Error())
				} else {
					parsedEntries = append(parsedEntries, obj)
				}
			} else {
				parsedEntries = append(parsedEntries, entry)
			}
		}
		// Flatten if only one entry
		if len(parsedEntries) == 1 {
			// If the entry is an empty string, flatten to ""
			if str, ok := parsedEntries[0].(string); ok && str == "" {
				results[labelName] = ""
			} else {
				results[labelName] = parsedEntries[0]
			}
		} else if len(parsedEntries) == 0 {
			// If no entries, flatten to ""
			results[labelName] = ""
		} else {
			results[labelName] = parsedEntries
		}
	}
	// Validate required fields and dependencies
	errList = append(errList, p.validateDependencies(rawData)...)
	return results, errList
}

// importJSONUnmarshal wraps json.Unmarshal for clarity and future flexibility.
func importJSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// validateDependencies checks required and required_with constraints.
func (p *Parser) validateDependencies(data map[string][]string) []string {
	errList := []string{}
	for _, label := range p.labels {
		key := strings.ToLower(label.Name)
		entries, present := data[key]
		// Treat empty string or empty slice as missing
		missing := !present || len(entries) == 0 || (len(entries) == 1 && entries[0] == "")
		if label.Required && missing {
			errList = append(errList, "'"+label.Name+"' is required")
		}
		if len(label.RequiredWith) > 0 {
			for _, dep := range label.RequiredWith {
				depKey := strings.ToLower(dep)
				depEntries, depPresent := data[depKey]
				depMissing := !depPresent || len(depEntries) == 0 || (len(depEntries) == 1 && depEntries[0] == "")
				// Enforce dependency if this label is present (even if empty)
				if present {
					if depMissing {
						errList = append(errList, "'"+label.Name+"' requires '"+dep+"'")
					}
				}
			}
		}
	}
	return errList
}

// ParseBlocks parses the text into blocks, splitting at the block start label.
// Each block is parsed as a separate document, and results are returned as a slice of maps.
// Errors are collected for each block and returned as a combined error list.
// Returns a slice of maps (one per block) and a slice of error strings.
func (p *Parser) ParseBlocks(text string) ([]map[string]interface{}, []string) {
	// Find the block start label (must be exactly one)
	blockLabel := ""
	for _, label := range p.labels {
		if label.IsBlockStart {
			blockLabel = label.Name
			break
		}
	}
	if blockLabel == "" {
		return nil, []string{"No block start label defined - must have at least one"}
	}

	// Clean and split input into lines
	cleaned := cleanText(text)
	lines := splitAndTrimLines(cleaned)

	var (
		blocks       [][]string // Each block is a slice of lines
		currentBlock []string
		inBlock      bool
	)

	// Iterate through lines, splitting at each new block start
	for _, line := range lines {
		labelName, _ := p.parseLine(line)
		if strings.ToLower(labelName) == blockLabel {
			if inBlock && len(currentBlock) > 0 {
				blocks = append(blocks, currentBlock)
				currentBlock = []string{}
			}
			inBlock = true
		}
		if inBlock {
			currentBlock = append(currentBlock, line)
		}
	}
	// Append last block if present
	if inBlock && len(currentBlock) > 0 {
		blocks = append(blocks, currentBlock)
	}

	// Parse each block using the normal Parse logic
	var (
		results []map[string]interface{}
		errList []string
	)
	for _, blockLines := range blocks {
		blockText := strings.Join(blockLines, "\n")
		result, blockErr := p.Parse(blockText)
		if len(blockErr) > 0 {
			errList = append(errList, blockErr...)
		}
		results = append(results, result)
	}
	return results, errList
}

// Additional helpers and logic to be implemented.
