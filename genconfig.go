package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// GenerateConfig mirrors all generate/video flags for YAML config file support.
// Fields use omitempty so that zero/empty values are treated as "not configured".
type GenerateConfig struct {
	Lyrics              string  `yaml:"lyrics,omitempty"`
	Image               string  `yaml:"image,omitempty"`
	Output              string  `yaml:"output,omitempty"`
	Quality             string  `yaml:"quality,omitempty"`
	AspectRatio         string  `yaml:"aspect-ratio,omitempty"`
	Width               int     `yaml:"width,omitempty"`
	Height              int     `yaml:"height,omitempty"`
	FontSize            int     `yaml:"font-size,omitempty"`
	FontSizeReference   int     `yaml:"font-size-reference,omitempty"`
	FontColor           string  `yaml:"font-color,omitempty"`
	HighlightColor      string  `yaml:"highlight-color,omitempty"`
	BgDim               float64 `yaml:"bg-dim,omitempty"`
	Transition          string  `yaml:"transition,omitempty"`
	TransitionDuration  float64 `yaml:"transition-duration,omitempty"`
	LyricPosition       float64 `yaml:"lyric-position,omitempty"`
	LyricFade           float64 `yaml:"lyric-fade,omitempty"`
	LyricFadeStyle      string  `yaml:"lyric-fade-style,omitempty"`
	FadeInSeconds       float64 `yaml:"fade-in-seconds,omitempty"`
	FadeInTitle         string  `yaml:"fade-in-title,omitempty"`
	FadeInTitleFadeOut  float64 `yaml:"fade-in-title-fade-out,omitempty"`
	FadeInFontSize      string  `yaml:"fade-in-font-size,omitempty"`
	FadeInFontColor     string  `yaml:"fade-in-font-color,omitempty"`
	FadeOutSeconds      float64 `yaml:"fade-out-seconds,omitempty"`
	FadeOutTitle        string  `yaml:"fade-out-title,omitempty"`
	FadeOutTitleFadeOut float64 `yaml:"fade-out-title-fade-out,omitempty"`
	FadeOutFontSize     string  `yaml:"fade-out-font-size,omitempty"`
	FadeOutFontColor    string  `yaml:"fade-out-font-color,omitempty"`
}

// loadGenerateConfig loads and merges one or more YAML config files in order.
// Files are applied lowest-to-highest priority (later files override earlier ones).
// Missing files are silently skipped; parse errors are returned.
// The second return value is the list of files that were actually loaded.
func loadGenerateConfig(paths []string) (GenerateConfig, []string, error) {
	var acc GenerateConfig
	var loaded []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return acc, loaded, fmt.Errorf("reading config %q: %w", p, err)
		}
		var tmp GenerateConfig
		if err := yaml.Unmarshal(data, &tmp); err != nil {
			return acc, loaded, fmt.Errorf("parsing config %q: %w", p, err)
		}
		loaded = append(loaded, p)
		// Merge: non-zero/non-empty values from tmp override acc.
		if tmp.Lyrics != "" {
			acc.Lyrics = tmp.Lyrics
		}
		if tmp.Image != "" {
			acc.Image = tmp.Image
		}
		if tmp.Output != "" {
			acc.Output = tmp.Output
		}
		if tmp.Quality != "" {
			acc.Quality = tmp.Quality
		}
		if tmp.AspectRatio != "" {
			acc.AspectRatio = tmp.AspectRatio
		}
		if tmp.Width != 0 {
			acc.Width = tmp.Width
		}
		if tmp.Height != 0 {
			acc.Height = tmp.Height
		}
		if tmp.FontSize != 0 {
			acc.FontSize = tmp.FontSize
		}
		if tmp.FontSizeReference != 0 {
			acc.FontSizeReference = tmp.FontSizeReference
		}
		if tmp.FontColor != "" {
			acc.FontColor = tmp.FontColor
		}
		if tmp.HighlightColor != "" {
			acc.HighlightColor = tmp.HighlightColor
		}
		if tmp.BgDim != 0 {
			acc.BgDim = tmp.BgDim
		}
		if tmp.Transition != "" {
			acc.Transition = tmp.Transition
		}
		if tmp.TransitionDuration != 0 {
			acc.TransitionDuration = tmp.TransitionDuration
		}
		if tmp.LyricPosition != 0 {
			acc.LyricPosition = tmp.LyricPosition
		}
		if tmp.LyricFade != 0 {
			acc.LyricFade = tmp.LyricFade
		}
		if tmp.LyricFadeStyle != "" {
			acc.LyricFadeStyle = tmp.LyricFadeStyle
		}
		if tmp.FadeInSeconds != 0 {
			acc.FadeInSeconds = tmp.FadeInSeconds
		}
		if tmp.FadeInTitle != "" {
			acc.FadeInTitle = tmp.FadeInTitle
		}
		if tmp.FadeInTitleFadeOut != 0 {
			acc.FadeInTitleFadeOut = tmp.FadeInTitleFadeOut
		}
		if tmp.FadeInFontSize != "" {
			acc.FadeInFontSize = tmp.FadeInFontSize
		}
		if tmp.FadeInFontColor != "" {
			acc.FadeInFontColor = tmp.FadeInFontColor
		}
		if tmp.FadeOutSeconds != 0 {
			acc.FadeOutSeconds = tmp.FadeOutSeconds
		}
		if tmp.FadeOutTitle != "" {
			acc.FadeOutTitle = tmp.FadeOutTitle
		}
		if tmp.FadeOutTitleFadeOut != 0 {
			acc.FadeOutTitleFadeOut = tmp.FadeOutTitleFadeOut
		}
		if tmp.FadeOutFontSize != "" {
			acc.FadeOutFontSize = tmp.FadeOutFontSize
		}
		if tmp.FadeOutFontColor != "" {
			acc.FadeOutFontColor = tmp.FadeOutFontColor
		}
	}
	return acc, loaded, nil
}

