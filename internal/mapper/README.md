# JSON Schema Error Mapper

The `internal/mapper` package provides functionality to map JSON Schema validation errors to precise YAML text spans using RFC6901 JSON Pointers and goccy/go-yaml AST position metadata.

## Overview

When JSON Schema validation fails, it typically provides:
- An `instancePath` (RFC6901 JSON Pointer) indicating where the error occurred
- An error type/kind (e.g., "type", "required", "additionalProperties")  
- Additional metadata about the error

This package translates these errors into precise line/column ranges in the original YAML source file, enabling IDEs and tools to highlight the exact problematic tokens.

## API

### Core Function

```go
func MapErrorToSpans(yamlBytes []byte, instancePath string, meta ErrorMeta) ([]Span, error)
```

Maps a JSON Schema validation error to one or more candidate YAML spans, ordered by confidence.

**Parameters:**
- `yamlBytes`: The original YAML content as bytes
- `instancePath`: RFC6901 JSON Pointer (e.g., "/jobs/build/steps/0/uses")
- `meta`: Error metadata including kind and property information

**Returns:**
- `[]Span`: Ordered list of candidate spans (highest confidence first)
- `error`: Parse error if YAML is invalid

### Types

```go
// Span describes a location in the source YAML file
type Span struct {
    StartLine  int     // 1-based line number
    StartCol   int     // 1-based column number  
    EndLine    int     // 1-based end line
    EndCol     int     // 1-based end column
    Confidence float64 // 0.0 - 1.0 confidence score
    Reason     string  // Human-readable explanation
}

// ErrorMeta contains validator-provided metadata
type ErrorMeta struct {
    Kind          string // Error type: "type", "required", "additionalProperties", etc.
    Property      string // Property name for property-specific errors
    SchemaSnippet string // Optional schema fragment for context
}
```

## Algorithm

The mapper uses a multi-step approach:

### 1. JSON Pointer Decoding
- Decodes RFC6901 pointers into path segments
- Handles escaping (`~0` → `~`, `~1` → `/`)
- Validates pointer format

### 2. AST Traversal
- Parses YAML using goccy/go-yaml with position metadata
- Traverses AST following pointer segments:
  - **Mapping nodes**: Match child keys to segment names
  - **Sequence nodes**: Use numeric indices
- Returns target node, parent, and key node

### 3. Error-Kind Mapping

#### Type Errors (`"type"`)
- **Direct hit**: Highlight the value token with high confidence (0.95)
- **Fallback**: Highlight entire node with good confidence (0.9)

#### Missing Required Properties (`"required"`)
- Find parent mapping that should contain the property
- Compute insertion anchor (position after last sibling)
- Confidence: 0.7-0.75 depending on context

#### Additional Properties (`"additionalProperties"`)
- Find the specific key node using `meta.Property`
- Highlight the key token (not the value)
- Confidence: 0.98 for exact key matches

#### Other Error Types
- Generic node highlighting with moderate confidence (0.8)

### 4. Fallback Heuristics

When exact traversal fails:

1. **Text Search**: Search for property names in raw YAML text (confidence: 0.6)
2. **Parent Context**: Find nearest existing parent node (confidence: 0.4)
3. **Document Fallback**: Return document-level span (confidence: 0.2)

## Confidence Scoring

Confidence scores guide tooling on which spans to prefer:

- **0.9-1.0**: Exact matches, high certainty
- **0.7-0.9**: Good matches with minor ambiguity  
- **0.4-0.7**: Reasonable guesses, contextual matches
- **0.2-0.4**: Fallback heuristics
- **0.0-0.2**: Last resort, document-level

## Usage Examples

### Type Mismatch Error

```go
yaml := `config:
  port: "8080"  # Should be integer
  host: "localhost"`

spans, err := MapErrorToSpans([]byte(yaml), "/config/port", ErrorMeta{
    Kind: "type",
})
// Returns: Span{StartLine: 2, StartCol: 9, EndLine: 2, EndCol: 14, Confidence: 0.95}
// Highlights the value "8080"
```

### Missing Required Property

```go
yaml := `config:
  host: "localhost"`

spans, err := MapErrorToSpans([]byte(yaml), "/config/port", ErrorMeta{
    Kind: "required",
    Property: "port",
})
// Returns: Span{StartLine: 3, StartCol: 18, ...} with insertion anchor
```

### Additional Property Error

```go
yaml := `config:
  port: 8080
  extra: "not allowed"`

spans, err := MapErrorToSpans([]byte(yaml), "/config/extra", ErrorMeta{
    Kind: "additionalProperties", 
    Property: "extra",
})
// Returns: Span highlighting the key "extra" with high confidence
```

## Supported YAML Features

- **Block style** mappings and sequences
- **Flow style** (`{key: value}`, `[item1, item2]`)
- **Mixed styles** within the same document
- **Nested structures** (arbitrarily deep)
- **Special characters** in keys
- **Numeric keys** 
- **Empty string keys**

## Limitations

- **Multi-document YAML**: Only processes first document
- **Anchors/Aliases**: Basic support, may return multiple candidates
- **Complex merge keys**: Limited support for YAML merge semantics
- **Position accuracy**: Depends on goccy/go-yaml token positions

## Error Handling

- **Invalid YAML**: Returns parse error
- **Invalid JSON Pointer**: Returns validation error  
- **Missing paths**: Returns fallback spans with low confidence
- **Empty input**: Returns document-level fallback span

## Testing

The package includes comprehensive tests covering:

- All supported error kinds
- Complex YAML structures (nested, flow style, mixed)
- Edge cases (invalid YAML, special characters, large indices)
- Confidence scoring validation
- Position accuracy verification

Run tests with:
```bash
go test ./internal/mapper -v
```

## Performance Considerations

- **Parse cost**: Full YAML parse required for each mapping
- **Memory usage**: AST held in memory during traversal
- **Caching**: Consider caching parsed AST for multiple error mappings

For high-frequency usage, parse YAML once and reuse the AST for multiple error mappings.