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

	rootCmd.AddCommand(generateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&lyricsPath, "lyrics", "", "Path to lyrics file (.lrc or .txt); auto-detected from audio directory if omitted")
	cmd.Flags().StringVar(&imagePath, "image", "", "Path to background image; auto-detected from FOLDER/images/ if omitted, black background if none found")
	cmd.Flags().StringVar(&outputPath, "output", "output.mp4", "Output video file path")
	cmd.Flags().IntVar(&width, "width", 1920, "Video width in pixels")
	cmd.Flags().IntVar(&height, "height", 1080, "Video height in pixels")
	cmd.Flags().IntVar(&fontSize, "font-size", 38, "Base font size for lyrics")
	cmd.Flags().StringVar(&fontColor, "font-color", "#FFFFFF", "Font color for context lyrics")
	cmd.Flags().StringVar(&highlightColor, "highlight-color", "#FFD700", "Font color for active lyric line")
	cmd.Flags().Float64Var(&bgDim, "bg-dim", 0.4, "Background dimming factor (0.0 = black, 1.0 = no dim)")
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

	// Auto-detect lyrics if not specified
	if lyricsPath == "" {
		folder, stem := audioStem(audioPath)
		for _, ext := range []string{".lrc", ".txt"} {
			candidate := filepath.Join(folder, stem+ext)
			if _, err := os.Stat(candidate); err == nil {
				lyricsPath = candidate
				fmt.Printf("Auto-detected lyrics: %s\n", lyricsPath)
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
		if len(imagePaths) > 0 {
			fmt.Printf("Auto-detected %d image(s) from %s\n", len(imagePaths), filepath.Join(filepath.Dir(audioPath), "images"))
		} else {
			fmt.Println("No images found; using black background")
		}
	} else {
		if err := validateFile(imagePath, []string{".jpg", ".jpeg", ".png", ".webp"}); err != nil {
			return fmt.Errorf("image: %w", err)
		}
		imagePaths = []string{imagePath}
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
	var lines []lyrics.Line
	if lyricsPath != "" {
		fmt.Println("Parsing lyrics...")
		var isLRC bool
		lines, isLRC, err = lyrics.ParseFile(lyricsPath)
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
	} else {
		fmt.Println("No lyrics; rendering video without text")
	}

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
		BgDim:          bgDim,
		Lines:          lines,
		Duration:       duration,
	}

	return video.Render(cfg)
}

func audioStem(audioPath string) (folder, stem string) {
	folder = filepath.Dir(audioPath)
	base := filepath.Base(audioPath)
	stem = strings.TrimSuffix(base, filepath.Ext(base))
	return
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