// applyConfig applies config file values to the package-level flag variables,
// but only for flags the user did not explicitly pass on the CLI.
func applyConfig(cmd *cobra.Command, gc GenerateConfig) {
	if gc.Lyrics != "" && !cmd.Flags().Changed("lyrics") {
		lyricsPath = gc.Lyrics
	}
	if gc.Image != "" && !cmd.Flags().Changed("image") {
		imagePath = gc.Image
	}
	if gc.Output != "" && !cmd.Flags().Changed("output") {
		outputPath = gc.Output
	}
	if gc.Quality != "" && !cmd.Flags().Changed("quality") {
		generateQuality = gc.Quality
	}
	if gc.AspectRatio != "" && !cmd.Flags().Changed("aspect-ratio") {
		aspectRatio = gc.AspectRatio
	}
	if gc.Width != 0 && !cmd.Flags().Changed("width") {
		width = gc.Width
	}
	if gc.Height != 0 && !cmd.Flags().Changed("height") {
		height = gc.Height
	}
	if gc.FontSize != 0 && !cmd.Flags().Changed("font-size") {
		fontSize = gc.FontSize
	}
	if gc.FontSizeReference != 0 && !cmd.Flags().Changed("font-size-reference") {
		fontSizeReference = gc.FontSizeReference
	}
	if gc.FontColor != "" && !cmd.Flags().Changed("font-color") {
		fontColor = gc.FontColor
	}
	if gc.HighlightColor != "" && !cmd.Flags().Changed("highlight-color") {
		highlightColor = gc.HighlightColor
	}
	if gc.BgDim != 0 && !cmd.Flags().Changed("bg-dim") {
		bgDim = gc.BgDim
	}
	if gc.Transition != "" && !cmd.Flags().Changed("transition") {
		transition = gc.Transition
	}
	if gc.TransitionDuration != 0 && !cmd.Flags().Changed("transition-duration") {
		transitionDuration = gc.TransitionDuration
	}
	if gc.LyricPosition != 0 && !cmd.Flags().Changed("lyric-position") {
		lyricVPosition = gc.LyricPosition
	}
	if gc.LyricFade != 0 && !cmd.Flags().Changed("lyric-fade") {
		lyricFade = gc.LyricFade
	}
	if gc.LyricFadeStyle != "" && !cmd.Flags().Changed("lyric-fade-style") {
		lyricFadeStyle = gc.LyricFadeStyle
	}
	if gc.FadeInSeconds != 0 && !cmd.Flags().Changed("fade-in-seconds") {
		fadeInSeconds = gc.FadeInSeconds
	}
	if gc.FadeInTitle != "" && !cmd.Flags().Changed("fade-in-title") {
		fadeInTitle = gc.FadeInTitle
	}
	if gc.FadeInTitleFadeOut != 0 && !cmd.Flags().Changed("fade-in-title-fade-out") {
		fadeInTitleFadeOut = gc.FadeInTitleFadeOut
	}
	if gc.FadeInFontSize != "" && !cmd.Flags().Changed("fade-in-font-size") {
		fadeInFontSize = gc.FadeInFontSize
	}
	if gc.FadeInFontColor != "" && !cmd.Flags().Changed("fade-in-font-color") {
		fadeInFontColor = gc.FadeInFontColor
	}
	if gc.FadeOutSeconds != 0 && !cmd.Flags().Changed("fade-out-seconds") {
		fadeOutSeconds = gc.FadeOutSeconds
	}
	if gc.FadeOutTitle != "" && !cmd.Flags().Changed("fade-out-title") {
		fadeOutTitle = gc.FadeOutTitle
	}
	if gc.FadeOutTitleFadeOut != 0 && !cmd.Flags().Changed("fade-out-title-fade-out") {
		fadeOutTitleFadeOut = gc.FadeOutTitleFadeOut
	}
	if gc.FadeOutFontSize != "" && !cmd.Flags().Changed("fade-out-font-size") {
		fadeOutFontSize = gc.FadeOutFontSize
	}
	if gc.FadeOutFontColor != "" && !cmd.Flags().Changed("fade-out-font-color") {
		fadeOutFontColor = gc.FadeOutFontColor
	}
}

