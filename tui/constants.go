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

	// setupWrap* — text wrap bounds used in the OAuth setup screen.
	setupWrapMargin = 8  // subtracted from terminal width before clamping
	setupWrapMin    = 40 // minimum wrap width
	setupWrapMax    = 72 // maximum wrap width
)

// API timeout constants.
const (
	timeoutMeta     = 15 * time.Second // metadata / directory listing
	timeoutOp       = 15 * time.Second // short mutations (mkdir, rename, publish…)
	timeoutDelete   = 30 * time.Second // delete (may involve server-side move to trash)
	timeoutTransfer = 30 * time.Minute // upload / download
)

// Asynchronous server-side operations (currently only upload-from-URL) are
// polled rather than waited on, since the API exposes no push notification.
const (
	operationPollInterval = 1 * time.Second

	// operationSuccess is the terminal status the API reports for an operation
	// that completed. The package only exports the in-progress one.
	operationSuccess = "success"
)

// Status icons used in user-facing messages.
const (
	iconOK  = "✓"
	iconErr = "✗"
)
