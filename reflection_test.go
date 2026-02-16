package kuniumi

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStringToType(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetType reflect.Type
		wantValue  interface{}
		wantErr    bool
	}{
		// int variants
		{"string to int", "42", reflect.TypeOf(int(0)), int(42), false},
		{"string to int8", "127", reflect.TypeOf(int8(0)), int8(127), false},
		{"string to int16", "1000", reflect.TypeOf(int16(0)), int16(1000), false},
		{"string to int32", "100000", reflect.TypeOf(int32(0)), int32(100000), false},
		{"string to int64", "9999999", reflect.TypeOf(int64(0)), int64(9999999), false},
		// uint variants
		{"string to uint", "10", reflect.TypeOf(uint(0)), uint(10), false},
		{"string to uint8", "255", reflect.TypeOf(uint8(0)), uint8(255), false},
		{"string to uint16", "65535", reflect.TypeOf(uint16(0)), uint16(65535), false},
		{"string to uint32", "100000", reflect.TypeOf(uint32(0)), uint32(100000), false},
		{"string to uint64", "100000", reflect.TypeOf(uint64(0)), uint64(100000), false},
		// float variants
		{"string to float32", "2.5", reflect.TypeOf(float32(0)), float32(2.5), false},
		{"string to float64", "3.14", reflect.TypeOf(float64(0)), float64(3.14), false},
		// bool variants
		{"string to bool true", "true", reflect.TypeOf(false), true, false},
		{"string to bool false", "false", reflect.TypeOf(false), false, false},
		{"string to bool 1", "1", reflect.TypeOf(false), true, false},
		{"string to bool 0", "0", reflect.TypeOf(false), false, false},
		// error cases
		{"invalid string to int", "abc", reflect.TypeOf(int(0)), nil, true},
		{"invalid string to float", "xyz", reflect.TypeOf(float64(0)), nil, true},
		{"invalid string to bool", "maybe", reflect.TypeOf(false), nil, true},
		{"negative string to uint", "-1", reflect.TypeOf(uint(0)), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertStringToType(tt.input, tt.targetType)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, got.Interface())
		})
	}
}

// Test target function for CallFunction tests
func addInts(ctx context.Context, x int, y int) (int, error) {
	return x + y, nil
}

func TestCallFunction_StringArgs(t *testing.T) {
	meta, err := AnalyzeFunction(addInts, "addInts", "test add")
	require.NoError(t, err)
	// Apply param names
	meta.Args[0].Name = "x"
	meta.Args[1].Name = "y"

	tests := []struct {
		name    string
		args    map[string]interface{}
		want    int
		wantErr bool
	}{
		{
			name: "string values",
			args: map[string]interface{}{"x": "10", "y": "20"},
			want: 30,
		},
		{
			name: "float64 values (existing behavior)",
			args: map[string]interface{}{"x": float64(5), "y": float64(3)},
			want: 8,
		},
		{
			name: "int values (direct ConvertibleTo)",
			args: map[string]interface{}{"x": 7, "y": 3},
			want: 10,
		},
		{
			name:    "invalid string value",
			args:    map[string]interface{}{"x": "abc", "y": "20"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := CallFunction(context.Background(), meta, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, tt.want, results[0])
		})
	}
}

// Test helper functions for GenerateOutputJSONSchema
func singleReturnFunc(ctx context.Context, x int) (string, error) { return "", nil }
func multiReturnFunc(ctx context.Context) (int, string, error)    { return 0, "", nil }
func noReturnFunc(ctx context.Context) error                      { return nil }

func TestGenerateOutputJSONSchema(t *testing.T) {
	t.Run("single return without description", func(t *testing.T) {
		meta, err := AnalyzeFunction(singleReturnFunc, "singleReturn", "test")
		require.NoError(t, err)

		schema := GenerateOutputJSONSchema(meta)
		require.NotNil(t, schema, "schema should not be nil for function with return value")

		assert.Equal(t, "object", schema["type"])

		props, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok, "schema should have properties")

		resultProp, ok := props["result"].(map[string]interface{})
		require.True(t, ok, "properties should contain 'result'")
		assert.Equal(t, "string", resultProp["type"])
		assert.Nil(t, resultProp["description"], "description should not be set")
	})

	t.Run("single return with description", func(t *testing.T) {
		meta, err := AnalyzeFunction(singleReturnFunc, "singleReturn", "test")
		require.NoError(t, err)
		meta.Returns[0].Description = "test description"

		schema := GenerateOutputJSONSchema(meta)
		require.NotNil(t, schema)

		props := schema["properties"].(map[string]interface{})
		resultProp := props["result"].(map[string]interface{})
		assert.Equal(t, "string", resultProp["type"])
		assert.Equal(t, "test description", resultProp["description"])
	})

	t.Run("multiple returns", func(t *testing.T) {
		meta, err := AnalyzeFunction(multiReturnFunc, "multiReturn", "test")
		require.NoError(t, err)

		schema := GenerateOutputJSONSchema(meta)
		require.NotNil(t, schema)

		assert.Equal(t, "object", schema["type"])

		props, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok)

		result0, ok := props["result0"].(map[string]interface{})
		require.True(t, ok, "properties should contain 'result0'")
		assert.Equal(t, "integer", result0["type"])

		result1, ok := props["result1"].(map[string]interface{})
		require.True(t, ok, "properties should contain 'result1'")
		assert.Equal(t, "string", result1["type"])
	})

	t.Run("no return (error only)", func(t *testing.T) {
		meta, err := AnalyzeFunction(noReturnFunc, "noReturn", "test")
		require.NoError(t, err)

		schema := GenerateOutputJSONSchema(meta)
		assert.Nil(t, schema, "schema should be nil for error-only function")
	})
}
