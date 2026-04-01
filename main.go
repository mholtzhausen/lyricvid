package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/lyricvid/internal/audio"
	"github.com/user/lyricvid/internal/lyrics"
	"github.com/user/lyricvid/internal/video"
)

var (
	audioPath      string
	lyricsPath     string
	imagePath      string
	outputPath     string
	width          int
	height         int
	fontSize       int
	fontColor      string
	highlightColor string
	bgDim          float64
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "lyricvid",
		Short: "Generate a lyrics music video from an MP3, lyrics file, and background image",
		Long: `lyricvid generates an MP4 music video with synchronized lyrics overlay.
It supports LRC (timestamped) and plain text lyrics files.`,
		RunE: runGenerate,
	}

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a lyrics video",
		RunE:  runGenerate,
	}

	for _, cmd := range []*cobra.Command{rootCmd, generateCmd} {
		addFlags(cmd)
	}

	rootCmd.AddCommand(generateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&audioPath, "audio", "", "Path to audio file (required)")
	cmd.Flags().StringVar(&lyricsPath, "lyrics", "", "Path to lyrics file (.lrc or .txt) (required)")
	cmd.Flags().StringVar(&imagePath, "image", "", "Path to background image (required)")
	cmd.Flags().StringVar(&outputPath, "output", "output.mp4", "Output video file path")
	cmd.Flags().IntVar(&width, "width", 1920, "Video width in pixels")
	cmd.Flags().IntVar(&height, "height", 1080, "Video height in pixels")
	cmd.Flags().IntVar(&fontSize, "font-size", 38, "Base font size for lyrics")
	cmd.Flags().StringVar(&fontColor, "font-color", "#FFFFFF", "Font color for context lyrics")
	cmd.Flags().StringVar(&highlightColor, "highlight-color", "#FFD700", "Font color for active lyric line")
	cmd.Flags().Float64Var(&bgDim, "bg-dim", 0.4, "Background dimming factor (0.0 = black, 1.0 = no dim)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Validate required flags
	if audioPath == "" {
		return fmt.Errorf("--audio is required")
	}
	if lyricsPath == "" {
		return fmt.Errorf("--lyrics is required")
	}
	if imagePath == "" {
		return fmt.Errorf("--image is required")
	}

	// Check FFmpeg availability
	if err := audio.CheckFFmpeg(); err != nil {
		return err
	}

	// Validate file existence and extensions
	if err := validateFile(audioPath, []string{".mp3", ".m4a", ".flac", ".wav"}); err != nil {
		return fmt.Errorf("audio: %w", err)
	}
	if err := validateFile(lyricsPath, []string{".lrc", ".txt"}); err != nil {
		return fmt.Errorf("lyrics: %w", err)
	}
	if err := validateFile(imagePath, []string{".jpg", ".jpeg", ".png", ".webp"}); err != nil {
		return fmt.Errorf("image: %w", err)
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
	fmt.Println("Analyzing audio...")
	duration, err := audio.GetDuration(audioPath)
	if err != nil {
		return fmt.Errorf("getting audio duration: %w", err)
	}
	fmt.Printf("Audio duration: %.1f seconds\n", duration)

	// Parse lyrics
	fmt.Println("Parsing lyrics...")
	lines, isLRC, err := lyrics.ParseFile(lyricsPath)
	if err != nil {
		return fmt.Errorf("parsing lyrics: %w", err)
	}

	if isLRC {
		fmt.Printf("Parsed %d timestamped lyric lines (LRC format)\n", len(lines))
		lyrics.SetEndTimes(lines, duration)
	} else {
		fmt.Printf("Parsed %d lyric lines (plain text, distributing evenly)\n", len(lines))
		lyrics.DistributeEvenly(lines, duration)
	}

	// Render video
	cfg := video.Config{
		AudioPath:      audioPath,
		ImagePath:      imagePath,
		OutputPath:     outputPath,
		Width:          width,
		Height:         height,
		FontSize:       fontSize,
		FontColor:      fontColor,
		HighlightColor: highlightColor,
		BgDim:          bgDim,
		Lines:          lines,
		Duration:       duration,
	}

	return video.Render(cfg)
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
