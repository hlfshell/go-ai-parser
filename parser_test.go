package arkaineparser

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Test scaffolding for parser, will load test cases from assets.

// TestBasicFunctionality verifies that the parser correctly parses a typical input with all fields present.
func TestBasicFunctionality(t *testing.T) {
	// Load input text from asset file
	input, err := os.ReadFile("assets/basic_functionality_input.txt")
	if err != nil {
		t.Fatalf("failed to read input asset: %v", err)
	}

	// Load expected output from asset file
	expectedBytes, err := os.ReadFile("assets/basic_functionality_output.json")
	if err != nil {
		t.Fatalf("failed to read output asset: %v", err)
	}
	var expected map[string]interface{}
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatalf("failed to unmarshal expected output: %v", err)
	}

	// Define parser labels as in the Python test
	labels := []Label{
		{Name: "Action Input", RequiredWith: []string{"Action"}, IsJSON: true},
		{Name: "Action", RequiredWith: []string{"Action Input"}},
		{Name: "Thought"},
		{Name: "Result", Required: true},
	}
	parser, err := NewParser(labels)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Parse the input
	result, errors := parser.Parse(string(input))
	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Compare result to expected output
	if !deepEqual(result, expected) {
		t.Errorf("result does not match expected.\nGot: %#v\nExpected: %#v", result, expected)
	}
}

// deepEqual is a helper for comparing parser outputs in tests.
// It recursively compares maps, slices, and primitive types for equality.
func deepEqual(a, b interface{}) bool {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal := b.(map[string]interface{})
		if len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if !deepEqual(v, bVal[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			if !deepEqual(aVal[i], bVal[i]) {
				return false
			}
		}
		return true
	// Handle slices of maps by converting both to []interface{} and comparing recursively
	case []map[string]interface{}:
		// Convert both a and b to []interface{}
		ai := make([]interface{}, len(aVal))
		for i := range aVal {
			ai[i] = aVal[i]
		}
		bi := make([]interface{}, len(b.([]map[string]interface{})))
		for i := range b.([]map[string]interface{}) {
			bi[i] = b.([]map[string]interface{})[i]
		}
		return deepEqual(ai, bi)
	default:
		return reflect.DeepEqual(a, b)
	}
}


// TestMixedCaseMultiline checks handling of mixed case labels and multiline values.
func TestMixedCaseMultiline(t *testing.T) {
	input, _ := os.ReadFile("assets/mixed_case_multiline_input.txt")
	expectedBytes, _ := os.ReadFile("assets/mixed_case_multiline_output.json")
	var expected map[string]interface{}
	json.Unmarshal(expectedBytes, &expected)
	labels := []Label{
		{Name: "Context"}, {Name: "Intention"}, {Name: "Role"}, {Name: "Action"}, {Name: "Outcome"}, {Name: "Notes"},
	}
	parser, _ := NewParser(labels)
	result, errors := parser.Parse(string(input))
	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
	if !deepEqual(result, expected) {
		t.Errorf("result mismatch.\nGot: %#v\nExpected: %#v", result, expected)
	}
}

// TestJSONAndMalformed checks JSON and malformed JSON parsing and error reporting.
func TestJSONAndMalformed(t *testing.T) {
	input, _ := os.ReadFile("assets/json_and_malformed_input.txt")
	expectedBytes, _ := os.ReadFile("assets/json_and_malformed_output.json")
	errorsBytes, _ := os.ReadFile("assets/json_and_malformed_errors.json")
	var expected map[string]interface{}
	var expectedErrors []string
	json.Unmarshal(expectedBytes, &expected)
	json.Unmarshal(errorsBytes, &expectedErrors)
	labels := []Label{
		{Name: "Config", IsJSON: true}, {Name: "Data", IsJSON: true}, {Name: "Description"},
	}
	parser, _ := NewParser(labels)
	result, errors := parser.Parse(string(input))
	if !deepEqual(result, expected) {
		t.Errorf("result mismatch.\nGot: %#v\nExpected: %#v", result, expected)
	}
	if len(errors) != len(expectedErrors) || (len(errors) > 0 && errors[0] != expectedErrors[0]) {
		t.Errorf("error mismatch.\nGot: %#v\nExpected: %#v", errors, expectedErrors)
	}
}

// TestRequiredDependency checks required and dependency validation.
func TestRequiredDependency(t *testing.T) {
	input, _ := os.ReadFile("assets/required_dependency_input.txt")
	expectedBytes, _ := os.ReadFile("assets/required_dependency_output.json")
	errorsBytes, _ := os.ReadFile("assets/required_dependency_errors.json")
	var expected map[string]interface{}
	var expectedErrors []string
	json.Unmarshal(expectedBytes, &expected)
	json.Unmarshal(errorsBytes, &expectedErrors)
	labels := []Label{
		{Name: "FieldA"}, {Name: "FieldB", RequiredWith: []string{"FieldA"}},
	}
	parser, _ := NewParser(labels)
	result, errors := parser.Parse(string(input))
	if !deepEqual(result, expected) {
		t.Errorf("result mismatch.\nGot: %#v\nExpected: %#v", result, expected)
	}
	if len(errors) != len(expectedErrors) || (len(errors) > 0 && errors[0] != expectedErrors[0]) {
		t.Errorf("error mismatch.\nGot: %#v\nExpected: %#v", errors, expectedErrors)
	}
}

// TestBlockParsing checks block parsing with multiple blocks.
func TestBlockParsing(t *testing.T) {
	input, _ := os.ReadFile("assets/block_parsing_input.txt")
	expectedBytes, _ := os.ReadFile("assets/block_parsing_output.json")
	var expected []map[string]interface{}
	json.Unmarshal(expectedBytes, &expected)
	labels := []Label{
		{Name: "Task", IsBlockStart: true}, {Name: "Input", IsJSON: true}, {Name: "Result"},
	}
	parser, _ := NewParser(labels)
	blocks, errors := parser.ParseBlocks(string(input))
	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
	if !deepEqual(blocks, expected) {
		t.Errorf("block result mismatch.\nGot: %#v\nExpected: %#v", blocks, expected)
	}
}

// ...additional tests matching Python test_parser.py
