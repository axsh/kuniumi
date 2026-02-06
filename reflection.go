package kuniumi

import (
	"context"
	"fmt"
	"reflect"
)

// FunctionMetadata holds information about a registered function.
type FunctionMetadata struct {
	Name        string
	Description string
	Args        []ArgMetadata
	Returns     []ReturnMetadata
	FnValue     reflect.Value
}

type ArgMetadata struct {
	Name        string
	Description string
	Type        reflect.Type
}

type ReturnMetadata struct {
	Description string
	Type        reflect.Type
}

// AnalyzeFunction inspects a function and returns its metadata.
func AnalyzeFunction(fn interface{}, name, desc string) (*FunctionMetadata, error) {
	val := reflect.ValueOf(fn)
	typ := val.Type()

	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %v", typ.Kind())
	}

	// Verify first argument is context.Context
	if typ.NumIn() == 0 || typ.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return nil, fmt.Errorf("function first argument must be context.Context")
	}

	// Verify last return is error
	if typ.NumOut() == 0 || typ.Out(typ.NumOut()-1) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf("function last return value must be error")
	}

	meta := &FunctionMetadata{
		Name:        name,
		Description: desc,
		FnValue:     val,
	}

	// Analyze Arguments (skip context)
	for i := 1; i < typ.NumIn(); i++ {
		meta.Args = append(meta.Args, ArgMetadata{
			Name: fmt.Sprintf("arg%d", i), // Default generic name, usage of WithArgs is recommended
			Type: typ.In(i),
		})
	}

	// Analyze Returns (skip error)
	for i := 0; i < typ.NumOut()-1; i++ {
		meta.Returns = append(meta.Returns, ReturnMetadata{
			Type: typ.Out(i),
		})
	}

	return meta, nil
}

// CallFunction invokes the function with a map of arguments.
func CallFunction(ctx context.Context, meta *FunctionMetadata, args map[string]interface{}) ([]interface{}, error) {
	in := []reflect.Value{reflect.ValueOf(ctx)}

	// Map generic arguments map to function input parameters
	for _, argMeta := range meta.Args {
		val, ok := args[argMeta.Name]
		if !ok {
			// Use zero value if argument is missing
			in = append(in, reflect.Zero(argMeta.Type))
			continue
		}

		// Convert interface{} value to target type
		targetVal := reflect.ValueOf(val)
		if !targetVal.IsValid() {
			in = append(in, reflect.Zero(argMeta.Type))
			continue
		}

		// Perform type conversion (e.g., float64 from JSON to int)
		targetType := argMeta.Type
		if targetVal.Type().ConvertibleTo(targetType) {
			in = append(in, targetVal.Convert(targetType))
		} else {
			if targetVal.Kind() == reflect.Float64 {
				// Special handling for JSON numbers (float64) to integer types
				switch targetType.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					in = append(in, reflect.ValueOf(int64(targetVal.Float())).Convert(targetType))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					in = append(in, reflect.ValueOf(uint64(targetVal.Float())).Convert(targetType))
				default:
					return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
				}
			} else {
				return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
			}
		}
	}

	// Invoke the function
	out := meta.FnValue.Call(in)

	// Check returned error
	errVal := out[len(out)-1]
	if !errVal.IsNil() {
		return nil, errVal.Interface().(error)
	}

	// Collect result values
	var results []interface{}
	for i := 0; i < len(out)-1; i++ {
		results = append(results, out[i].Interface())
	}

	return results, nil
}

// GenerateJSONSchema generates a JSON Schema for the function arguments.
func GenerateJSONSchema(meta *FunctionMetadata) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for _, arg := range meta.Args {
		schema := typeToSchema(arg.Type)
		if arg.Description != "" {
			schema["description"] = arg.Description
		}
		properties[arg.Name] = schema
		required = append(required, arg.Name)
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// typeToSchema converts a Go reflect.Type to a JSON Schema definition.
func typeToSchema(t reflect.Type) map[string]interface{} {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice:
		return map[string]interface{}{
			"type":  "array",
			"items": typeToSchema(t.Elem()),
		}
	case reflect.Struct:
		// Recursive struct analysis is not yet supported
		return map[string]interface{}{"type": "object"}
	default:
		return map[string]interface{}{"type": "string"} // Fallback
	}
}
