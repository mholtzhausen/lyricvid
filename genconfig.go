package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	Drift               string  `yaml:"drift,omitempty"`
	DriftMax            int     `yaml:"drift-max,omitempty"`
	DriftMin            int     `yaml:"drift-min,omitempty"`
	DriftDurationPct    float64 `yaml:"drift-duration-percentage,omitempty"`
	DriftEasing         string  `yaml:"drift-easing,omitempty"`
	EnableCUDA          string  `yaml:"enable-cuda,omitempty"` // "true" or "false"; empty = not configured

	Framerate           int     `yaml:"framerate,omitempty"`

	// audio visualizer settings
	VisualizerType     string  `yaml:"visualizer-type,omitempty"`
	VisualizerColor    string  `yaml:"visualizer-color,omitempty"`
	VisualizerHeight   float64 `yaml:"visualizer-height,omitempty"`
	VisualizerPosition string  `yaml:"visualizer-position,omitempty"`
	VisualizerOpacity  float64 `yaml:"visualizer-opacity,omitempty"`
	VisualizerMode     string  `yaml:"visualizer-mode,omitempty"`
	VisualizerScale    string  `yaml:"visualizer-scale,omitempty"`
	VisualizerSlide    string  `yaml:"visualizer-slide,omitempty"`
	VisualizerStart    float64 `yaml:"visualizer-start,omitempty"`
	VisualizerEnd      float64 `yaml:"visualizer-end,omitempty"`
	VisualizerFade     float64 `yaml:"visualizer-fade,omitempty"`

	// image-gen settings
	ImagegenInspirationText string `yaml:"imagegen-inspiration,omitempty"`
	ImagegenStyle           string `yaml:"imagegen-style,omitempty"`
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
		if tmp.Drift != "" {
			acc.Drift = tmp.Drift
		}
		if tmp.DriftMax != 0 {
			acc.DriftMax = tmp.DriftMax
		}
		if tmp.DriftMin != 0 {
			acc.DriftMin = tmp.DriftMin
		}
		if tmp.DriftDurationPct != 0 {
			acc.DriftDurationPct = tmp.DriftDurationPct
		}
		if tmp.DriftEasing != "" {
			acc.DriftEasing = tmp.DriftEasing
		}
		if tmp.EnableCUDA != "" {
			acc.EnableCUDA = tmp.EnableCUDA
		}
		if tmp.Framerate != 0 {
			acc.Framerate = tmp.Framerate
		}
		if tmp.VisualizerType != "" {
			acc.VisualizerType = tmp.VisualizerType
		}
		if tmp.VisualizerColor != "" {
			acc.VisualizerColor = tmp.VisualizerColor
		}
		if tmp.VisualizerHeight != 0 {
			acc.VisualizerHeight = tmp.VisualizerHeight
		}
		if tmp.VisualizerPosition != "" {
			acc.VisualizerPosition = tmp.VisualizerPosition
		}
		if tmp.VisualizerOpacity != 0 {
			acc.VisualizerOpacity = tmp.VisualizerOpacity
		}
		if tmp.VisualizerMode != "" {
			acc.VisualizerMode = tmp.VisualizerMode
		}
		if tmp.VisualizerScale != "" {
			acc.VisualizerScale = tmp.VisualizerScale
		}
		if tmp.VisualizerSlide != "" {
			acc.VisualizerSlide = tmp.VisualizerSlide
		}
		if tmp.VisualizerStart != 0 {
			acc.VisualizerStart = tmp.VisualizerStart
		}
		if tmp.VisualizerEnd != 0 {
			acc.VisualizerEnd = tmp.VisualizerEnd
		}
		if tmp.VisualizerFade != 0 {
			acc.VisualizerFade = tmp.VisualizerFade
		}
		if tmp.ImagegenInspirationText != "" {
			acc.ImagegenInspirationText = tmp.ImagegenInspirationText
		}
		if tmp.ImagegenStyle != "" {
			acc.ImagegenStyle = tmp.ImagegenStyle
		}
	}
	return acc, loaded, nil
}

