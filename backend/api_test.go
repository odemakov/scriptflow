package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLogsForRun(t *testing.T) {
	tests := []struct {
		name         string
		logContent   string
		runId        string
		expectedLogs []string
	}{
		{
			name: "Common case - single runId with logs",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 1
Log line 2
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id
Other log line`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
				"Log line 1",
				"Log line 2",
			},
		},
		{
			name: "Edge case - no delimiter at all",
			logContent: `Log line 1
Log line 2
Log line 3`,
			runId:        "85egyv91mcmw0ug",
			expectedLogs: []string{},
		},
		{
			name: "Edge case - no log lines between delimiters",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
			},
		},
		{
			// theoretically, only one tash is write to the log file as the given time
			// all the tasks are singletones
			name: "Common case - multiple log lines for runId",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 1
Log line 2
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id
Other log line
[2024-11-21T18:01:00+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 3`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
				"Log line 1",
				"Log line 2",
			},
		},
		{
			name: "Edge case - runId not found",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run another_run_id
Other log line`,
			runId:        "85egyv91mcmw0ug",
			expectedLogs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary log file
			tmpFile, err := os.CreateTemp("", "logfile-*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write log content to the temporary file
			_, err = tmpFile.WriteString(tt.logContent)
			if err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Call the function
			logs, err := extractLogsForRun(tmpFile.Name(), tt.runId)
			if err != nil {
				t.Fatalf("Function returned an error: %v", err)
			}

			// Compare the result with the expected output
			if len(logs) != len(tt.expectedLogs) {
				t.Fatalf("Expected %d logs, got %d", len(tt.expectedLogs), len(logs))
			}
			for i, logLine := range logs {
				if logLine != tt.expectedLogs[i] {
					t.Errorf("Mismatch at line %d: expected %q, got %q", i, tt.expectedLogs[i], logLine)
				}
			}
		})
	}
}

func writeTempFile(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "logtest-*.log")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	_, err = f.Seek(0, 0)
	require.NoError(t, err)
	return f
}

func TestAppendWithRollingWindow(t *testing.T) {
	tests := []struct {
		name     string
		logs     []string
		line     string
		maxLines int
		want     []string
	}{
		{"below cap", []string{"a", "b"}, "c", 5, []string{"a", "b", "c"}},
		{"at cap drops oldest", []string{"a", "b", "c"}, "d", 3, []string{"b", "c", "d"}},
		{"empty slice", nil, "x", 3, []string{"x"}},
		{"maxLines=1 always one elem", []string{"a"}, "b", 1, []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendWithRollingWindow(tt.logs, tt.line, tt.maxLines)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadLastLines(t *testing.T) {
	content10 := strings.Join([]string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}, "\n")
	// Real log files always end with \n (formatLogLine appends \n)
	content10NL := content10 + "\n"

	tests := []struct {
		name    string
		content string
		n       int
		want    []string
	}{
		{"empty file", "", 5, nil},
		{"fewer lines than n", "a\nb\nc", 10, []string{"a", "b", "c"}},
		{"exactly n lines", "a\nb\nc", 3, []string{"a", "b", "c"}},
		{"last 5 of 10", content10, 5, []string{"L6", "L7", "L8", "L9", "L10"}},
		{"last 1 of 10", content10, 1, []string{"L10"}},
		{"n=0 returns empty", "a\nb", 0, nil},
		// trailing newline cases (real log file format)
		{"trailing newline: fewer than n", "a\nb\nc\n", 10, []string{"a", "b", "c"}},
		{"trailing newline: exactly n", "a\nb\nc\n", 3, []string{"a", "b", "c"}},
		{"trailing newline: last 5 of 10", content10NL, 5, []string{"L6", "L7", "L8", "L9", "L10"}},
		{"trailing newline: last 1 of 10", content10NL, 1, []string{"L10"}},
		{"trailing newline: all 10 of 10", content10NL, 10, []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}},
		{"trailing newline: first line not dropped", "a\nb\nc\n", 3, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := writeTempFile(t, tt.content)
			got, err := readLastLines(f, tt.n)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadLinesPage(t *testing.T) {
	// Both forms: without and with trailing newline (real log file format)
	content10 := strings.Join([]string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}, "\n")
	content10NL := content10 + "\n" // real log format

	tests := []struct {
		name        string
		content     string
		offset      int
		limit       int
		wantLines   []string
		wantHasMore bool
	}{
		{
			name:    "first page (offset=0)",
			content: content10,
			offset:  0, limit: 5,
			wantLines:   []string{"L6", "L7", "L8", "L9", "L10"},
			wantHasMore: true,
		},
		{
			name:    "second page (offset=5)",
			content: content10,
			offset:  5, limit: 5,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5"},
			wantHasMore: false,
		},
		{
			name:    "partial last page (offset=8)",
			content: content10,
			offset:  8, limit: 5,
			wantLines:   []string{"L1", "L2"},
			wantHasMore: false,
		},
		{
			name:    "offset past end returns nil",
			content: content10,
			offset:  11, limit: 5,
			wantLines:   nil,
			wantHasMore: false,
		},
		{
			name:    "limit larger than file",
			content: content10,
			offset:  0, limit: 100,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"},
			wantHasMore: false,
		},
		{
			name:    "exactly one page fits (offset=0, limit=10)",
			content: content10,
			offset:  0, limit: 10,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"},
			wantHasMore: false,
		},
		{
			name:    "hasMore true when file has 11 lines",
			content: strings.Join([]string{"L0", "L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}, "\n"),
			offset:  0, limit: 10,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"},
			wantHasMore: true,
		},
		{
			name:    "empty file returns nil",
			content: "",
			offset:  0, limit: 5,
			wantLines:   nil,
			wantHasMore: false,
		},
		// trailing-newline variants — real log format must behave identically
		{
			name:    "trailing newline: first page",
			content: content10NL,
			offset:  0, limit: 5,
			wantLines:   []string{"L6", "L7", "L8", "L9", "L10"},
			wantHasMore: true,
		},
		{
			name:    "trailing newline: second page",
			content: content10NL,
			offset:  5, limit: 5,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5"},
			wantHasMore: false,
		},
		{
			name:    "trailing newline: exactly one page",
			content: content10NL,
			offset:  0, limit: 10,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"},
			wantHasMore: false,
		},
		{
			name:    "trailing newline: hasMore true for 11 lines",
			content: strings.Join([]string{"L0", "L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}, "\n") + "\n",
			offset:  0, limit: 10,
			wantLines:   []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"},
			wantHasMore: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := writeTempFile(t, tt.content)
			gotLines, gotHasMore, err := readLinesPage(f, tt.offset, tt.limit)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLines, gotLines)
			assert.Equal(t, tt.wantHasMore, gotHasMore)
		})
	}
}

func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name       string
		v          string
		defaultVal int
		minVal     int
		maxVal     int
		want       int
	}{
		{"empty uses default", "", 100, 0, 0, 100},
		{"valid value", "42", 100, 0, 0, 42},
		{"below min uses default", "-1", 100, 0, 0, 100},
		{"equal to min", "0", 100, 0, 0, 0},
		{"above max uses default", "600", 100, 1, 500, 100},
		{"equal to max", "500", 100, 1, 500, 500},
		{"maxVal=0 means no upper bound", "9999", 100, 0, 0, 9999},
		{"non-numeric uses default", "abc", 100, 0, 0, 100},
		{"float uses default", "1.5", 100, 0, 0, 100},
		{"min=1 rejects zero", "0", 100, 1, 500, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQueryInt(tt.v, tt.defaultVal, tt.minVal, tt.maxVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// appendToFile appends `count` new sequentially-numbered lines to f (already open).
// Writes line + "\n" matching the formatLogLine convention so the file always ends with \n.
func appendToFile(t *testing.T, f *os.File, count int, lines *[]string) {
	t.Helper()
	_, err := f.Seek(0, io.SeekEnd)
	require.NoError(t, err)
	for i := 0; i < count; i++ {
		line := fmt.Sprintf("L%05d", len(*lines)+1)
		*lines = append(*lines, line)
		_, werr := f.WriteString(line + "\n")
		require.NoError(t, werr)
	}
}

// simulateScrollAll models the exact LogViewer.vue pagination contract:
//
//   - WS sends last wsPageSize lines via sendLastLines (does NOT increment liveLineOffset)
//   - Frontend initializes liveLineOffset = wsPageSize
//   - Each loadMoreLines call: GET /log?offset=liveLineOffset&limit=apiPageSize
//     then liveLineOffset += len(received)
//   - Scroll continues until has_more=false
//
// Returns all lines in file order (oldest first).
func simulateScrollAll(t *testing.T, file *os.File, wsPageSize, apiPageSize int) []string {
	t.Helper()

	wsLines, err := readLastLines(file, wsPageSize)
	require.NoError(t, err)

	// liveLineOffset is NOT incremented for the initial WS burst
	liveLineOffset := wsPageSize

	var olderLines []string
	for {
		page, hasMore, err := readLinesPage(file, liveLineOffset, apiPageSize)
		require.NoError(t, err)
		if len(page) > 0 {
			olderLines = append(page, olderLines...)
			liveLineOffset += len(page)
		}
		if !hasMore {
			break
		}
	}
	return append(olderLines, wsLines...)
}

// TestApiTaskLogLinesScrollInvariant verifies that for a static file the WS initial
// load plus all scroll pages cover every line exactly once with no gaps or duplicates.
func TestApiTaskLogLinesScrollInvariant(t *testing.T) {
	tests := []struct {
		name        string
		totalLines  int
		wsPageSize  int
		apiPageSize int
	}{
		// file smaller than WS page: WS covers everything, scroll returns nil
		{"file smaller than ws page", 50, 100, 100},
		// exactly one WS page
		{"file equals ws page", 100, 100, 100},
		// one line older than WS window
		{"one line beyond ws", 101, 100, 100},
		// WS page + one full API page
		{"ws + one api page", 200, 100, 100},
		// boundary: readLinesPage sentinel edge (offset+limit+1 == file size)
		{"ws + api page + 1", 201, 100, 100},
		// multiple full API pages
		{"many pages", 1000, 100, 100},
		// last page is partial
		{"partial last api page", 250, 100, 100},
		// different ws and api page sizes
		{"ws=50 api=100", 500, 50, 100},
		{"ws=100 api=50", 500, 100, 50},
		// single line file
		{"single line", 1, 100, 100},
		// two lines: one in WS, one older
		{"two lines", 2, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantLines := make([]string, tt.totalLines)
			for i := range wantLines {
				wantLines[i] = fmt.Sprintf("L%05d", i+1)
			}
			// Use trailing newline — real log format (formatLogLine always appends \n)
			f := writeTempFile(t, strings.Join(wantLines, "\n")+"\n")

			got := simulateScrollAll(t, f, tt.wsPageSize, tt.apiPageSize)
			assert.Equal(t, wantLines, got,
				"all %d lines must be covered exactly once (wsPage=%d apiPage=%d)",
				tt.totalLines, tt.wsPageSize, tt.apiPageSize)
		})
	}
}

// TestApiTaskLogLinesScrollWithLiveGrowth verifies the invariant holds when new lines
// arrive via WS (incrementing liveLineOffset) between scroll calls.
// The file grows after the initial burst; each growth batch simulates live streaming.
func TestApiTaskLogLinesScrollWithLiveGrowth(t *testing.T) {
	tests := []struct {
		name         string
		initialLines int
		liveBatches  []int // lines added between scrolls (each scroll fetches one page)
		wsPageSize   int
		apiPageSize  int
	}{
		{
			name:         "50 live lines arrive before first scroll",
			initialLines: 300, liveBatches: []int{50},
			wsPageSize: 100, apiPageSize: 100,
		},
		{
			name:         "live lines between each scroll",
			initialLines: 400, liveBatches: []int{10, 10, 10},
			wsPageSize: 100, apiPageSize: 100,
		},
		{
			name:         "no live lines (static baseline)",
			initialLines: 250, liveBatches: nil,
			wsPageSize: 100, apiPageSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build initial file content with trailing newline (real log format)
			allWantLines := make([]string, tt.initialLines)
			for i := range allWantLines {
				allWantLines[i] = fmt.Sprintf("L%05d", i+1)
			}
			f := writeTempFile(t, strings.Join(allWantLines, "\n")+"\n")

			// WS initial burst (does NOT increment liveLineOffset)
			wsLines, err := readLastLines(f, tt.wsPageSize)
			require.NoError(t, err)
			liveLineOffset := tt.wsPageSize

			batchIdx := 0

			// Simulate live lines arriving before first scroll
			if batchIdx < len(tt.liveBatches) {
				count := tt.liveBatches[batchIdx]
				batchIdx++
				appendToFile(t, f, count, &allWantLines)
				// WS delivers these live lines; liveLineOffset increments
				liveLineOffset += count
				// Append them to wsLines (live lines land at bottom of viewer)
				wsLines = append(wsLines, allWantLines[len(allWantLines)-count:]...)
			}

			var olderLines []string
			for {
				page, hasMore, err := readLinesPage(f, liveLineOffset, tt.apiPageSize)
				require.NoError(t, err)
				if len(page) > 0 {
					olderLines = append(page, olderLines...)
					liveLineOffset += len(page)
				}
				if !hasMore {
					break
				}
				// Simulate more live lines arriving between scrolls
				if batchIdx < len(tt.liveBatches) {
					count := tt.liveBatches[batchIdx]
					batchIdx++
					appendToFile(t, f, count, &allWantLines)
					liveLineOffset += count
					wsLines = append(wsLines, allWantLines[len(allWantLines)-count:]...)
				}
			}

			got := append(olderLines, wsLines...)
			assert.Equal(t, allWantLines, got,
				"all %d lines must be covered exactly once", len(allWantLines))
		})
	}
}

// TestApiTaskLogLinesHasMoreBoundary checks the exact hasMore boundary conditions
// that determine whether the "load older" indicator shows in the UI.
func TestApiTaskLogLinesHasMoreBoundary(t *testing.T) {
	// file with N lines, wsPageSize=100: first scroll at offset=100
	tests := []struct {
		totalLines  int
		wantHasMore bool
	}{
		{100, false}, // all in WS window, scroll returns nil
		{200, false}, // exactly ws+api page: scroll returns [L1..L100], no more
		{201, true},  // one extra: first scroll has more (returns only 100 of 101 older)
		{300, true},  // 200 older lines: first scroll returns 100, still has more
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("total=%d", tt.totalLines), func(t *testing.T) {
			lines := make([]string, tt.totalLines)
			for i := range lines {
				lines[i] = fmt.Sprintf("L%05d", i+1)
			}
			f := writeTempFile(t, strings.Join(lines, "\n"))
			_, hasMore, err := readLinesPage(f, 100, 100)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHasMore, hasMore,
				"hasMore for %d-line file at offset=100 limit=100", tt.totalLines)
		})
	}
}
