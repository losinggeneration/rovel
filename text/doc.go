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
// All functions operate on UTF-8 encoded strings. Display width is currently a
// simplified cell-width policy shared with the renderer (CJK is treated as
// wide; emoji/combining behavior is intentionally conservative).
package text
