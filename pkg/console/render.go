package console

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var renderLog = logger.New("console:render")

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
func RenderStruct(v any) string {
	renderLog.Printf("Rendering struct: type=%T", v)
	var output strings.Builder
	renderValue(reflect.ValueOf(v), "", &output, 0)
	renderLog.Printf("Struct rendering complete: output_size=%d bytes", output.Len())
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
	renderLog.Printf("Rendering struct: type=%s, title=%s, depth=%d, fields=%d", typ.Name(), title, depth, val.NumField())

	// Print title without FormatInfoMessage styling
	if title != "" {
		if depth == 0 {
			fmt.Fprintf(output, "# %s\n\n", title)
		} else {
			fmt.Fprintf(output, "%s %s\n\n", strings.Repeat("#", depth+1), title)
		}
	}

	maxFieldLen := computeMaxFieldLen(val, typ)

	// Iterate through struct fields
	for i := range val.NumField() {
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

		renderStructField(field, fieldName, tag, maxFieldLen, output, depth)
	}

	output.WriteString("\n")
}

// computeMaxFieldLen computes the longest visible field name for alignment.
func computeMaxFieldLen(val reflect.Value, typ reflect.Type) int {
	maxFieldLen := 0
	for i := range val.NumField() {
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
	return maxFieldLen
}

// renderStructField renders a single struct field to output, dispatching on its kind.
func renderStructField(field reflect.Value, fieldName string, tag consoleTag, maxFieldLen int, output *strings.Builder, depth int) {
	// Dereference pointer to check underlying type
	fieldToCheck := field
	if field.Kind() == reflect.Ptr && !field.IsNil() {
		fieldToCheck = field.Elem()
	}

	subTitle := tag.title
	if subTitle == "" {
		subTitle = fieldName
	}

	switch {
	case fieldToCheck.Kind() == reflect.Struct && fieldToCheck.Type().String() != "time.Time":
		// Nested struct – render recursively
		renderValue(field, subTitle, output, depth+1)
	case fieldToCheck.Kind() == reflect.Slice || fieldToCheck.Kind() == reflect.Array:
		// Slice – render as table
		renderValue(field, subTitle, output, depth+1)
	case fieldToCheck.Kind() == reflect.Map:
		// Map – render as headers
		renderValue(field, subTitle, output, depth+1)
	default:
		// Simple field – render as key-value pair with alignment
		paddedName := fmt.Sprintf("%-*s", maxFieldLen, fieldName)
		fmt.Fprintf(output, "  %s: %v\n", paddedName, formatFieldValueWithTag(field, tag))
	}
}

// renderSlice renders a slice as a table using the console table renderer
func renderSlice(val reflect.Value, title string, output *strings.Builder, depth int) {
	if val.Len() == 0 {
		return
	}

	renderLog.Printf("Rendering slice: title=%s, length=%d, element_type=%s", title, val.Len(), val.Type().Elem().Name())

	// Print title without FormatInfoMessage styling
	if title != "" {
		if depth == 0 {
			fmt.Fprintf(output, "# %s\n\n", title)
		} else {
			fmt.Fprintf(output, "%s %s\n\n", strings.Repeat("#", depth+1), title)
		}
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
		for i := range val.Len() {
			elem := val.Index(i)
			fmt.Fprintf(output, "  • %v\n", formatFieldValue(elem))
		}
		output.WriteString("\n")
	}
}

// renderMap renders a map as markdown-style headers
func renderMap(val reflect.Value, title string, output *strings.Builder, depth int) {
	if val.Len() == 0 {
		return
	}

	// Print title without FormatInfoMessage styling
	if title != "" {
		if depth == 0 {
			fmt.Fprintf(output, "# %s\n\n", title)
		} else {
			fmt.Fprintf(output, "%s %s\n\n", strings.Repeat("#", depth+1), title)
		}
	}

	// Render map entries
	for _, key := range val.MapKeys() {
		mapValue := val.MapIndex(key)
		fmt.Fprintf(output, "  %-18s %v\n", fmt.Sprintf("%v:", key), formatFieldValue(mapValue))
	}
	output.WriteString("\n")
}

// buildTableConfig builds a TableConfig from a slice of structs
func buildTableConfig(val reflect.Value, title string) TableConfig {
	renderLog.Printf("Building table config: title=%s, elements=%d", title, val.Len())

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
	headers, fieldIndices, fieldTags := buildTableHeaders(elemType)
	config.Headers = headers

	// Build rows
	config.Rows = buildTableRows(val, fieldIndices, fieldTags)

	return config
}

// buildTableHeaders extracts column headers, field indices, and tags from a struct type.
func buildTableHeaders(elemType reflect.Type) (headers []string, fieldIndices []int, fieldTags []consoleTag) {
	for i := range elemType.NumField() {
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
		fieldTags = append(fieldTags, tag)
	}
	return headers, fieldIndices, fieldTags
}

// buildTableRows builds the row data for a slice of struct elements.
func buildTableRows(val reflect.Value, fieldIndices []int, fieldTags []consoleTag) [][]string {
	var rows [][]string
	for i := range val.Len() {
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
		for j, fieldIdx := range fieldIndices {
			field := elem.Field(fieldIdx)
			row = append(row, formatFieldValueWithTag(field, fieldTags[j]))
		}
		rows = append(rows, row)
	}
	return rows
}

// consoleTag represents parsed console struct tag
type consoleTag struct {
	title      string
	header     string
	format     string
	defaultVal string // Default value for zero/empty values
	maxLen     int    // Maximum length for string truncation
	omitempty  bool
	skip       bool
}

// parseConsoleTag parses the console struct tag
func parseConsoleTag(tag string) consoleTag {
	result := consoleTag{}

	if tag == "-" {
		result.skip = true
		return result
	}

	parts := strings.SplitSeq(tag, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "omitempty" {
			result.omitempty = true
		} else if after, ok := strings.CutPrefix(part, "title:"); ok {
			result.title = after
		} else if after, ok := strings.CutPrefix(part, "header:"); ok {
			result.header = after
		} else if after, ok := strings.CutPrefix(part, "format:"); ok {
			result.format = after
		} else if after, ok := strings.CutPrefix(part, "default:"); ok {
			result.defaultVal = after
		} else if after, ok := strings.CutPrefix(part, "maxlen:"); ok {
			maxLenStr := after
			if len, err := strconv.Atoi(maxLenStr); err == nil {
				result.maxLen = len
			}
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
		if val.Kind() == reflect.String {
			return "-"
		}
		// For numeric types, return the actual value
		if val.Kind() >= reflect.Int && val.Kind() <= reflect.Float64 {
			if val.CanInterface() {
				return fmt.Sprintf("%v", val.Interface())
			}
			return formatNumericKind(val)
		}
		return "-"
	}

	// Special handling for time.Time to avoid unexported field panic
	if val.Type().String() == "time.Time" {
		return formatTimeValue(val)
	}

	// Only call Interface() if we can
	if !val.CanInterface() {
		return formatUnexportedValue(val)
	}

	return fmt.Sprintf("%v", val.Interface())
}

// formatTimeValue formats a time.Time reflect value as a display string.
func formatTimeValue(val reflect.Value) string {
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
	return val.Type().String()
}

// formatUnexportedValue formats unexported struct fields by kind without using Interface().
func formatUnexportedValue(val reflect.Value) string {
	switch val.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", val.Float())
	case reflect.String:
		return val.String()
	default:
		return val.Type().String()
	}
}

// formatNumericKind formats numeric kinds without Interface() as a fallback.
func formatNumericKind(val reflect.Value) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", val.Float())
	default:
		return val.Type().String()
	}
}