// saveGenerateConfig writes the CLI-changed flags into the YAML file at path,
// merging with any existing content. The --save flag itself is never written.
// saveGenerateConfig writes only the CLI-changed flags into the YAML file at
// path, leaving all other content (and comments) untouched. Keys that already
// exist in the file have their values updated in-place; new keys are appended.
func saveGenerateConfig(cmd *cobra.Command, path string) error {
	// Map every flag name to its YAML key (they're identical for all generate flags).
	allFlags := []string{
		"lyrics", "image", "output", "quality", "aspect-ratio",
		"width", "height", "font-size", "font-size-reference",
		"font-color", "highlight-color", "bg-dim",
		"transition", "transition-duration",
		"lyric-position", "lyric-fade", "lyric-fade-style",
		"fade-in-seconds", "fade-in-title", "fade-in-title-fade-out",
		"fade-in-font-size", "fade-in-font-color",
		"fade-out-seconds", "fade-out-title", "fade-out-title-fade-out",
		"fade-out-font-size", "fade-out-font-color",
		"drift", "drift-max", "drift-min", "drift-duration-percentage", "drift-easing",
		"enable-cuda",
		"framerate",
		"visualizer-type", "visualizer-color", "visualizer-height",
		"visualizer-position", "visualizer-opacity", "visualizer-mode", "visualizer-scale", "visualizer-slide",
		"visualizer-start", "visualizer-end", "visualizer-fade",
	}

	type entry struct{ val, typ string }
	updates := map[string]entry{}
	for _, name := range allFlags {
		if cmd.Flags().Changed(name) {
			f := cmd.Flags().Lookup(name)
			updates[name] = entry{f.Value.String(), f.Value.Type()}
		}
	}
	if len(updates) == 0 {
		return nil
	}

	// Load existing file into a yaml.Node tree (preserves comments).
	var docNode yaml.Node
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &docNode); err != nil {
			return fmt.Errorf("parsing config %q: %w", path, err)
		}
	}
	if docNode.Kind == 0 {
		docNode = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Tag: "!!map"},
			},
		}
	}
	mapping := docNode.Content[0]

	for key, u := range updates {
		yamlSetValue(mapping, key, u.val, u.typ)
	}

	// Marshal the document node; yaml.v3 prepends "---\n" for DocumentNode.
	out, err := yaml.Marshal(&docNode)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	out = bytes.TrimPrefix(out, []byte("---\n"))
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing config %q: %w", path, err)
	}
	return nil
}

// yamlSetValue finds key in a mapping node and updates its value, or appends
// a new key/value pair if the key is not found.
func yamlSetValue(mapping *yaml.Node, key, val, pflType string) {
	tag := pflTypeToYAMLTag(pflType)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Value = val
			mapping.Content[i+1].Tag = tag
			return
		}
	}
	// Not found — append.
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: val},
	)
}

func pflTypeToYAMLTag(pflType string) string {
	switch pflType {
	case "int":
		return "!!int"
	case "float64":
		return "!!float"
	case "bool":
		return "!!bool"
	default:
		return "!!str"
	}
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
	if gc.Drift != "" && !cmd.Flags().Changed("drift") {
		driftTypes = gc.Drift
	}
	if gc.DriftMax != 0 && !cmd.Flags().Changed("drift-max") {
		driftMax = gc.DriftMax
	}
	if gc.DriftMin != 0 && !cmd.Flags().Changed("drift-min") {
		driftMin = gc.DriftMin
	}
	if gc.DriftDurationPct != 0 && !cmd.Flags().Changed("drift-duration-percentage") {
		driftDurationPct = gc.DriftDurationPct
	}
	if gc.DriftEasing != "" && !cmd.Flags().Changed("drift-easing") {
		driftEasing = gc.DriftEasing
	}
	if gc.EnableCUDA != "" && !cmd.Flags().Changed("enable-cuda") {
		enableCUDA = gc.EnableCUDA == "true"
	}
	if gc.Framerate != 0 && !cmd.Flags().Changed("framerate") {
		framerate = gc.Framerate
	}
	if gc.VisualizerType != "" && !cmd.Flags().Changed("visualizer-type") {
		visualizerType = gc.VisualizerType
	}
	if gc.VisualizerColor != "" && !cmd.Flags().Changed("visualizer-color") {
		visualizerColor = gc.VisualizerColor
	}
	if gc.VisualizerHeight != 0 && !cmd.Flags().Changed("visualizer-height") {
		visualizerHeight = gc.VisualizerHeight
	}
	if gc.VisualizerPosition != "" && !cmd.Flags().Changed("visualizer-position") {
		visualizerPosition = gc.VisualizerPosition
	}
	if gc.VisualizerOpacity != 0 && !cmd.Flags().Changed("visualizer-opacity") {
		visualizerOpacity = gc.VisualizerOpacity
	}
	if gc.VisualizerMode != "" && !cmd.Flags().Changed("visualizer-mode") {
		visualizerMode = gc.VisualizerMode
	}
	if gc.VisualizerScale != "" && !cmd.Flags().Changed("visualizer-scale") {
		visualizerScale = gc.VisualizerScale
	}
	if gc.VisualizerSlide != "" && !cmd.Flags().Changed("visualizer-slide") {
		visualizerSlide = gc.VisualizerSlide
	}
	if gc.VisualizerStart != 0 && !cmd.Flags().Changed("visualizer-start") {
		visualizerStart = gc.VisualizerStart
	}
	if gc.VisualizerEnd != 0 && !cmd.Flags().Changed("visualizer-end") {
		visualizerEnd = gc.VisualizerEnd
	}
	if gc.VisualizerFade != 0 && !cmd.Flags().Changed("visualizer-fade") {
		visualizerFade = gc.VisualizerFade
	}
}

