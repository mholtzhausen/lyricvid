package lyrics

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Line represents a single lyric line with timing information.
type Line struct {
	StartTime float64 // Start time in seconds
	EndTime   float64 // End time in seconds (set during post-processing)
	Text      string
}

// ParseFile reads a lyrics file and returns parsed lines.
// It auto-detects LRC format (timestamped) vs plain text.
func ParseFile(path string) ([]Line, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, fmt.Errorf("opening lyrics file: %w", err)
	}
	defer f.Close()

	var rawLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			rawLines = append(rawLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("reading lyrics file: %w", err)
	}

	if len(rawLines) == 0 {
		return nil, false, fmt.Errorf("lyrics file is empty")
	}

	// Detect LRC format by checking if any line starts with a timestamp
	if isLRC(rawLines) {
		lines, err := parseLRC(rawLines)
		return lines, true, err
	}
	return parsePlainText(rawLines), false, nil
}

// SetEndTimes fills in EndTime for each line based on the next line's start or total duration.
// Empty-text sentinel lines (e.g. a bare LRC timestamp) act as an end marker and are removed
// from the returned slice.
func SetEndTimes(lines []Line, totalDuration float64) []Line {
	for i := range lines {
		if i < len(lines)-1 {
			lines[i].EndTime = lines[i+1].StartTime
		} else {
			lines[i].EndTime = totalDuration
		}
	}
	// Filter out empty-text sentinels now that they've served their purpose.
	out := lines[:0]
	for _, l := range lines {
		if l.Text != "" {
			out = append(out, l)
		}
	}
	return out
}

// DistributeEvenly assigns timestamps to plain text lines spread across the total duration.
func DistributeEvenly(lines []Line, totalDuration float64) {
	if len(lines) == 0 {
		return
	}
	perLine := totalDuration / float64(len(lines))
	for i := range lines {
		lines[i].StartTime = float64(i) * perLine
		lines[i].EndTime = float64(i+1) * perLine
	}
}

// lrcPattern matches LRC timestamp lines: [mm:ss.xx], [mm:ss.xxx], or [mm:ss:xx] (colon variant)
var lrcPattern = regexp.MustCompile(`^\[(\d{1,3}):(\d{2})[.:](\d{2,3})\]\s*(.*)$`)

func isLRC(lines []string) bool {
	lrcCount := 0
	for _, line := range lines {
		if lrcPattern.MatchString(line) {
			lrcCount++
		}
	}
	// Consider it LRC if at least half the non-empty lines match
	return lrcCount > 0 && lrcCount >= len(lines)/2
}

func parseLRC(rawLines []string) ([]Line, error) {
	var lines []Line
	for _, raw := range rawLines {
		matches := lrcPattern.FindStringSubmatch(raw)
		if matches == nil {
			// Skip metadata lines like [ti:Title], [ar:Artist], etc.
			if strings.HasPrefix(raw, "[") && strings.Contains(raw, ":") {
				continue
			}
			continue
		}

		minutes, _ := strconv.Atoi(matches[1])
		seconds, _ := strconv.Atoi(matches[2])
		fracStr := matches[3]
		frac, _ := strconv.Atoi(fracStr)

		var fracSeconds float64
		if len(fracStr) == 2 {
			fracSeconds = float64(frac) / 100.0
		} else {
			fracSeconds = float64(frac) / 1000.0
		}

		timestamp := float64(minutes)*60.0 + float64(seconds) + fracSeconds
		text := strings.TrimSpace(matches[4])

		lines = append(lines, Line{
			StartTime: timestamp,
			Text:      text,
		})
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("no valid lyric lines found in LRC file")
	}

	return lines, nil
}

func parsePlainText(rawLines []string) []Line {
	lines := make([]Line, len(rawLines))
	for i, text := range rawLines {
		lines[i] = Line{Text: text}
	}
	return lines
}
