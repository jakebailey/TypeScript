package packagejson

import "github.com/microsoft/TypeScript/tsc/internal/json"

type Expected[T any] struct {
	actualJSONType string
	Null           bool
	Valid          bool
	Value          T
}

func (e *Expected[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = Expected[T]{Null: true, actualJSONType: "null"}
		return nil
	}
	if json.Unmarshal(data, &e.Value) == nil {
		e.Valid = true
	}
	switch data[0] {
	case '"':
		e.actualJSONType = "string"
	case 't', 'f':
		e.actualJSONType = "boolean"
	case '[':
		e.actualJSONType = "array"
	case '{':
		e.actualJSONType = "object"
	default:
		e.actualJSONType = "number"
	}
	return nil
}

func (e *Expected[T]) IsPresent() bool {
	return e.actualJSONType != ""
}

func (e *Expected[T]) GetValue() (value T, ok bool) {
	return e.Value, e.Valid
}

func (e *Expected[T]) IsValid() bool {
	return e.Valid
}

func (e *Expected[T]) ExpectedJSONType() string {
	var zero T
	switch any(zero).(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case []any, []string:
		return "array"
	case map[string]any, map[string]string:
		return "object"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	default:
		return "unknown"
	}
}

func (e *Expected[T]) ActualJSONType() string {
	return e.actualJSONType
}

func ExpectedOf[T any](value T) Expected[T] {
	return Expected[T]{Value: value, Valid: true, actualJSONType: (*Expected[T])(nil).ExpectedJSONType()}
}
