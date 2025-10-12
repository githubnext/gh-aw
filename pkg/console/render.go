package console

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// RenderStruct renders a Go struct to console output using reflection and struct tags.
// It supports:
// - Rendering structs as markdown-style headers with key-value pairs
// - Rendering slices as tables using the console table renderer
// - Rendering maps as markdown headers
//
// Struct tags:
// - `console:"title:My Title"` - Sets the title for a section
// - `console:"header:Column Name"` - Sets the column header name for table columns
// - `console:"omitempty"` - Skips zero values
// - `console:"-"` - Skips the field entirely
func RenderStruct(v interface{}) string {
	var output strings.Builder
	renderValue(reflect.ValueOf(v), "", &output, 0)
	return output.String()
}

// renderValue recursively renders a reflect.Value to the output builder
func renderValue(val reflect.Value, title string, output *strings.Builder, depth int) {
	// Dereference pointers
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		renderStruct(val, title, output, depth)
	case reflect.Slice, reflect.Array:
		renderSlice(val, title, output, depth)
	case reflect.Map:
		renderMap(val, title, output, depth)
	}
}

// renderStruct renders a struct as markdown-style headers with key-value pairs
func renderStruct(val reflect.Value, title string, output *strings.Builder, depth int) {
	typ := val.Type()

	// Print title if provided
	if title != "" {
		if depth == 0 {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("# %s", title)))
		} else {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("%s %s", strings.Repeat("#", depth+1), title)))
		}
		output.WriteString("\n\n")
	}

	// Track the longest field name for alignment
	maxFieldLen := 0
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := parseConsoleTag(fieldType.Tag.Get("console"))

		if tag.skip || (tag.omitempty && isZeroValue(field)) {
			continue
		}

		fieldName := fieldType.Name
		if tag.header != "" {
			fieldName = tag.header
		}

		if len(fieldName) > maxFieldLen {
			maxFieldLen = len(fieldName)
		}
	}

	// Iterate through struct fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check if field should be skipped
		tag := parseConsoleTag(fieldType.Tag.Get("console"))
		if tag.skip {
			continue
		}

		// Check omitempty
		if tag.omitempty && isZeroValue(field) {
			continue
		}

		// Get field name (use tag header if available, otherwise use field name)
		fieldName := fieldType.Name
		if tag.header != "" {
			fieldName = tag.header
		}

		// Render based on field type
		if field.Kind() == reflect.Struct && field.Type().String() != "time.Time" {
			// Nested struct - render recursively with title (but not time.Time)
			subTitle := tag.title
			if subTitle == "" {
				subTitle = fieldName
			}
			renderValue(field, subTitle, output, depth+1)
		} else if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			// Slice - render as table
			sliceTitle := tag.title
			if sliceTitle == "" {
				sliceTitle = fieldName
			}
			renderValue(field, sliceTitle, output, depth+1)
		} else if field.Kind() == reflect.Map {
			// Map - render as headers
			mapTitle := tag.title
			if mapTitle == "" {
				mapTitle = fieldName
			}
			renderValue(field, mapTitle, output, depth+1)
		} else {
			// Simple field - render as key-value pair with proper alignment
			paddedName := fmt.Sprintf("%-*s", maxFieldLen, fieldName)
			output.WriteString(fmt.Sprintf("  %s: %v\n", paddedName, formatFieldValue(field)))
		}
	}

	output.WriteString("\n")
}

// renderSlice renders a slice as a table using the console table renderer
func renderSlice(val reflect.Value, title string, output *strings.Builder, depth int) {
	if val.Len() == 0 {
		return
	}

	// Print title
	if title != "" {
		if depth == 0 {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("# %s", title)))
		} else {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("%s %s", strings.Repeat("#", depth+1), title)))
		}
		output.WriteString("\n\n")
	}

	// Check if slice elements are structs (for table rendering)
	elemType := val.Type().Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() == reflect.Struct {
		// Render as table
		config := buildTableConfig(val, title)
		output.WriteString(RenderTable(config))
	} else {
		// Render as list
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			output.WriteString(fmt.Sprintf("  • %v\n", formatFieldValue(elem)))
		}
		output.WriteString("\n")
	}
}

// renderMap renders a map as markdown-style headers
func renderMap(val reflect.Value, title string, output *strings.Builder, depth int) {
	if val.Len() == 0 {
		return
	}

	// Print title
	if title != "" {
		if depth == 0 {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("# %s", title)))
		} else {
			output.WriteString(FormatInfoMessage(fmt.Sprintf("%s %s", strings.Repeat("#", depth+1), title)))
		}
		output.WriteString("\n\n")
	}

	// Render map entries
	for _, key := range val.MapKeys() {
		mapValue := val.MapIndex(key)
		output.WriteString(fmt.Sprintf("  %-18s %v\n", fmt.Sprintf("%v:", key), formatFieldValue(mapValue)))
	}
	output.WriteString("\n")
}

