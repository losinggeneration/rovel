// Package text provides shared text primitives for measurement, truncation,
// wrapping, and grapheme-aware navigation.
//
// This package establishes a single source of truth for display-width
// measurement and text navigation across all widgets.
//
// The toolkit uses these primitives to ensure consistent behavior between:
//   - Widget sizing and layout
//   - Text input cursor movement
//   - Text truncation and wrapping
//   - Content alignment
//
// All functions operate on UTF-8 encoded strings and handle wide characters
// (CJK, emoji) correctly.
package text
