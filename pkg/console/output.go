package console

import (
	"encoding/json"
	"os"
)

// OutputStructOrJSON outputs a Go struct as either formatted console output or JSON
// based on the asJSON flag. This provides a unified interface for commands that
// support both console and JSON output modes.
//
// When asJSON is true, the struct is marshaled to JSON with indentation and written to stdout.
// When asJSON is false, the struct is rendered using RenderStruct and written to stdout.
//
// Example usage:
//
//	type Report struct {
//	    Title string `json:"title" console:"header:Title"`
//	    Count int    `json:"count" console:"header:Count"`
//	}
//
//	report := Report{Title: "Summary", Count: 42}
//	err := console.OutputStructOrJSON(report, jsonOutput)
func OutputStructOrJSON(v interface{}, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(v)
	}

	// For console output, use RenderStruct
	output := RenderStruct(v)
	_, err := os.Stdout.WriteString(output)
	return err
}
