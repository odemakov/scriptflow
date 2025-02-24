package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		params   dbx.Params
		expected []string
	}{
		{
			dbx.Params{"name": "test", "age": 30},
			[]string{"{:age}", "{:name}"},
		},
		{
			dbx.Params{"username": "admin", "password": "secret"},
			[]string{"{:password}", "{:username}"},
		},
		{
			dbx.Params{"host": "localhost", "port": 5432},
			[]string{"{:host}", "{:port}"},
		},
		{
			dbx.Params{"key1": "value1", "key2": "value2", "key3": "value3"},
			[]string{"{:key1}", "{:key2}", "{:key3}"},
		},
		{
			dbx.Params{"a": 1, "b": 2, "c": 3},
			[]string{"{:a}", "{:b}", "{:c}"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("params=%v", test.params), func(t *testing.T) {
			result := placeholders(test.params)
			for i, v := range result {
				if v != test.expected[i] {
					t.Errorf("expected placeholder: %v, got: %v", test.expected[i], v)
				}
			}
		})
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		params   dbx.Params
		expected []string
	}{
		{
			dbx.Params{"name": "test", "age": 30},
			[]string{"age", "name"},
		},
		{
			dbx.Params{"username": "admin", "password": "secret"},
			[]string{"password", "username"},
		},
		{
			dbx.Params{"host": "localhost", "port": 5432},
			[]string{"host", "port"},
		},
		{
			dbx.Params{"key1": "value1", "key2": "value2", "key3": "value3"},
			[]string{"key1", "key2", "key3"},
		},
		{
			dbx.Params{"a": 1, "b": 2, "c": 3},
			[]string{"a", "b", "c"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("params=%v", test.params), func(t *testing.T) {
			result := keys(test.params)
			for i, v := range result {
				if v != test.expected[i] {
					t.Errorf("expected key: %v, got: %v", test.expected[i], v)
				}
			}
		})
	}
}
func TestSubscriptionFilterOut(t *testing.T) {
	tests := []struct {
		configValues   []string
		correctValues  []string
		expectedResult []string
	}{
		{
			[]string{"started", "completed", "invalid"},
			[]string{"started", "error", "completed", "interrupted", "internal_error"},
			[]string{"started", "completed"},
		},
		{
			[]string{"error", "interrupted"},
			[]string{"started", "error", "completed", "interrupted", "internal_error"},
			[]string{"error", "interrupted"},
		},
		{
			[]string{"internal_error", "unknown"},
			[]string{"started", "error", "completed", "interrupted", "internal_error"},
			[]string{"internal_error"},
		},
		{
			[]string{"started", "completed"},
			[]string{"started", "completed"},
			[]string{"started", "completed"},
		},
		{
			[]string{"invalid1", "invalid2"},
			[]string{"started", "error", "completed", "interrupted", "internal_error"},
			[]string{},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("configValues=%v", test.configValues), func(t *testing.T) {
			sf := &ScriptFlow{}
			result, err := sf.subscriptionFilterOut(&test.configValues, &test.correctValues)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var resultSlice []string
			err = json.Unmarshal(result, &resultSlice)
			if err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if len(resultSlice) != len(test.expectedResult) {
				t.Errorf("expected result length: %v, got: %v", len(test.expectedResult), len(resultSlice))
			}

			for i, v := range resultSlice {
				if v != test.expectedResult[i] {
					t.Errorf("expected result: %v, got: %v", test.expectedResult[i], v)
				}
			}
		})
	}
}

func TestGenerateIdFromName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "with special characters",
			input:    "Hello! @#$% World",
			expected: "hello-world",
		},
		{
			name:     "with numbers",
			input:    "Hello123 World456",
			expected: "hello123-world456",
		},
		{
			name:     "multiple spaces and dashes",
			input:    "Hello   ---   World",
			expected: "hello-world",
		},
		{
			name:     "leading and trailing special chars",
			input:    "---Hello World!!!",
			expected: "hello-world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "starts from digit",
			input:    "123 Hello 1 World",
			expected: "hello-1-world",
		},
		{
			name:     "Random task 2",
			input:    "Random task 2",
			expected: "random-task-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateIdFromName(tt.input)
			if got != tt.expected {
				t.Errorf("generateIdFromName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