// formatFieldValueWithTag formats a reflect.Value as a string for display with format tag support
func formatFieldValueWithTag(val reflect.Value, tag consoleTag) string {
	// Get the base formatted value
	baseValue := formatFieldValue(val)

	// Check if value is zero/empty and apply default if specified
	if tag.defaultVal != "" && isZeroValue(val) {
		baseValue = tag.defaultVal
	}

	// Apply format based on tag
	if tag.format != "" && baseValue != "-" {
		baseValue = applyTagFormat(val, tag.format, baseValue)
	}

	// Apply maxlen truncation if specified
	if tag.maxLen > 0 && len(baseValue) > tag.maxLen {
		if tag.maxLen > 3 {
			baseValue = baseValue[:tag.maxLen-3] + "..."
		} else {
			baseValue = baseValue[:tag.maxLen]
		}
	}

	return baseValue
}

// applyTagFormat applies a named format (number, cost, filesize) to baseValue.
func applyTagFormat(val reflect.Value, format, baseValue string) string {
	switch format {
	case "number":
		return applyNumberFormat(val, baseValue)
	case "cost":
		return applyCostFormat(val, baseValue)
	case "filesize":
		return applyFilesizeFormat(val, baseValue)
	}
	return baseValue
}

// applyNumberFormat formats a value as a human-readable number (e.g., "1k", "1.2M").
func applyNumberFormat(val reflect.Value, baseValue string) string {
	if val.CanInterface() {
		switch v := val.Interface().(type) {
		case int:
			return FormatNumber(v)
		case int64:
			// #nosec G115 - Converting int64 to int for display formatting
			return FormatNumber(int(v))
		case int32:
			return FormatNumber(int(v))
		case uint:
			// #nosec G115 - Converting uint to int for display formatting
			return FormatNumber(int(v))
		case uint64:
			// #nosec G115 - Converting uint64 to int for display formatting
			return FormatNumber(int(v))
		case uint32:
			return FormatNumber(int(v))
		}
	}
	// Fallback: use integer kind directly, keeping signed and unsigned separate
	// to avoid calling Int() on an unsigned kind (which panics).
	switch {
	case val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64:
		return FormatNumber(int(val.Int()))
	case val.Kind() >= reflect.Uint && val.Kind() <= reflect.Uint64:
		// #nosec G115 - Converting uint to int for display formatting
		return FormatNumber(int(val.Uint()))
	}
	return baseValue
}

