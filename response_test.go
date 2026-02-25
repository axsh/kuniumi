package kuniumi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSuccessResponse(t *testing.T) {
	tests := []struct {
		name    string
		results []any
		want    map[string]any
	}{
		{
			name:    "single return value",
			results: []any{10},
			want:    map[string]any{"result": 10},
		},
		{
			name:    "multiple return values",
			results: []any{10, "hello"},
			want:    map[string]any{"result0": 10, "result1": "hello"},
		},
		{
			name:    "no return values (empty slice)",
			results: []any{},
			want:    map[string]any{},
		},
		{
			name:    "nil results",
			results: nil,
			want:    map[string]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSuccessResponse(tt.results)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want map[string]any
	}{
		{
			name: "simple error message",
			msg:  "something went wrong",
			want: map[string]any{"error": "something went wrong"},
		},
		{
			name: "empty message",
			msg:  "",
			want: map[string]any{"error": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildErrorResponse(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}
