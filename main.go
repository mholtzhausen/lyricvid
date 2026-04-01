package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/lyricvid/internal/audio"
	"github.com/user/lyricvid/internal/config"
	"github.com/user/lyricvid/internal/imagegen"
	"github.com/user/lyricvid/internal/lyrics"
	"github.com/user/lyricvid/internal/video"
)

var (
	lyricsPath         string
	imagePath          string
	outputPath         string
	width              int
	height             int
	fontSize           int
	fontColor          string
	highlightColor     string
	bgDim              float64
	transition         string
	transitionDuration float64
	lyricVPosition     float64
	lyricFade          float64

	// config
	configFilePath string

	// generate flags
	generateQuality string
	aspectRatio     string

	// fade-in flags
	fadeInSeconds      float64
	fadeInTitle        string
	fadeInTitleFadeOut float64
	fadeInFontSize     string // pipe-separated sizes
	fadeInFontColor    string

	// fade-out flags
	fadeOutSeconds      float64
	fadeOutTitle        string
	fadeOutTitleFadeOut float64
	fadeOutFontSize     string // pipe-separated sizes
	fadeOutFontColor    string

	// imagegen flags
	imagegenInspirationPath string
	imagegenCount           int
	imagegenAPIKey          string
	imagegenQuality         string
	imagegenStyle           string
	imagegenAspectRatio     string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "lyricvid <audio>",
		Short: "Generate a lyrics music video from an audio file",
		Long: `lyricvid generates an MP4 music video with synchronized lyrics overlay.
It supports LRC (timestamped) and plain text lyrics files.`,
		Args: cobra.ExactArgs(1),
		RunE: runGenerate,
	}

	generateCmd := &cobra.Command{
		Use:   "generate <audio>",
		Short: "Generate a lyrics video",
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerate,
	}

	for _, cmd := range []*cobra.Command{rootCmd, generateCmd} {
		addFlags(cmd)
	}

	setgeminiCmd := &cobra.Command{
		Use:   "set-gemini <api_key>",
		Short: "Store a Gemini API key in ~/.lyricvid.yaml (encrypted)",
		Args:  cobra.ExactArgs(1),
		RunE:  runSetGemini,
	}

	imagegenCmd := &cobra.Command{
		Use:   "image-gen <mp3_file>",
		Short: "Generate scene images from song lyrics using Gemini",
		Args:  cobra.ExactArgs(1),
		RunE:  runImagegen,
	}
	imagegenCmd.Flags().StringVar(&imagegenInspirationPath, "inspiration", "",
		"Path to lyrics/text file for inspiration; auto-detected alongside audio if omitted")
	imagegenCmd.Flags().IntVar(&imagegenCount, "count", 5, "Number of images to generate")
	imagegenCmd.Flags().StringVar(&imagegenAPIKey, "api-key", "",
		"Gemini API key (stored to ~/.lyricvid.yaml if not already saved)")
	imagegenCmd.Flags().StringVar(&imagegenQuality, "quality", "480p",
		"Output quality: 480p, 720p, 1080p, 1440p (16:9 aspect ratio)")
	imagegenCmd.Flags().StringVar(&imagegenStyle, "style", "",
		"Visual style/theme applied to every generated image (e.g. \"cinematic film noir, high contrast\")")
	imagegenCmd.Flags().StringVar(&imagegenAspectRatio, "aspect-ratio", "16:9",
		"Image aspect ratio as W:H passed to Gemini (e.g. 16:9, 4:3, 1:1)")

	createConfigCmd := &cobra.Command{
		Use:   "create-config [path]",
		Short: "Write a YAML config file with all generate defaults",
		Long:  "Writes a fully-commented YAML config file. Defaults to ./lyricvid.yml if no path is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCreateConfig,
	}

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(setgeminiCmd)
	rootCmd.AddCommand(imagegenCmd)
	rootCmd.AddCommand(createConfigCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&configFilePath, "config", "", "Path to a YAML config file (overrides auto-detected lyricvid.yml / <stem>.yml)")
	cmd.Flags().StringVar(&lyricsPath, "lyrics", "", "Path to lyrics file (.lrc or .txt); auto-detected from audio directory if omitted")
	cmd.Flags().StringVar(&imagePath, "image", "", "Path to background image; auto-detected from FOLDER/images/ if omitted, black background if none found")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output video file path (default: same folder and name as audio, with .mp4 extension)")
	cmd.Flags().StringVar(&generateQuality, "quality", "", "Output quality: 480p, 720p, 1080p, 1440p (overrides --width/--height when set)")
	cmd.Flags().StringVar(&aspectRatio, "aspect-ratio", "16:9", "Video aspect ratio as W:H (e.g. 16:9, 4:3, 21:9); used with --quality to compute width")
	cmd.Flags().IntVar(&width, "width", 1920, "Video width in pixels (ignored when --quality is set)")
	cmd.Flags().IntVar(&height, "height", 1080, "Video height in pixels (ignored when --quality is set)")
	cmd.Flags().IntVar(&fontSize, "font-size", 38, "Base font size for lyrics")
	cmd.Flags().StringVar(&fontColor, "font-color", "#FFFFFF", "Font color for context lyrics")
	cmd.Flags().StringVar(&highlightColor, "highlight-color", "#FFD700", "Font color for active lyric line")
	cmd.Flags().Float64Var(&bgDim, "bg-dim", 0.4, "Background dimming factor (0.0 = black, 1.0 = no dim)")
	cmd.Flags().StringVar(&transition, "transition", "fade", "Transition between images: fade|fadeblack|fadewhite|dissolve|wipeleft|wiperight|wipeup|wipedown|slideleft|slideright|radial|pixelize|none")
	cmd.Flags().Float64Var(&transitionDuration, "transition-duration", 3.0, "Duration of transition effect in seconds")
	cmd.Flags().Float64Var(&lyricVPosition, "lyric-position", 0.65, "Vertical position of the focused lyric line (0.0=top, 1.0=bottom)")
	cmd.Flags().Float64Var(&lyricFade, "lyric-fade", 0.3, "Seconds to cross-fade between lyric lines; 0 = hard cut")

	cmd.Flags().Float64Var(&fadeInSeconds, "fade-in-seconds", 0, "Seconds to fade from black at the start; 0 = no fade-in")
	cmd.Flags().StringVar(&fadeInTitle, "fade-in-title", "", "Title text shown during fade-in; use | to separate multiple lines")
	cmd.Flags().Float64Var(&fadeInTitleFadeOut, "fade-in-title-fade-out", 1, "Seconds to fade out the title after the fade-in period ends")
	cmd.Flags().StringVar(&fadeInFontSize, "fade-in-font-size", "60", "Font size(s) for fade-in title; use | to set per-line sizes (last size is the default for remaining lines)")
	cmd.Flags().StringVar(&fadeInFontColor, "fade-in-font-color", "", "Font color(s) for fade-in title; use | for per-line colors (defaults to --font-color)")

	cmd.Flags().Float64Var(&fadeOutSeconds, "fade-out-seconds", 0, "Seconds to fade to black at the end; 0 = no fade-out")
	cmd.Flags().StringVar(&fadeOutTitle, "fade-out-title", "", "Title text shown during fade-out; use | to separate multiple lines")
	cmd.Flags().Float64Var(&fadeOutTitleFadeOut, "fade-out-title-fade-out", 1, "Seconds to fade in the title before the fade-out period starts")
	cmd.Flags().StringVar(&fadeOutFontSize, "fade-out-font-size", "60", "Font size(s) for fade-out title; use | to set per-line sizes (last size is the default for remaining lines)")
	cmd.Flags().StringVar(&fadeOutFontColor, "fade-out-font-color", "", "Font color(s) for fade-out title; use | for per-line colors (defaults to --font-color)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	audioPath := args[0]

	// Check FFmpeg availability
	if err := audio.CheckFFmpeg(); err != nil {
		return err
	}

	// Validate file existence and extensions
	if err := validateFile(audioPath, []string{".mp3", ".m4a", ".flac", ".wav"}); err != nil {
		return fmt.Errorf("audio: %w", err)
	}

	// Load config files (lowest → highest priority):
	//   <audio-dir>/lyricvid.yml, <audio-dir>/<stem>.yml, --config <path>
	audioDir, audioStemName := audioStem(audioPath)
	cfgPaths := []string{
		filepath.Join(audioDir, "lyricvid.yml"),
		filepath.Join(audioDir, audioStemName+".yml"),
	}
	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err != nil {
			return fmt.Errorf("config file not found: %s", configFilePath)
		}
		cfgPaths = append(cfgPaths, configFilePath)
	}
	gc, loadedCfgPaths, err := loadGenerateConfig(cfgPaths)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	applyConfig(cmd, gc)

	// Auto-enable fade-in/out if a title is set but duration was left at 0
	if fadeInTitle != "" && fadeInSeconds == 0 && !cmd.Flags().Changed("fade-in-seconds") {
		fadeInSeconds = 2.0
	}
	if fadeOutTitle != "" && fadeOutSeconds == 0 && !cmd.Flags().Changed("fade-out-seconds") {
		fadeOutSeconds = 3.0
	}

	// Apply quality preset if specified
	if generateQuality != "" {
		preset, ok := imagegen.QualityPresets[generateQuality]
		if !ok {
			return fmt.Errorf("unsupported quality %q; valid values: 480p, 720p, 1080p, 1440p", generateQuality)
		}
		height = preset.Height
		w, err := computeWidthFromAspect(preset.Height, aspectRatio)
		if err != nil {
			return fmt.Errorf("aspect-ratio: %w", err)
		}
		width = w
	}

	// Auto-scale font size proportional to width (reference: 1920px → 38pt)
	// unless the user explicitly set --font-size or a config file provided one
	if !cmd.Flags().Changed("font-size") && gc.FontSize == 0 {
		fontSize = int(math.Round(38.0 * float64(width) / 1920.0))
		if fontSize < 8 {
			fontSize = 8
		}
	}

	// Auto-detect lyrics if not specified
	if lyricsPath == "" {
		folder, stem := audioStem(audioPath)
		for _, ext := range []string{".lrc", ".txt"} {
			candidate := filepath.Join(folder, stem+ext)
			if _, err := os.Stat(candidate); err == nil {
				lyricsPath = candidate
				break
			}
		}
	} else {
		if err := validateFile(lyricsPath, []string{".lrc", ".txt"}); err != nil {
			return fmt.Errorf("lyrics: %w", err)
		}
	}

	// Auto-detect images if not specified
	var imagePaths []string
	if imagePath == "" {
		folder, _ := audioStem(audioPath)
		imagesDir := filepath.Join(folder, "images")
		if info, err := os.Stat(imagesDir); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(imagesDir)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				switch strings.ToLower(filepath.Ext(e.Name())) {
				case ".jpg", ".jpeg", ".png", ".webp":
					imagePaths = append(imagePaths, filepath.Join(imagesDir, e.Name()))
				}
			}
		}
	} else {
		if err := validateFile(imagePath, []string{".jpg", ".jpeg", ".png", ".webp"}); err != nil {
			return fmt.Errorf("image: %w", err)
		}
		imagePaths = []string{imagePath}
	}

	// Default output path: same folder and stem as audio, .mp4 extension
	if outputPath == "" {
		folder, stem := audioStem(audioPath)
		outputPath = filepath.Join(folder, stem+".mp4")
	}

	// Validate numeric parameters
	if width <= 0 || height <= 0 {
		return fmt.Errorf("width and height must be positive integers")
	}
	if fontSize <= 0 {
		return fmt.Errorf("font-size must be a positive integer")
	}
	if bgDim < 0 || bgDim > 1 {
		return fmt.Errorf("bg-dim must be between 0.0 and 1.0")
	}

	// Get audio duration
	duration, err := audio.GetDuration(audioPath)
	if err != nil {
		return fmt.Errorf("getting audio duration: %w", err)
	}

	// Parse lyrics
	var lines []lyrics.Line
	var lyricsDesc string
	if lyricsPath != "" {
		var isLRC bool
		lines, isLRC, err = lyrics.ParseFile(lyricsPath)
		if err != nil {
			return fmt.Errorf("parsing lyrics: %w", err)
		}
		if isLRC {
			lyrics.SetEndTimes(lines, duration)
			lyricsDesc = fmt.Sprintf("%s  (%d lines · LRC)", lyricsPath, len(lines))
		} else {
			lyrics.DistributeEvenly(lines, duration)
			lyricsDesc = fmt.Sprintf("%s  (%d lines · plain text)", lyricsPath, len(lines))
		}
	} else {
		lyricsDesc = "none"
	}

	// Images description
	var imagesDesc string
	switch len(imagePaths) {
	case 0:
		imagesDesc = "black background"
	case 1:
		imagesDesc = imagePaths[0]
	default:
		imagesDesc = fmt.Sprintf("%d images  (%s)", len(imagePaths), filepath.Dir(imagePaths[0]))
	}

	// Transition description
	transitionDesc := "none"
	if transition != "" && transition != "none" {
		transitionDesc = fmt.Sprintf("%s  %.1fs", transition, transitionDuration)
	}

	// Resolve fade title font colors — default to --font-color unless config provided one
	// (must happen before printing so the summary shows the final value)
	if !cmd.Flags().Changed("fade-in-font-color") && gc.FadeInFontColor == "" {
		fadeInFontColor = fontColor
	}
	if !cmd.Flags().Changed("fade-out-font-color") && gc.FadeOutFontColor == "" {
		fadeOutFontColor = fontColor
	}

	// Print summary
	p := func(label, value string) { fmt.Printf("  %-24s %s\n", label, value) }

	if len(loadedCfgPaths) == 0 {
		p("config", "none")
	} else {
		for i, cp := range loadedCfgPaths {
			if i == 0 {
				p("config", cp)
			} else {
				p("", cp)
			}
		}
	}
	fmt.Println()

	p("audio", fmt.Sprintf("%s  (%s)", audioPath, formatDuration(duration)))
	p("lyrics", lyricsDesc)
	p("images", imagesDesc)
	p("output", outputPath)
	fmt.Println()

	p("quality", func() string {
		if generateQuality != "" {
			return generateQuality
		}
		return "auto"
	}())
	p("size", fmt.Sprintf("%dx%d", width, height))
	p("aspect-ratio", aspectRatio)
	p("font-size", fmt.Sprintf("%dpt", fontSize))
	p("font-color", fontColor)
	p("highlight-color", highlightColor)
	p("bg-dim", fmt.Sprintf("%.2f", bgDim))
	p("transition", transitionDesc)
	p("lyric-position", fmt.Sprintf("%.2f", lyricVPosition))
	p("lyric-fade", fmt.Sprintf("%.2fs", lyricFade))
	fmt.Println()

	if fadeInSeconds > 0 {
		p("fade-in", fmt.Sprintf("%.1fs", fadeInSeconds))
		if fadeInTitle != "" {
			p("fade-in-title", fadeInTitle)
			p("fade-in-title-fade-out", fmt.Sprintf("%.1fs", fadeInTitleFadeOut))
			p("fade-in-font-size", fadeInFontSize)
			p("fade-in-font-color", fadeInFontColor)
		}
	} else {
		p("fade-in", "disabled")
	}
	if fadeOutSeconds > 0 {
		p("fade-out", fmt.Sprintf("%.1fs", fadeOutSeconds))
		if fadeOutTitle != "" {
			p("fade-out-title", fadeOutTitle)
			p("fade-out-title-fade-out", fmt.Sprintf("%.1fs", fadeOutTitleFadeOut))
			p("fade-out-font-size", fadeOutFontSize)
			p("fade-out-font-color", fadeOutFontColor)
		}
	} else {
		p("fade-out", "disabled")
	}
	fmt.Println()

	// Render video
	cfg := video.Config{
		AudioPath:      audioPath,
		ImagePaths:     imagePaths,
		OutputPath:     outputPath,
		Width:          width,
		Height:         height,
		FontSize:       fontSize,
		FontColor:      fontColor,
		HighlightColor: highlightColor,
		BgDim:              bgDim,
		Lines:              lines,
		Duration:           duration,
		Transition:         transition,
		TransitionDuration: transitionDuration,
		LyricVPosition:     lyricVPosition,
		LyricFade:          lyricFade,

		FadeInSeconds:      fadeInSeconds,
		FadeInTitle:        fadeInTitle,
		FadeInTitleFadeOut: fadeInTitleFadeOut,
		FadeInFontSizes:    parsePipeSizes(fadeInFontSize, 60),
		FadeInFontColors:   parsePipeColors(fadeInFontColor),

		FadeOutSeconds:      fadeOutSeconds,
		FadeOutTitle:        fadeOutTitle,
		FadeOutTitleFadeOut: fadeOutTitleFadeOut,
		FadeOutFontSizes:    parsePipeSizes(fadeOutFontSize, 60),
		FadeOutFontColors:   parsePipeColors(fadeOutFontColor),
	}

	return video.Render(cfg)
}