// applyCostFormat formats a value as currency with $ prefix.
func applyCostFormat(val reflect.Value, baseValue string) string {
	if val.CanInterface() {
		switch v := val.Interface().(type) {
		case float64:
			if v > 0 {
				return fmt.Sprintf("$%.3f", v)
			}
		case float32:
			if v > 0 {
				return fmt.Sprintf("$%.3f", v)
			}
		}
	}
	if val.Kind() == reflect.Float64 || val.Kind() == reflect.Float32 {
		if val.Float() > 0 {
			return fmt.Sprintf("$%.3f", val.Float())
		}
	}
	return baseValue
}

// applyFilesizeFormat formats a value as a human-readable file size (e.g., "1.2 MB").
func applyFilesizeFormat(val reflect.Value, baseValue string) string {
	if val.CanInterface() {
		switch v := val.Interface().(type) {
		case int:
			return FormatFileSize(int64(v))
		case int64:
			return FormatFileSize(v)
		case int32:
			return FormatFileSize(int64(v))
		case uint:
			// #nosec G115 - Converting uint to int64 for file size display
			return FormatFileSize(int64(v))
		case uint64:
			// #nosec G115 - Converting uint64 to int64 for file size display
			return FormatFileSize(int64(v))
		case uint32:
			return FormatFileSize(int64(v))
		}
	}
	// Fallback for integer kinds
	if val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64 {
		return FormatFileSize(val.Int())
	}
	if val.Kind() >= reflect.Uint && val.Kind() <= reflect.Uint64 {
		// #nosec G115 - Converting uint to int64 for file size display
		return FormatFileSize(int64(val.Uint()))
	}
	return baseValue
}

// FormatNumber formats large numbers in a human-readable way (e.g., "1k", "1.2k", "1.12M")
func FormatNumber(n int) string {
	if n == 0 {
		return "0"
	}

	f := float64(n)

	if f < 1000 {
		return strconv.Itoa(n)
	} else if f < 1000000 {
		// Format as thousands (k)
		k := f / 1000
		if k >= 100 {
			return fmt.Sprintf("%.0fk", k)
		} else if k >= 10 {
			return fmt.Sprintf("%.1fk", k)
		} else {
			return fmt.Sprintf("%.2fk", k)
		}
	} else if f < 1000000000 {
		// Format as millions (M)
		m := f / 1000000
		if m >= 100 {
			return fmt.Sprintf("%.0fM", m)
		} else if m >= 10 {
			return fmt.Sprintf("%.1fM", m)
		} else {
			return fmt.Sprintf("%.2fM", m)
		}
	} else {
		// Format as billions (B)
		b := f / 1000000000
		if b >= 100 {
			return fmt.Sprintf("%.0fB", b)
		} else if b >= 10 {
			return fmt.Sprintf("%.1fB", b)
		} else {
			return fmt.Sprintf("%.2fB", b)
		}
	}
}

// ToRelativePath converts an absolute path to a relative path from the current working directory
// If the relative path contains "..", returns the absolute path instead for clarity
func ToRelativePath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}

	wd, err := os.Getwd()
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(wd, path)
	if err != nil {
		return path
	}

	if strings.Contains(relPath, "..") {
		return path
	}

	return relPath
}

// FormatErrorWithSuggestions formats an error message with actionable suggestions
func FormatErrorWithSuggestions(message string, suggestions []string) string {
	var output strings.Builder
	output.WriteString(FormatErrorMessage(message))

	if len(suggestions) > 0 {
		output.WriteString("\n\nSuggestions:\n")
		for _, suggestion := range suggestions {
			output.WriteString("  • " + suggestion + "\n")
		}
	}

	return output.String()
}

// findWordEnd finds the end of a word starting at the given position
func findWordEnd(line string, start int) int {
	if start >= len(line) {
		return len(line)
	}

	end := start
	for end < len(line) {
		char := line[end]
		if char == ' ' || char == '\t' || char == ':' || char == '\n' || char == '\r' {
			break
		}
		end++
	}

	return end
}
