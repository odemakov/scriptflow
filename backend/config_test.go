package main

import (
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