func runSetGemini(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.GeminiAPIKey = args[0]
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("Gemini API key saved to ~/.lyricvid.yaml")
	return nil
}

func runImagegen(_ *cobra.Command, args []string) error {
	audioPath := args[0]

	if err := validateFile(audioPath, []string{".mp3", ".m4a", ".flac", ".wav"}); err != nil {
		return fmt.Errorf("audio: %w", err)
	}

	preset, ok := imagegen.QualityPresets[imagegenQuality]
	if !ok {
		return fmt.Errorf("unsupported quality %q; valid values: 480p, 720p, 1080p, 1440p", imagegenQuality)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if imagegenAPIKey != "" {
		cfg.GeminiAPIKey = imagegenAPIKey
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("Gemini API key saved to ~/.lyricvid.yaml")
	}
	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("no Gemini API key; use --api-key or run 'set-gemini <key>'")
	}

	inspirationPath := imagegenInspirationPath
	if inspirationPath == "" {
		folder, stem := audioStem(audioPath)
		for _, ext := range []string{".lrc", ".txt"} {
			candidate := filepath.Join(folder, stem+ext)
			if _, err := os.Stat(candidate); err == nil {
				inspirationPath = candidate
				fmt.Printf("Auto-detected inspiration: %s\n", inspirationPath)
				break
			}
		}
		if inspirationPath == "" {
			return fmt.Errorf("no inspiration file found; provide one with --inspiration")
		}
	} else {
		if err := validateFile(inspirationPath, []string{".lrc", ".txt"}); err != nil {
			return fmt.Errorf("inspiration: %w", err)
		}
	}

	inspirationText, err := os.ReadFile(inspirationPath)
	if err != nil {
		return fmt.Errorf("reading inspiration file: %w", err)
	}

	folder, _ := audioStem(audioPath)
	outputDir := filepath.Join(folder, "images")

	return imagegen.Generate(context.Background(), imagegen.Config{
		APIKey:          cfg.GeminiAPIKey,
		InspirationText: strings.TrimSpace(string(inspirationText)),
		Count:           imagegenCount,
		OutputDir:       outputDir,
		ImageSize:       preset.ImageSize,
		Style:           imagegenStyle,
		AspectRatio:     imagegenAspectRatio,
	})
}

