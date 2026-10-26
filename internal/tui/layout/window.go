package layout

const (
	// HeaderRowsDefault is the vertical row offset above the first list item when search/filter is inactive:
	// - Row 0: App tabs header bar
	// - Row 1: Left pane rounded top border (╭──────────)
	// - Row 2: Table/tree column header title
	// - Row 3: Blank line (\n\n) after header for visual breathing room
	//
	// List items begin at Row 4.
	HeaderRowsDefault = 4

	// HeaderRowsWithSearch is the vertical row offset above the first list item when search/filter is active:
	// - Row 0: App tabs header bar
	// - Row 1: Left pane rounded top border (╭──────────)
	// - Row 2: Search input line (/ ...)
	// - Row 3: Blank line (\n\n) after search bar
	// - Row 4: Table/tree column header title
	// - Row 5: Blank line (\n\n) after header for visual breathing room
	//
	// List items begin at Row 6.
	HeaderRowsWithSearch = 6

	// ChromeRowsDefault is the total number of non-item vertical lines inside the pane:
	// 1 top border + 1 header title + 1 blank line + 1 bottom border = 4.
	ChromeRowsDefault = 4

	// ChromeRowsWithSearch is the total number of non-item vertical lines when search is active:
	// 4 (ChromeRowsDefault) + 2 (search input + blank line) = 6.
	ChromeRowsWithSearch = 6
)

// ListWindow calculates the visible slice range [startIdx, endIdx) and maxVisible items
// for a vertically scrollable list with a moving cursor.
//
// It guarantees:
//   - 0 <= startIdx <= cursorIndex < endIdx <= totalItems (when totalItems > 0).
//   - maxVisible >= 1 even in cramped terminal windows to avoid zero-division or out-of-bounds slicing.
func ListWindow(height, cursorIndex, totalItems int, hasFilter bool) (startIdx, endIdx, maxVisible int) {
	chrome := ChromeRowsDefault
	if hasFilter {
		chrome = ChromeRowsWithSearch
	}

	maxVisible = max(height - chrome, 1)

	if totalItems <= 0 {
		return 0, 0, maxVisible
	}

	if cursorIndex >= maxVisible {
		startIdx = cursorIndex - maxVisible + 1
	}

	endIdx = min(startIdx + maxVisible, totalItems)

	return startIdx, endIdx, maxVisible
}

// ListClickTarget maps a terminal mouse click Y-coordinate to the corresponding item index
// in the visible scroll window.
//
// It returns (targetIdx, true) if the click landed on an actual visible item row.
// It returns (-1, false) if the click landed on borders, headers, blank padding lines, or beyond items.
func ListClickTarget(mouseY, height, cursorIndex, totalItems int, hasFilter bool) (int, bool) {
	if totalItems <= 0 {
		return -1, false
	}

	headerOffset := HeaderRowsDefault
	if hasFilter {
		headerOffset = HeaderRowsWithSearch
	}

	clickedRow := mouseY - headerOffset
	if clickedRow < 0 {
		return -1, false
	}

	startIdx, endIdx, _ := ListWindow(height, cursorIndex, totalItems, hasFilter)
	targetIdx := startIdx + clickedRow
	if targetIdx >= startIdx && targetIdx < endIdx && targetIdx < totalItems {
		return targetIdx, true
	}

	return -1, false
}