// rStr resolves a string config field. If existing is non-empty, returns it quoted
// and active (uncommented); otherwise returns the factory default with its original state.
func rStr(existing, factory string, commented bool) (string, bool) {
	if existing != "" {
		return fmt.Sprintf("%q", existing), false
	}
	return factory, commented
}

// rInt resolves an int config field.
func rInt(existing int, factory string, commented bool) (string, bool) {
	if existing != 0 {
		return strconv.Itoa(existing), false
	}
	return factory, commented
}

// rFloat resolves a float64 config field.
func rFloat(existing float64, factory string, commented bool) (string, bool) {
	if existing != 0 {
		return strconv.FormatFloat(existing, 'f', -1, 64), false
	}
	return factory, commented
}

// buildConfigTemplate returns the fully-commented YAML config template as a string.
// gc carries any pre-existing values to substitute in place of factory defaults;
// pass a zero-value GenerateConfig for a clean template with factory defaults.
func buildConfigTemplate(gc GenerateConfig) string {
	var b strings.Builder

	b.WriteString("# lyricvid configuration file\n")
	b.WriteString("# Generated by: lyricvid config-file\n")
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

	v, c := rStr(gc.Lyrics, `""`, true)
	wf(&b, c, "Path to lyrics file (.lrc or .txt).\nAuto-detected from the audio folder if omitted (same name as audio file).", "lyrics", v)

	v, c = rStr(gc.Image, `""`, true)
	wf(&b, c, "Path to a single background image (.jpg, .jpeg, .png, .webp).\nAuto-detected from <audio-folder>/images/ if omitted. Black background if none found.", "image", v)

	v, c = rStr(gc.Output, `""`, true)
	wf(&b, c, "Output video file path.\nDefaults to the same folder and filename as the audio file, with .mp4 extension.", "output", v)

	// --- Quality / Dimensions ---
	b.WriteString("\n\n# --- Quality / Dimensions ---\n")

	v, c = rStr(gc.Quality, `""`, true)
	wf(&b, c, "Output quality preset. When set, overrides width and height.\nValid values: 480p, 720p, 1080p, 1440p", "quality", v)

	v, c = rStr(gc.AspectRatio, `"16:9"`, false)
	wf(&b, c, "Video aspect ratio as W:H. Used with quality to compute width, or independently with width/height.\nExamples: 16:9, 4:3, 21:9, 1:1", "aspect-ratio", v)

	v, c = rInt(gc.Width, "1920", false)
	wf(&b, c, "Video width in pixels. Ignored when quality is set.", "width", v)

	v, c = rInt(gc.Height, "1080", false)
	wf(&b, c, "Video height in pixels. Ignored when quality is set.", "height", v)

	// --- Typography ---
	b.WriteString("\n\n# --- Typography ---\n")

	v, c = rInt(gc.FontSize, "38", false)
	wf(&b, c, "Base font size for lyrics in points.\nAuto-scaled proportionally to width using font-size-reference when not set.", "font-size", v)

	v, c = rInt(gc.FontSizeReference, "38", false)
	wf(&b, c, "Reference font size in pt at 1920px width.\nUsed for auto-scaling when font-size is not explicitly set.", "font-size-reference", v)

	v, c = rStr(gc.FontColor, `"#FFFFFF"`, false)
	wf(&b, c, "Font color for context (non-active) lyric lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors (white, yellow, etc.).", "font-color", v)

	v, c = rStr(gc.HighlightColor, `"#FFD700"`, false)
	wf(&b, c, "Font color for the currently active (highlighted) lyric line.\nAccepts hex (#RRGGBB) or FFmpeg named colors.", "highlight-color", v)

	// --- Background / Transition ---
	b.WriteString("\n\n# --- Background / Transition ---\n")

	v, c = rFloat(gc.BgDim, "0.4", false)
	wf(&b, c, "Background image dimming factor. 0.0 = pure black overlay, 1.0 = no dimming.", "bg-dim", v)

	v, c = rStr(gc.Transition, `"fade"`, false)
	wf(&b, c, "Transition effect between images in a slideshow.\nValid values: fade, fadeblack, fadewhite, dissolve, wipeleft, wiperight,\n             wipeup, wipedown, slideleft, slideright, radial, pixelize, none", "transition", v)

	v, c = rFloat(gc.TransitionDuration, "3.0", false)
	wf(&b, c, "Duration of the transition effect in seconds.", "transition-duration", v)

	// --- Lyric Positioning ---
	b.WriteString("\n\n# --- Lyric Positioning ---\n")

	v, c = rFloat(gc.LyricPosition, "0.65", false)
	wf(&b, c, "Vertical position of the focused (active) lyric line as a fraction of the frame height.\n0.0 = top of frame, 1.0 = bottom of frame.", "lyric-position", v)

	v, c = rFloat(gc.LyricFade, "0.3", false)
	wf(&b, c, "Seconds to cross-fade between lyric lines. Set to 0 for hard cuts.", "lyric-fade", v)

	v, c = rStr(gc.LyricFadeStyle, `"linear"`, false)
	wf(&b, c, "Alpha curve for lyric cross-fade transitions.\nValid values: linear, smooth", "lyric-fade-style", v)

	// --- Fade-In ---
	b.WriteString("\n\n# --- Fade-In ---\n")

	v, c = rFloat(gc.FadeInSeconds, "0", true)
	wf(&b, c, "Seconds to fade the video in from black at the start. Set to 0 to disable.", "fade-in-seconds", v)

	v, c = rStr(gc.FadeInTitle, `""`, true)
	wf(&b, c, "Title text to display during the fade-in period.\nUse | to separate multiple lines. Example: \"Song Title|Artist Name\"", "fade-in-title", v)

	v, c = rFloat(gc.FadeInTitleFadeOut, "1.0", false)
	wf(&b, c, "Seconds to fade the title out after the fade-in period ends.\nThe title is fully visible during the fade-in, then fades out over this duration.", "fade-in-title-fade-out", v)

	v, c = rStr(gc.FadeInFontSize, `"60"`, false)
	wf(&b, c, "Font size(s) for the fade-in title in points.\nUse | to set different sizes per line. The last value is used for any remaining lines.\nExample: \"80|60\" sets the first line to 80pt and remaining lines to 60pt.", "fade-in-font-size", v)

	v, c = rStr(gc.FadeInFontColor, `""`, true)
	wf(&b, c, "Font color(s) for the fade-in title. Defaults to font-color when not set.\nUse | to set different colors per line. The last value is used for any remaining lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors.", "fade-in-font-color", v)

	// --- Fade-Out ---
	b.WriteString("\n\n# --- Fade-Out ---\n")

	v, c = rFloat(gc.FadeOutSeconds, "0", true)
	wf(&b, c, "Seconds to fade the video out to black at the end. Set to 0 to disable.", "fade-out-seconds", v)

	v, c = rStr(gc.FadeOutTitle, `""`, true)
	wf(&b, c, "Title text to display during the fade-out period.\nUse | to separate multiple lines. Example: \"The End|© 2025\"", "fade-out-title", v)

	v, c = rFloat(gc.FadeOutTitleFadeOut, "1.0", false)
	wf(&b, c, "Seconds to fade the title in before the fade-out period begins.\nThe title fades in over this duration, then remains fully visible during the fade-out.", "fade-out-title-fade-out", v)

	v, c = rStr(gc.FadeOutFontSize, `"60"`, false)
	wf(&b, c, "Font size(s) for the fade-out title in points.\nUse | to set different sizes per line. The last value is used for any remaining lines.", "fade-out-font-size", v)

	v, c = rStr(gc.FadeOutFontColor, `""`, true)
	wf(&b, c, "Font color(s) for the fade-out title. Defaults to font-color when not set.\nUse | to set different colors per line. The last value is used for any remaining lines.\nAccepts hex (#RRGGBB) or FFmpeg named colors.", "fade-out-font-color", v)

	// --- Drift ---
	b.WriteString("\n\n# --- Drift ---\n")

	v, c = rStr(gc.Drift, `""`, true)
	wf(&b, c, "Background pan/zoom animation applied to each image.\nComma-separated list of drift types; one is chosen randomly per image.\nValid values: random, left, right, up, down, zoom-in, zoom-out\n  random           — pick from all types\n  left/right/up/down — slow pan in that direction\n  zoom-in/zoom-out   — slow zoom animation\nLeave empty to disable.", "drift", v)

	v, c = rInt(gc.DriftMax, "60", false)
	wf(&b, c, "Maximum drift distance in pixels (applies to both pan offset and zoom margin).", "drift-max", v)

	v, c = rInt(gc.DriftMin, "10", false)
	wf(&b, c, "Minimum drift distance in pixels.", "drift-min", v)

	v, c = rFloat(gc.DriftDurationPct, "90", false)
	wf(&b, c, "Percentage of each image\u2019s display time used for the drift animation (1\u2013100).\nThe image holds at the final drift position for the remainder of its display time.", "drift-duration-percentage", v)

	v, c = rStr(gc.DriftEasing, `"quad"`, false)
	wf(&b, c, "Easing curve applied to the drift motion.\nValid values: linear, quad (default), cubic, smooth\n  linear — constant speed\n  quad   — ease-out quadratic (decelerates, natural camera feel)\n  cubic  — stronger deceleration\n  smooth — ease-in-out (starts and ends slowly)", "drift-easing", v)

	// --- Audio Visualizer ---
	b.WriteString("\n\n# --- Audio Visualizer ---\n")

	v, c = rStr(gc.VisualizerType, `"none"`, true)
	wf(&b, c, "Audio visualizer overlay drawn on top of the video.\nValid values: none, waveform, spectrum, freqs\n  none      — disabled (default)\n  waveform  — animated audio waveform\n  spectrum  — frequency spectrogram\n  freqs     — frequency bar graph", "visualizer-type", v)

	v, c = rStr(gc.VisualizerColor, `"white"`, false)
	wf(&b, c, "Color of the visualizer. Accepts hex (#RRGGBB) or FFmpeg named colors (white, cyan, etc.).", "visualizer-color", v)

	v, c = rFloat(gc.VisualizerHeight, "0.15", false)
	wf(&b, c, "Height of the visualizer as a fraction of the video height (0.0–1.0).\nExample: 0.15 = bottom 15% of the frame.", "visualizer-height", v)

	v, c = rStr(gc.VisualizerPosition, `"bottom"`, false)
	wf(&b, c, "Vertical placement of the visualizer.\nValid values: top, bottom", "visualizer-position", v)

	v, c = rFloat(gc.VisualizerOpacity, "0.8", false)
	wf(&b, c, "Opacity of the visualizer overlay (0.0 = fully transparent, 1.0 = fully opaque).", "visualizer-opacity", v)

	v, c = rStr(gc.VisualizerMode, `"line"`, false)
	wf(&b, c, "Drawing mode for the waveform visualizer (only used when visualizer-type is waveform).\nValid values: line, point, p2p, cline", "visualizer-mode", v)

	v, c = rStr(gc.VisualizerScale, `"log"`, false)
	wf(&b, c, "Amplitude/magnitude scale for spectrum and freqs visualizers (ignored for waveform).\nValid values: lin, log, sqrt, cbrt", "visualizer-scale", v)

	v, c = rStr(gc.VisualizerSlide, `"scroll"`, false)
	wf(&b, c, "Sliding mode for the spectrum visualizer (ignored for waveform and freqs).\nValid values: replace, scroll, fullframe, rscroll, lreplace", "visualizer-slide", v)

	v, c = rFloat(gc.VisualizerStart, "10", false)
	wf(&b, c, "Time in seconds from the start of audio when the visualizer appears.\nSet to 0 to show from the very beginning.", "visualizer-start", v)

	v, c = rFloat(gc.VisualizerEnd, "-1", false)
	wf(&b, c, "Time in seconds from the start of audio when the visualizer disappears.\nNegative value = 10 seconds before the end of the audio.", "visualizer-end", v)

	v, c = rFloat(gc.VisualizerFade, "1", false)
	wf(&b, c, "Seconds to fade the visualizer in at its start time and out at its end time.\nSet to 0 for an instant cut.", "visualizer-fade", v)

	// --- Hardware Acceleration ---
	b.WriteString("\n\n# --- Hardware Acceleration ---\n")

	v, c = rStr(gc.EnableCUDA, `"true"`, false)
	wf(&b, c, "Enable CUDA hardware acceleration for encoding (h264_nvenc).\nWhen true, CUDA availability is probed on each run; falls back to libx264 if not found.\nSet to false to always use CPU encoding (libx264).\nValid values: true, false", "enable-cuda", v)

	v, c = rInt(gc.Framerate, "0", true)
	wf(&b, c, "Output frame rate in fps. Set to 0 to use FFmpeg's default.", "framerate", v)

	// --- Image Generation ---
	b.WriteString("\n\n# --- Image Generation ---\n")

	b.WriteString("\n")
	b.WriteString("# Visual style/theme applied to every image generated by 'lyricvid image-gen'.\n")
	b.WriteString("# Overridden by the --style flag.\n")
	b.WriteString("# Example: \"cinematic film noir, high contrast, muted tones\"\n")
	v, c = rStr(gc.ImagegenStyle, `""`, true)
	wf(&b, c, "", "imagegen-style", v)

	b.WriteString("\n")
	b.WriteString("# Inspiration text for the 'lyricvid image-gen' command.\n")
	b.WriteString("# Embed your lyrics or descriptive text directly instead of a separate file.\n")
	b.WriteString("# Used when --inspiration is not given and no .lrc/.txt is auto-detected.\n")
	b.WriteString("# Use {lyrics} as a placeholder — it will be replaced with the song lyrics at runtime.\n")
	b.WriteString("# Use a YAML literal block scalar (|) for multi-line content.\n")
	if gc.ImagegenInspirationText != "" {
		fmt.Fprintf(&b, "imagegen-inspiration: |\n")
		for _, line := range strings.Split(gc.ImagegenInspirationText, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	} else {
		b.WriteString("# imagegen-inspiration: |\n")
		b.WriteString("#   Paste your song lyrics or scene description here.\n")
		b.WriteString("#   Each line is preserved as-is.\n")
	}

	b.WriteString("\n")

	return b.String()
}

// runConfigFile writes a fully-commented YAML config template.
// If the target file already exists its current values are read and carried
// over into the fresh template, preserving settings while updating comments
// and adding any fields introduced in newer versions of lyricvid.
func runConfigFile(_ *cobra.Command, args []string) error {
	outPath := filepath.Join(".", "lyricvid.yml")
	if len(args) > 0 {
		outPath = args[0]
	}

	var gc GenerateConfig
	if _, err := os.Stat(outPath); err == nil {
		loaded, _, err := loadGenerateConfig([]string{outPath})
		if err != nil {
			return fmt.Errorf("reading existing config %q: %w", outPath, err)
		}
		gc = loaded
		fmt.Printf("Merging existing values from %s\n", outPath)
	}

	if err := os.WriteFile(outPath, []byte(buildConfigTemplate(gc)), 0644); err != nil {
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