func formatDuration(secs float64) string {
	total := int(math.Round(secs))
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func computeWidthFromAspect(height int, ratio string) (int, error) {
	parts := strings.SplitN(ratio, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid aspect ratio %q, expected W:H (e.g. 16:9)", ratio)
	}
	wPart, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	hPart, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || hPart == 0 {
		return 0, fmt.Errorf("invalid aspect ratio %q, expected W:H (e.g. 16:9)", ratio)
	}
	w := int(math.Round(float64(height) * wPart / hPart))
	if w%2 != 0 {
		w++ // libx264 requires even dimensions
	}
	return w, nil
}

func audioStem(audioPath string) (folder, stem string) {
	folder = filepath.Dir(audioPath)
	base := filepath.Base(audioPath)
	stem = strings.TrimSuffix(base, filepath.Ext(base))
	return
}

func parsePipeSizes(s string, defaultSize int) []int {
	if s == "" {
		return []int{defaultSize}
	}
	parts := strings.Split(s, "|")
	var sizes []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			n = defaultSize
		}
		sizes = append(sizes, n)
	}
	if len(sizes) == 0 {
		return []int{defaultSize}
	}
	return sizes
}

func parsePipeColors(s string) []string {
	if s == "" {
		return []string{"#FFFFFF"}
	}
	parts := strings.Split(s, "|")
	var colors []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			colors = append(colors, p)
		}
	}
	if len(colors) == 0 {
		return []string{"#FFFFFF"}
	}
	return colors
}

func validateFile(path string, allowedExts []string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range allowedExts {
		if ext == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported file extension %q (allowed: %s)", ext, strings.Join(allowedExts, ", "))
}