// buildTableConfig builds a TableConfig from a slice of structs
func buildTableConfig(val reflect.Value, title string) TableConfig {
	config := TableConfig{
		Title: "",
	}

	if val.Len() == 0 {
		return config
	}

	// Get the element type
	elemType := val.Type().Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// Build headers from struct fields
	var headers []string
	var fieldIndices []int

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		tag := parseConsoleTag(field.Tag.Get("console"))

		// Skip fields marked with "-"
		if tag.skip {
			continue
		}

		// Use header tag if available, otherwise use field name
		headerName := field.Name
		if tag.header != "" {
			headerName = tag.header
		}

		headers = append(headers, headerName)
		fieldIndices = append(fieldIndices, i)
	}

	config.Headers = headers

	// Build rows
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		// Dereference pointer if needed
		for elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				break
			}
			elem = elem.Elem()
		}

		if elem.Kind() != reflect.Struct {
			continue
		}

		var row []string
		for _, fieldIdx := range fieldIndices {
			field := elem.Field(fieldIdx)
			row = append(row, formatFieldValue(field))
		}
		config.Rows = append(config.Rows, row)
	}

	return config
}

// consoleTag represents parsed console struct tag
type consoleTag struct {
	title     string
	header    string
	omitempty bool
	skip      bool
}

// parseConsoleTag parses the console struct tag
func parseConsoleTag(tag string) consoleTag {
	result := consoleTag{}

	if tag == "-" {
		result.skip = true
		return result
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "omitempty" {
			result.omitempty = true
		} else if strings.HasPrefix(part, "title:") {
			result.title = strings.TrimPrefix(part, "title:")
		} else if strings.HasPrefix(part, "header:") {
			result.header = strings.TrimPrefix(part, "header:")
		}
	}

	return result
}

// isZeroValue checks if a reflect.Value is the zero value for its type
func isZeroValue(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}

	// Special handling for time.Time
	if val.Type().String() == "time.Time" {
		if val.CanInterface() {
			if t, ok := val.Interface().(time.Time); ok {
				return t.IsZero()
			}
		}
		// For unexported time.Time fields, we can't easily check, so assume not zero
		return false
	}

	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	}

	return false
}

// formatFieldValue formats a reflect.Value as a string for display
func formatFieldValue(val reflect.Value) string {
	// Dereference pointers
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "-"
		}
		val = val.Elem()
	}

	if !val.IsValid() {
		return "-"
	}

	// Handle zero values
	if isZeroValue(val) {
		// Special case: empty string should return "-", but 0 for numbers might be valid
		if val.Kind() == reflect.String {
			return "-"
		}
		// For numeric types, return "0" or the actual value
		if val.Kind() >= reflect.Int && val.Kind() <= reflect.Float64 {
			// For numeric types, we can safely use Interface()
			if val.CanInterface() {
				return fmt.Sprintf("%v", val.Interface())
			}
			// Fallback for unexported fields
			switch val.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return fmt.Sprintf("%d", val.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return fmt.Sprintf("%d", val.Uint())
			case reflect.Float32, reflect.Float64:
				return fmt.Sprintf("%f", val.Float())
			}
		}
		return "-"
	}

	// Special handling for time.Time to avoid unexported field panic
	if val.Type().String() == "time.Time" {
		// Can't use Interface() on unexported fields, so use Format method via reflection
		if val.CanInterface() {
			if timeVal, ok := val.Interface().(time.Time); ok {
				return timeVal.Format("2006-01-02 15:04:05")
			}
		}
		// For unexported time.Time fields, try to call the String method
		stringMethod := val.MethodByName("String")
		if stringMethod.IsValid() {
			result := stringMethod.Call(nil)
			if len(result) > 0 {
				return result[0].String()
			}
		}
		return val.Type().String() // return type name as fallback
	}

	// Only call Interface() if we can
	if !val.CanInterface() {
		// For unexported fields, try to format based on kind
		switch val.Kind() {
		case reflect.Bool:
			return fmt.Sprintf("%t", val.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("%d", val.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return fmt.Sprintf("%d", val.Uint())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprintf("%f", val.Float())
		case reflect.String:
			return val.String()
		default:
			return val.Type().String()
		}
	}

	return fmt.Sprintf("%v", val.Interface())
}
