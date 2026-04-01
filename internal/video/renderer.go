package video

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/user/lyricvid/internal/lyrics"
)

// Config holds all parameters for video generation.
type Config struct {
	AudioPath      string
	ImagePath      string
	OutputPath     string
	Width          int
	Height         int
	FontSize       int
	FontColor      string
	HighlightColor string
	BgDim          float64
	Lines          []lyrics.Line
	Duration       float64
}

// Render builds and executes the FFmpeg command to produce the video.
func Render(cfg Config) error {
	fontPath := findFont()
	filterComplex := buildFilterComplex(cfg, fontPath)

	args := []string{
		"-y",
		"-loop", "1", "-i", cfg.ImagePath,
		"-i", cfg.AudioPath,
		"-filter_complex", filterComplex,
		"-map", "[out]", "-map", "1:a",
		"-c:v", "libx264", "-preset", "medium", "-crf", "23",
		"-c:a", "aac", "-b:a", "192k",
		"-shortest",
		cfg.OutputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// FFmpeg writes progress to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	fmt.Printf("Rendering video to %s ...\n", cfg.OutputPath)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ffmpeg: %w", err)
	}

	// Parse progress from stderr
	stderrContent := monitorProgress(stderr, cfg.Duration)

	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr) // newline after progress bar
		fmt.Fprintf(os.Stderr, "\nFFmpeg error output:\n%s\n", stderrContent)
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}

	fmt.Printf("\rProgress: [##################################################] 100%%  \n")
	fmt.Printf("Video saved to %s\n", cfg.OutputPath)
	return nil
}

var timePattern = regexp.MustCompile(`time=(\d+):(\d+):(\d+)\.(\d+)`)

func monitorProgress(r io.Reader, totalDuration float64) string {
	var sb strings.Builder
	scanner := bufio.NewScanner(r)
	scanner.Split(scanFFmpegLines)

	for scanner.Scan() {
		line := scanner.Text()
		sb.WriteString(line)
		sb.WriteString("\n")

		matches := timePattern.FindStringSubmatch(line)
		if matches != nil && totalDuration > 0 {
			hours, _ := strconv.ParseFloat(matches[1], 64)
			minutes, _ := strconv.ParseFloat(matches[2], 64)
			seconds, _ := strconv.ParseFloat(matches[3], 64)
			frac, _ := strconv.ParseFloat("0."+matches[4], 64)

			currentTime := hours*3600 + minutes*60 + seconds + frac
			pct := currentTime / totalDuration
			if pct > 1.0 {
				pct = 1.0
			}

			barWidth := 50
			filled := int(pct * float64(barWidth))
			bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
			fmt.Printf("\rProgress: [%s] %3.0f%%", bar, pct*100)
		}
	}

	return sb.String()
}

// scanFFmpegLines splits on \r or \n since FFmpeg uses \r for progress updates.
func scanFFmpegLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' || data[i] == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func buildFilterComplex(cfg Config, fontPath string) string {
	w := cfg.Width
	h := cfg.Height
	dim := 1.0 - cfg.BgDim

	// Base video chain: scale, pad, format, dim
	filter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,"+
			"format=yuv420p,"+
			"colorchannelmixer=rr=%.2f:gg=%.2f:bb=%.2f[bg];[bg]",
		w, h, w, h, dim, dim, dim,
	)

	highlightSize := int(float64(cfg.FontSize) * 1.2)
	contextSize := cfg.FontSize
	lineHeight := int(float64(highlightSize) * 1.6)

	// Build drawtext filters for each lyric line with context
	var drawTexts []string

	for i, line := range cfg.Lines {
		// Active/highlight line (centered)
		dt := buildDrawtext(fontPath, highlightSize, cfg.HighlightColor, "1.0",
			"(w-text_w)/2", fmt.Sprintf("(h/2)-(%d/2)", highlightSize),
			line.Text, line.StartTime, line.EndTime)
		drawTexts = append(drawTexts, dt)

		// Context lines: up to 2 before and 2 after
		for offset := -2; offset <= 2; offset++ {
			if offset == 0 {
				continue
			}
			ctxIdx := i + offset
			if ctxIdx < 0 || ctxIdx >= len(cfg.Lines) {
				continue
			}

			yExpr := fmt.Sprintf("(h/2)-(%d/2)+(%d)", highlightSize, offset*lineHeight)
			opacity := "0.7"
			if offset == -2 || offset == 2 {
				opacity = "0.4"
			}

			dt := buildDrawtext(fontPath, contextSize, cfg.FontColor, opacity,
				"(w-text_w)/2", yExpr,
				cfg.Lines[ctxIdx].Text, line.StartTime, line.EndTime)
			drawTexts = append(drawTexts, dt)
		}
	}

	filter += "\n" + strings.Join(drawTexts, ",\n")
	filter += "\n[out]"

	return filter
}

func buildDrawtext(fontPath string, fontSize int, color, opacity, x, y, text string, start, end float64) string {
	escapedText := escapeDrawtext(text)

	fontSpec := ""
	if fontPath != "" {
		fontSpec = fmt.Sprintf("fontfile='%s':", escapeDrawtextValue(fontPath))
	}

	// Handle color with opacity
	colorSpec := color
	if opacity != "1.0" && opacity != "1" {
		colorSpec = fmt.Sprintf("%s@%s", color, opacity)
	}

	return fmt.Sprintf(
		"drawtext=%sfontsize=%d:fontcolor=%s:x=%s:y=%s:text='%s':enable='between(t,%.3f,%.3f)'",
		fontSpec, fontSize, colorSpec, x, y, escapedText, start, end,
	)
}

// escapeDrawtext escapes text for use in FFmpeg drawtext filter.
// Characters that need escaping: \ : ' % and also [ ]
func escapeDrawtext(text string) string {
	// Order matters: escape backslash first
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, "'", "\u2019") // Replace apostrophe with unicode right single quote
	text = strings.ReplaceAll(text, ":", `\:`)
	text = strings.ReplaceAll(text, "%", `%%`)
	text = strings.ReplaceAll(text, "[", `\[`)
	text = strings.ReplaceAll(text, "]", `\]`)
	text = strings.ReplaceAll(text, ";", `\;`)
	return text
}

func escapeDrawtextValue(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, "'", `\'`)
	text = strings.ReplaceAll(text, ":", `\:`)
	return text
}

func findFont() string {
	var candidates []string

	switch runtime.GOOS {
	case "linux":
		candidates = []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
			"/usr/share/fonts/TTF/DejaVuSans-Bold.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
			"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
		}
	case "darwin":
		candidates = []string{
			"/Library/Fonts/Arial Bold.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
			"/System/Library/Fonts/SFNSDisplay.ttf",
		}
	case "windows":
		candidates = []string{
			`C:\Windows\Fonts\arialbd.ttf`,
			`C:\Windows\Fonts\arial.ttf`,
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "" // FFmpeg will use its default font
}
