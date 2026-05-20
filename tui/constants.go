package tui

import "time"

// Layout constants — column widths used when building list rows.
// Keep these in sync with the corresponding Style* widths in styles.go.
const (
	colSizeWidth    = 9  // matches StyleSize.Width(9)
	colDateWidth    = 18 // StyleDate.Width(17) + 1-char separator
	colPubMarkWidth = 2  // "⇡ " or "  "
	colRowPadding   = 4  // PaddingLeft(1) + mark glyph (2) + inner gap (1)
	colNameMinWidth = 10 // minimum usable name column width

	trashOriginWidth = 20 // truncated "deleted from" path column in trash view

	dialogMargin   = 8  // horizontal space consumed by dialog border + outer padding
	dialogMinWidth = 40 // minimum dialog width to remain readable
)

// API timeout constants.
const (
	timeoutMeta     = 15 * time.Second // metadata / directory listing
	timeoutOp       = 15 * time.Second // short mutations (mkdir, rename, publish…)
	timeoutDelete   = 30 * time.Second // delete (may involve server-side move to trash)
	timeoutTransfer = 30 * time.Minute // upload / download
)

// Status icons used in user-facing messages.
const (
	iconOK  = "✓"
	iconErr = "✗"
)