// runCreateConfig writes a fully-commented YAML config template.
func runCreateConfig(_ *cobra.Command, args []string) error {
	outPath := filepath.Join(".", "lyricvid.yml")
	if len(args) > 0 {
		outPath = args[0]
	}

	var b strings.Builder

	b.WriteString("# lyricvid configuration file\n")
	b.WriteString("# Generated by: lyricvid create-config\n")
	b.WriteString("#\n")
	b.WriteString("# Place this file alongside your audio file as:\n")
	b.WriteString("#   lyricvid.yml        — applies to all audio files in that folder\n")
	b.WriteString("#   <audio-stem>.yml    — applies only to that specific audio file\n")
	b.WriteString("#\n")
	b.WriteString("# Config files are loaded in that order; later files override earlier ones.\n")
	b.WriteString("# CLI flags always take the highest priority and override everything.\n")
	b.WriteString("#\n")
	b.WriteString("# Fields with their default value are active.\n")
	b.WriteString("# Fields prefixed with '# ' are commented out and have no effect until uncommented.\n")

	// --- Input / Output ---
	b.WriteString("\n\n# --- Input / Output ---\n")

	wf(&b, true,
		"Path to lyrics file (.lrc or .txt).\nAuto-detected from the audio folder if omitted (same name as audio file).",
		"lyrics", `""`)

	wf(&b, true,
		"Path to a single background image (.jpg, .jpeg, .png, .webp).\nAuto-detected from <audio-folder>/images/ if omitted. Black background if none found.",
		"image", `""`)

	wf(&b, true,
		"Output video file path.\nDefaults to the same folder and filename as the audio file, with .mp4 extension.",
		"output", `""`)

	// --- Quality / Dimensions ---
	b.WriteString("\n\n# --- Quality / Dimensions ---\n")

	wf(&b, true,
		"Output quality preset. When set, overrides width and height.\nValid values: 480p, 720p, 1080p, 1440p",
		"quality", `""`)

	wf(&b, false,
		"Video aspect ratio as W:H. Used with quality to compute width, or independently with width/height.\nExamples: 16:9, 4:3, 21:9, 1:1",
		"aspect-ratio", `"16:9"`)

	wf(&b, false,
		"Video width in pixels. Ignored when quality is set.",
		"width", "1920")

	wf(&b, false,
		"Video height in pixels. Ignored when quality is set.",
		"height", "1080")

	// --- Typography ---
	b.WriteString("\n\n# --- Typography ---\n")

	wf(&b, false,
		"Base font size for lyrics in points.\nAuto-scaled proportionally to width using font-size-reference when not set.",
		"font-size", "38")

	wf(&b, false,
		"Reference font size in pt at 1920px width.\nUsed for auto-scaling when font-size is not explicitly set.",
		"font-size-reference", "38")

	wf(&b, false,
		"Font color for context (non-active) lyric lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors (white, yellow, etc.).",
		"font-color", `"#FFFFFF"`)

	wf(&b, false,
		"Font color for the currently active (highlighted) lyric line.\nAccepts hex (#RRGGBB) or FFmpeg named colors.",
		"highlight-color", `"#FFD700"`)

	// --- Background / Transition ---
	b.WriteString("\n\n# --- Background / Transition ---\n")

	wf(&b, false,
		"Background image dimming factor. 0.0 = pure black overlay, 1.0 = no dimming.",
		"bg-dim", "0.4")

	wf(&b, false,
		"Transition effect between images in a slideshow.\nValid values: fade, fadeblack, fadewhite, dissolve, wipeleft, wiperight,\n             wipeup, wipedown, slideleft, slideright, radial, pixelize, none",
		"transition", `"fade"`)

	wf(&b, false,
		"Duration of the transition effect in seconds.",
		"transition-duration", "3.0")

	// --- Lyric Positioning ---
	b.WriteString("\n\n# --- Lyric Positioning ---\n")

	wf(&b, false,
		"Vertical position of the focused (active) lyric line as a fraction of the frame height.\n0.0 = top of frame, 1.0 = bottom of frame.",
		"lyric-position", "0.65")

	wf(&b, false,
		"Seconds to cross-fade between lyric lines. Set to 0 for hard cuts.",
		"lyric-fade", "0.3")

	wf(&b, false,
		"Alpha curve for lyric cross-fade transitions.\nValid values: linear, smooth",
		"lyric-fade-style", `"linear"`)

	// --- Fade-In ---
	b.WriteString("\n\n# --- Fade-In ---\n")

	wf(&b, true,
		"Seconds to fade the video in from black at the start. Set to 0 to disable.",
		"fade-in-seconds", "0")

	wf(&b, true,
		"Title text to display during the fade-in period.\nUse | to separate multiple lines. Example: \"Song Title|Artist Name\"",
		"fade-in-title", `""`)

	wf(&b, false,
		"Seconds to fade the title out after the fade-in period ends.\nThe title is fully visible during the fade-in, then fades out over this duration.",
		"fade-in-title-fade-out", "1.0")

	wf(&b, false,
		"Font size(s) for the fade-in title in points.\nUse | to set different sizes per line. The last value is used for any remaining lines.\nExample: \"80|60\" sets the first line to 80pt and remaining lines to 60pt.",
		"fade-in-font-size", `"60"`)

	wf(&b, true,
		"Font color(s) for the fade-in title. Defaults to font-color when not set.\nUse | to set different colors per line. The last value is used for any remaining lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors.",
		"fade-in-font-color", `""`)

	// --- Fade-Out ---
	b.WriteString("\n\n# --- Fade-Out ---\n")

	wf(&b, true,
		"Seconds to fade the video out to black at the end. Set to 0 to disable.",
		"fade-out-seconds", "0")

	wf(&b, true,
		"Title text to display during the fade-out period.\nUse | to separate multiple lines. Example: \"The End|© 2025\"",
		"fade-out-title", `""`)

	wf(&b, false,
		"Seconds to fade the title in before the fade-out period begins.\nThe title fades in over this duration, then remains fully visible during the fade-out.",
		"fade-out-title-fade-out", "1.0")

	wf(&b, false,
		"Font size(s) for the fade-out title in points.\nUse | to set different sizes per line. The last value is used for any remaining lines.",
		"fade-out-font-size", `"60"`)

	wf(&b, true,
		"Font color(s) for the fade-out title. Defaults to font-color when not set.\nUse | to set different colors per line. The last value is used for any remaining lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors.",
		"fade-out-font-color", `""`)

	b.WriteString("\n")

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	fmt.Printf("Config written to %s\n", outPath)
	return nil
}

// wf writes a single YAML field entry with descriptive comment lines.
// commented=true prefixes the key: value line with "# " so it has no effect.
func wf(b *strings.Builder, commented bool, comment, key, value string) {
	b.WriteString("\n")
	for _, line := range strings.Split(comment, "\n") {
		fmt.Fprintf(b, "# %s\n", line)
	}
	if commented {
		fmt.Fprintf(b, "# %s: %s\n", key, value)
	} else {
		fmt.Fprintf(b, "%s: %s\n", key, value)
	}
}
