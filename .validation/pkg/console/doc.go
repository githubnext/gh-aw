// Package console provides terminal UI components and formatting utilities for
// the gh-aw CLI.
//
// # Naming Convention: Format* vs Render*
//
// Functions in this package follow a consistent naming convention:
//
//   - Format* functions return a formatted string for a single item or message.
//     They are pure string transformations with no side effects.
//     Examples: FormatSuccessMessage, FormatErrorMessage, FormatFileSize,
//     FormatCommandMessage, FormatProgressMessage.
//
//   - Render* functions produce multi-element or structured output (tables, boxes,
//     trees, structs). They may return strings, slices of strings, or write
//     directly to output. They are used when the output requires layout or
//     structural composition.
//     Examples: RenderTable, RenderStruct, RenderTitleBox, RenderErrorBox,
//     RenderInfoSection, RenderTree, RenderComposedSections.
//
// # Output Routing
//
// All diagnostic output (messages, warnings, errors) should be written to stderr.
// Structured data output (JSON, hashes, graphs) should be written to stdout.
// Use fmt.Fprintln(os.Stderr, ...) with the Format* helpers for diagnostic output.
package console
