package video

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
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
	ImagePaths     []string // empty = black bg, 1 = single image, N = slideshow
	OutputPath     string
	Width          int
	Height         int
	FontSize       int
	FontColor      string
	HighlightColor string
	BgDim          float64
	Lines              []lyrics.Line
	Duration           float64
	Transition         string  // xfade transition name, or "none"
	TransitionDuration float64 // seconds
	LyricVPosition     float64 // vertical position of focused lyric, 0.0=top 1.0=bottom
	LyricFade          float64 // seconds to cross-fade between lyric lines; 0 = hard cut
	LyricFadeStyle     string  // alpha curve for lyric cross-fade: "linear" (default) or "smooth"

	FadeInSeconds      float64  // 0 = no fade-in
	FadeInTitle        string   // pipe-separated lines; empty = no title
	FadeInTitleFadeOut float64  // seconds to fade out title after fade-in ends
	FadeInFontSizes    []int    // per-line sizes; last entry used for overflow
	FadeInFontColors   []string // per-line colors; last entry used for overflow

	FadeOutSeconds      float64
	FadeOutTitle        string
	FadeOutTitleFadeOut float64
	FadeOutFontSizes    []int
	FadeOutFontColors   []string // per-line colors; last entry used for overflow

	DriftTypes       []string // drift types to pick from (may include "random"); empty = disabled
	DriftMax         int      // maximum drift in pixels
	DriftMin         int      // minimum drift in pixels
	DriftDurationPct float64  // percentage of image duration to animate (0–100)
	DriftEasing      string   // easing curve: linear, quad (default), cubic, smooth

	MaxLength  float64 // 0 = no limit; positive = truncate output at this many seconds

	EnableCUDA bool // use h264_nvenc encoder when true

	VisualizerType     string  // "waveform", "spectrum", "freqs", or "" / "none" to disable
	VisualizerColor    string  // color of the visualizer (e.g. "white", "#FFD700")
	VisualizerHeight   float64 // height as a fraction of video height (e.g. 0.15)
	VisualizerPosition string  // "top" or "bottom"
	VisualizerOpacity  float64 // 0.0–1.0
	VisualizerMode     string  // waveform mode: "line", "point", "p2p", "cline"
	VisualizerScale    string  // amplitude/magnitude scale: lin, log, sqrt, cbrt (spectrum/freqs only)
	VisualizerSlide    string  // sliding mode: replace, scroll, fullframe, rscroll, lreplace (spectrum only)
	VisualizerStart    float64 // seconds from start when visualizer appears; negative = use default (10s)
	VisualizerEnd      float64 // seconds from start when visualizer disappears; negative = Duration-10
	VisualizerFade     float64 // seconds to fade in/out; 0 = instant

	Framerate int // output frame rate (fps); 0 = FFmpeg default
}

// Render builds and executes the FFmpeg command to produce the video.
func Render(cfg Config) error {
	fontPath := findFont()
	filterComplex, err := buildFilterComplex(cfg, fontPath)
	if err != nil {
		return err
	}

	args := []string{"-y"}
	for _, p := range cfg.ImagePaths {
		args = append(args, "-loop", "1", "-i", p)
	}
	args = append(args,
		"-i", cfg.AudioPath,
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-map", fmt.Sprintf("%d:a", len(cfg.ImagePaths)),
	)
	if cfg.EnableCUDA {
		args = append(args, "-c:v", "h264_nvenc", "-preset", "p4", "-cq", "23", "-b:v", "0")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "23")
	}
	if cfg.Framerate > 0 {
		args = append(args, "-r", strconv.Itoa(cfg.Framerate))
	}
	args = append(args, "-c:a", "aac", "-b:a", "192k", "-shortest")
	if cfg.MaxLength > 0 {
		args = append(args, "-t", fmt.Sprintf("%.6f", cfg.MaxLength))
	}
	args = append(args, cfg.OutputPath)

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

func buildFilterComplex(cfg Config, fontPath string) (string, error) {
	w := cfg.Width
	h := cfg.Height
	dim := 1.0 - cfg.BgDim

	bgSource, err := buildBgSource(cfg, w, h, dim)
	if err != nil {
		return "", err
	}

	highlightSize := int(float64(cfg.FontSize) * 1.2)
	contextSize := cfg.FontSize
	lineHeight := int(float64(highlightSize) * 1.6)

	var chain []string

	// 1. Lyrics drawtexts
	for i, line := range cfg.Lines {
		dt := buildDrawtext(fontPath, highlightSize, cfg.HighlightColor, "1.0",
			"(w-text_w)/2", fmt.Sprintf("(h*%.4f)-(%d/2)", cfg.LyricVPosition, highlightSize),
			line.Text, line.StartTime, line.EndTime, cfg.LyricFade, cfg.LyricFadeStyle)
		chain = append(chain, dt)

		for offset := -2; offset <= 2; offset++ {
			if offset == 0 {
				continue
			}
			ctxIdx := i + offset
			if ctxIdx < 0 || ctxIdx >= len(cfg.Lines) {
				continue
			}
			yExpr := fmt.Sprintf("(h*%.4f)-(%d/2)+(%d)", cfg.LyricVPosition, highlightSize, offset*lineHeight)
			opacity := "0.7"
			if offset == -2 || offset == 2 {
				opacity = "0.4"
			}
			dt := buildDrawtext(fontPath, contextSize, cfg.FontColor, opacity,
				"(w-text_w)/2", yExpr,
				cfg.Lines[ctxIdx].Text, line.StartTime, line.EndTime, cfg.LyricFade, cfg.LyricFadeStyle)
			chain = append(chain, dt)
		}
	}

	// 2. Fade-in (drawn after lyrics so titles emerge from black alongside the video)
	if cfg.FadeInSeconds > 0 {
		chain = append(chain, fmt.Sprintf(
			"fade=type=in:start_time=0:duration=%.3f", cfg.FadeInSeconds))
	}

	// 3. Fade-in title (drawn on already-faded-in video)
	if cfg.FadeInSeconds > 0 && cfg.FadeInTitle != "" {
		enableEnd := cfg.FadeInSeconds + cfg.FadeInTitleFadeOut
		var alphaExpr string
		if cfg.FadeInTitleFadeOut > 0 {
			alphaExpr = fmt.Sprintf(
				"if(lt(t,%.3f),1,max(0,(%.3f-t)/%.3f))",
				cfg.FadeInSeconds, enableEnd, cfg.FadeInTitleFadeOut)
		} else {
			alphaExpr = "1"
		}
		dts := buildTitleDrawtexts(fontPath, strings.Split(cfg.FadeInTitle, "|"),
			cfg.FadeInFontSizes, cfg.FadeInFontColors, 0, enableEnd, alphaExpr)
		chain = append(chain, dts...)
	}

	// 4. Fade-out (darkens lyrics and background, but not the fade-out title)
	if cfg.FadeOutSeconds > 0 {
		start := cfg.Duration - cfg.FadeOutSeconds
		if start < 0 {
			start = 0
		}
		chain = append(chain, fmt.Sprintf(
			"fade=type=out:start_time=%.3f:duration=%.3f", start, cfg.FadeOutSeconds))
	}

	// 5. Fade-out title (drawn after fade-out so it is not dimmed; uses its own alpha via --fade-out-title-fade-out)
	if cfg.FadeOutSeconds > 0 && cfg.FadeOutTitle != "" {
		fadeOutStart := cfg.Duration - cfg.FadeOutSeconds
		enableStart := fadeOutStart - cfg.FadeOutTitleFadeOut
		if enableStart < 0 {
			enableStart = 0
		}
		var alphaExpr string
		if cfg.FadeOutTitleFadeOut > 0 {
			alphaExpr = fmt.Sprintf(
				"if(lt(t,%.3f),max(0,(t-%.3f)/%.3f),1)",
				fadeOutStart, enableStart, cfg.FadeOutTitleFadeOut)
		} else {
			alphaExpr = "1"
		}
		dts := buildTitleDrawtexts(fontPath, strings.Split(cfg.FadeOutTitle, "|"),
			cfg.FadeOutFontSizes, cfg.FadeOutFontColors, enableStart, cfg.Duration, alphaExpr)
		chain = append(chain, dts...)
	}

	if len(chain) == 0 {
		chain = []string{"null"}
	}

	// 6. Audio visualizer overlay
	// When CUDA encoding is active, explicitly upload the finished CPU frame to
	// CUDA VRAM before h264_nvenc touches it. This lets the encoder read directly
	// from device memory instead of doing an internal pinned-memory copy.
	finalLabel := "out"
	hwuploadStep := ""
	if cfg.EnableCUDA {
		finalLabel = "pre_enc"
		hwuploadStep = ";\n[pre_enc]format=yuv420p,hwupload_cuda[out]"
	}

	visFilter, err := buildVisualizerFilter(cfg, len(cfg.ImagePaths), finalLabel)
	if err != nil {
		return "", err
	}

	var drawtextSection string
	if visFilter != "" {
		drawtextSection = "[bg]\n" + strings.Join(chain, ",\n") + "\n[pre_vis];\n" + visFilter + hwuploadStep
	} else {
		drawtextSection = "[bg]\n" + strings.Join(chain, ",\n") + "\n[" + finalLabel + "]" + hwuploadStep
	}
	return bgSource + ";\n" + drawtextSection, nil
}

// buildVisualizerFilter returns the FFmpeg filter fragment for the audio visualizer overlay,
// or an empty string if the visualizer is disabled.
// audioIdx is the FFmpeg input index of the audio stream (= number of image inputs).
// outputLabel is the filtergraph label to assign the final output stream (without brackets).
func buildVisualizerFilter(cfg Config, audioIdx int, outputLabel string) (string, error) {
	if cfg.VisualizerType == "" || cfg.VisualizerType == "none" {
		return "", nil
	}

	visH := int(float64(cfg.Height) * cfg.VisualizerHeight)
	if visH <= 0 {
		visH = int(float64(cfg.Height) * 0.15)
	}

	color := cfg.VisualizerColor
	if color == "" {
		color = "white"
	}

	var vizChain string
	switch cfg.VisualizerType {
	case "waveform":
		mode := cfg.VisualizerMode
		if mode == "" {
			mode = "line"
		}
		vizChain = fmt.Sprintf("[%d:a]showwaves=s=%dx%d:mode=%s:rate=30:colors=%s[vis_raw]",
			audioIdx, cfg.Width, visH, mode, color)
	case "spectrum":
		// showspectrum uses `color` as an enum scheme, not a free-form color;
		// `colors` is not a valid option and will cause an FFmpeg error.
		scale := cfg.VisualizerScale
		if scale == "" {
			scale = "log"
		}
		slide := cfg.VisualizerSlide
		if slide == "" {
			slide = "scroll"
		}
		vizChain = fmt.Sprintf("[%d:a]showspectrum=s=%dx%d:color=channel:scale=%s:slide=%s[vis_raw]",
			audioIdx, cfg.Width, visH, scale, slide)
	case "freqs":
		scale := cfg.VisualizerScale
		if scale == "" {
			scale = "log"
		}
		vizChain = fmt.Sprintf("[%d:a]showfreqs=s=%dx%d:mode=bar:ascale=%s:colors=%s[vis_raw]",
			audioIdx, cfg.Width, visH, scale, color)
	default:
		return "", fmt.Errorf("unknown visualizer type %q; valid values: waveform, spectrum, freqs", cfg.VisualizerType)
	}

	opacity := cfg.VisualizerOpacity
	if opacity <= 0 {
		opacity = 0.8
	}

	// Resolve start/end times; negative values use defaults (10s from each end).
	visStart := cfg.VisualizerStart
	if visStart < 0 {
		visStart = 10
	}
	visEnd := cfg.VisualizerEnd
	if visEnd < 0 {
		visEnd = cfg.Duration - 10
		if visEnd < visStart {
			visEnd = visStart
		}
	}

	// Key out the black background produced by the visualizer filters, then apply opacity.
	alphaChain := fmt.Sprintf("[vis_raw]colorkey=color=black:similarity=0.05:blend=0.1,format=rgba,colorchannelmixer=aa=%.3f[vis_colored]", opacity)

	// Optionally fade the visualizer in/out at its start and end times.
	visLabel := "vis_colored"
	var fadePart string
	if cfg.VisualizerFade > 0 {
		fadeOutStart := visEnd - cfg.VisualizerFade
		if fadeOutStart < visStart {
			fadeOutStart = visStart
		}
		fadePart = fmt.Sprintf(
			"[vis_colored]fade=type=in:start_time=%.3f:duration=%.3f:alpha=1,fade=type=out:start_time=%.3f:duration=%.3f:alpha=1[vis_faded]",
			visStart, cfg.VisualizerFade, fadeOutStart, cfg.VisualizerFade,
		)
		visLabel = "vis_faded"
	}

	var yExpr string
	if cfg.VisualizerPosition == "top" {
		yExpr = "0"
	} else {
		yExpr = fmt.Sprintf("%d", cfg.Height-visH)
	}

	overlayChain := fmt.Sprintf(
		"[pre_vis][%s]overlay=x=0:y=%s:format=auto:enable='between(t,%.3f,%.3f)'[%s]",
		visLabel, yExpr, visStart, visEnd, outputLabel,
	)

	parts := []string{vizChain, alphaChain}
	if fadePart != "" {
		parts = append(parts, fadePart)
	}
	parts = append(parts, overlayChain)
	return strings.Join(parts, ";\n"), nil
}

// buildTitleDrawtexts produces one drawtext filter per title line, centered as a block.
// alphaExpr is an FFmpeg expression for per-frame alpha (can be "1" for constant).
func buildTitleDrawtexts(fontPath string, lines []string, fontSizes []int, colors []string,
	enableStart, enableEnd float64, alphaExpr string) []string {

	if len(lines) == 0 {
		return nil
	}

	// Resolve per-line font sizes; last provided size is the fallback.
	sizes := make([]int, len(lines))
	for i := range lines {
		switch {
		case i < len(fontSizes):
			sizes[i] = fontSizes[i]
		case len(fontSizes) > 0:
			sizes[i] = fontSizes[len(fontSizes)-1]
		default:
			sizes[i] = 60
		}
	}

	// Resolve per-line colors; last provided color is the fallback.
	lineColors := make([]string, len(lines))
	for i := range lines {
		switch {
		case i < len(colors):
			lineColors[i] = colors[i]
		case len(colors) > 0:
			lineColors[i] = colors[len(colors)-1]
		default:
			lineColors[i] = "#FFFFFF"
		}
	}

	// Compute line heights and total block height for vertical centering.
	lineHeights := make([]int, len(lines))
	totalH := 0
	for i, sz := range sizes {
		lineHeights[i] = int(float64(sz) * 1.4)
		totalH += lineHeights[i]
	}

	fontSpec := ""
	if fontPath != "" {
		fontSpec = fmt.Sprintf("fontfile='%s':", escapeDrawtextValue(fontPath))
	}

	var result []string
	yOffset := 0
	for i, line := range lines {
		yExpr := fmt.Sprintf("(h-%d)/2+%d", totalH, yOffset)
		escapedText := escapeDrawtext(line)
		dt := fmt.Sprintf(
			"drawtext=%sfontsize=%d:fontcolor=%s:x=(w-text_w)/2:y=%s:text='%s':enable='between(t,%.3f,%.3f)':alpha='%s'",
			fontSpec, sizes[i], lineColors[i], yExpr, escapedText, enableStart, enableEnd, alphaExpr,
		)
		result = append(result, dt)
		yOffset += lineHeights[i]
	}
	return result
}

func buildBgSource(cfg Config, w, h int, dim float64) (string, error) {
	fps := cfg.Framerate
	if fps <= 0 {
		fps = 25
	}
	switch len(cfg.ImagePaths) {
	case 0:
		return fmt.Sprintf(
			"color=c=black:s=%dx%d:r=%d,format=yuv420p,colorchannelmixer=rr=%.2f:gg=%.2f:bb=%.2f[bg]",
			w, h, fps, dim, dim, dim,
		), nil
	case 1:
		if len(cfg.DriftTypes) == 0 {
			return fmt.Sprintf(
				"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,"+
					"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,"+
					"format=yuv420p,"+
					"colorchannelmixer=rr=%.2f:gg=%.2f:bb=%.2f[bg]",
				w, h, w, h, dim, dim, dim,
			), nil
		}
		dp := cfg.DriftMin + rand.Intn(cfg.DriftMax-cfg.DriftMin+1)
		return fmt.Sprintf(
			"[0:v]%s,format=yuv420p,colorchannelmixer=rr=%.2f:gg=%.2f:bb=%.2f[bg]",
			buildDriftChain(pickDriftType(cfg.DriftTypes), w, h, dp, cfg.Duration, cfg.DriftDurationPct, cfg.DriftEasing),
			dim, dim, dim,
		), nil
	default:
		n := len(cfg.ImagePaths)
		t := cfg.Transition
		td := 0.0
		if t != "" && t != "none" {
			td = cfg.TransitionDuration
		}
		// Each image must be longer than its nominal share to compensate for xfade
		// overlaps: total_output = n*perImage - (n-1)*td, so solve for perImage
		// such that total_output == cfg.Duration.
		perImage := (cfg.Duration + float64(n-1)*td) / float64(n)

		var parts []string
		for i := range cfg.ImagePaths {
			var scalePad string
			if len(cfg.DriftTypes) == 0 {
				scalePad = fmt.Sprintf(
					"scale=%d:%d:force_original_aspect_ratio=decrease,"+
						"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black",
					w, h, w, h,
				)
			} else {
				dp := cfg.DriftMin + rand.Intn(cfg.DriftMax-cfg.DriftMin+1)
				scalePad = buildDriftChain(pickDriftType(cfg.DriftTypes), w, h, dp, perImage, cfg.DriftDurationPct, cfg.DriftEasing)
			}
			parts = append(parts, fmt.Sprintf(
				"[%d:v]%s,"+
					"format=yuv420p,"+
					"setsar=1,"+
					"colorchannelmixer=rr=%.2f:gg=%.2f:bb=%.2f,"+
					"trim=duration=%.6f,"+
					"setpts=PTS-STARTPTS[slid%d]",
				i, scalePad, dim, dim, dim, perImage, i,
			))
		}

		if t == "" || t == "none" {
			concatInputs := ""
			for i := range cfg.ImagePaths {
				concatInputs += fmt.Sprintf("[slid%d]", i)
			}
			parts = append(parts, fmt.Sprintf("%sconcat=n=%d:v=1:a=0[bg]", concatInputs, n))
			return strings.Join(parts, ";\n"), nil
		}

		if td >= perImage {
			return "", fmt.Errorf("transition-duration (%.1fs) must be less than per-image duration (%.1fs)", td, perImage)
		}

		// Chain xfade filters: each offset is (i+1)*(perImage-td) relative to the
		// accumulated output of the previous xfade.
		prev := "[slid0]"
		for i := 0; i < n-1; i++ {
			outLabel := fmt.Sprintf("[xf%d]", i)
			if i == n-2 {
				outLabel = "[bg]"
			}
			offset := float64(i+1) * (perImage - td)
			parts = append(parts, fmt.Sprintf(
				"%s[slid%d]xfade=transition=%s:duration=%.6f:offset=%.6f%s",
				prev, i+1, t, td, offset, outLabel,
			))
			prev = outLabel
		}
		return strings.Join(parts, ";\n"), nil
	}
}

// driftConcreteTypes lists all valid non-random drift animation types in a fixed order.
var driftConcreteTypes = []string{"left", "right", "up", "down", "zoom-in", "zoom-out"}

// pickDriftType selects one drift type randomly from the provided list.
// "random" is expanded to all concrete types. Returns "" if the list is empty.
func pickDriftType(types []string) string {
	if len(types) == 0 {
		return ""
	}
	seen := make(map[string]bool)
	for _, t := range types {
		if t == "random" {
			for _, ct := range driftConcreteTypes {
				seen[ct] = true
			}
		} else {
			seen[t] = true
		}
	}
	pool := make([]string, 0, len(seen))
	for _, ct := range driftConcreteTypes {
		if seen[ct] {
			pool = append(pool, ct)
		}
	}
	if len(pool) == 0 {
		return ""
	}
	return pool[rand.Intn(len(pool))]
}

// buildEasingExpr returns an FFmpeg expression that applies the chosen easing curve
// to the normalised progress variable p (already clamped to [0,1]).
// Curves:
//
//	linear : constant speed
//	quad   : ease-out quadratic  p*(2-p)           — decelerates to rest (default)
//	cubic  : ease-out cubic       p*(3+p*(p-3))     — stronger deceleration
//	smooth : ease-in-out (smoothstep) p²*(3-2p)     — gradual start and end
func buildEasingExpr(easing, p string) string {
	switch easing {
	case "linear":
		return p
	case "cubic":
		// ease-out-cubic: 1-(1-p)^3 = p*(3+p*(p-3))
		return fmt.Sprintf("(%[1]s)*(3+(%[1]s)*((%[1]s)-3))", p)
	case "smooth":
		// smoothstep: p²*(3-2p)
		return fmt.Sprintf("(%[1]s)*(%[1]s)*(3-2*(%[1]s))", p)
	default: // "quad"
		// ease-out-quad: p*(2-p)
		return fmt.Sprintf("(%[1]s)*(2-(%[1]s))", p)
	}
}

// buildDriftChain returns the FFmpeg filter fragment (scale+crop[+scale]) that
// animates the background image with the given drift type.
// dp is the drift distance in pixels (pan offset or zoom margin).
// driftDurPct controls what percentage of dur is used for the animation;
// after that the frame holds at the final position.
func buildDriftChain(driftType string, w, h, dp int, dur, driftDurPct float64, easing string) string {
	driftDur := dur * driftDurPct / 100.0
	if driftDur <= 0 {
		driftDur = dur
	}
	// min(1,t/T) rewritten without commas (FFmpeg filtergraph splits on ','):
	// min(1,r) == (1 + r - abs(1 - r)) / 2  where r = t/T
	p := fmt.Sprintf("((1+t/%.3f-abs(1-t/%.3f))/2)", driftDur, driftDur)
	e := buildEasingExpr(easing, p)
	forward := fmt.Sprintf("%d*(%s)", dp, e)
	reverse := fmt.Sprintf("%d*(1-(%s))", dp, e)

	switch driftType {
	case "left":
		// pan left: x 0→dp (content drifts left)
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d:%d:%s:(ih-%d)/2",
			w+dp, h+dp, w, h, forward, h,
		)
	case "right":
		// pan right: x dp→0 (content drifts right)
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d:%d:%s:(ih-%d)/2",
			w+dp, h+dp, w, h, reverse, h,
		)
	case "up":
		// pan up: y 0→dp (content drifts up)
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d:%d:(iw-%d)/2:%s",
			w+dp, h+dp, w, h, w, forward,
		)
	case "down":
		// pan down: y dp→0 (content drifts down)
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d:%d:(iw-%d)/2:%s",
			w+dp, h+dp, w, h, w, reverse,
		)
	case "zoom-in":
		// zoom in: crop shrinks W+2dp→W (image enlarges), then output scaled to W:H
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d+2*%d*(1-(%s)):%d+2*%d*(1-(%s)):(iw-ow)/2:(ih-oh)/2,"+
				"scale=%d:%d",
			w+dp*2, h+dp*2,
			w, dp, e, h, dp, e,
			w, h,
		)
	case "zoom-out":
		// zoom out: crop grows W→W+2dp (image shrinks), then output scaled to W:H
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d+2*%d*(%s):%d+2*%d*(%s):(iw-ow)/2:(ih-oh)/2,"+
				"scale=%d:%d",
			w+dp*2, h+dp*2,
			w, dp, e, h, dp, e,
			w, h,
		)
	default:
		// Fallback: scale+pad (no drift)
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=decrease,"+
				"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black",
			w, h, w, h,
		)
	}
}

func buildDrawtext(fontPath string, fontSize int, color, opacity, x, y, text string, start, end, lyricFade float64, lyricFadeStyle string) string {
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

	if lyricFade > 0 {
		// Clamp fade to half the line duration so both ramps fit
		lineDur := end - start
		fade := lyricFade
		if lineDur < 2*fade {
			fade = lineDur / 2
		}
		enableStart := start - fade
		if enableStart < 0 {
			enableStart = 0
		}
		// Alpha ramps: fade-in during [enableStart, start], hold, fade-out during [end-fade, end]
		fadeOutStart := end - fade

		var fadeInExpr, fadeOutExpr string
		switch lyricFadeStyle {
		case "smooth":
			// Cubic smoothstep: p*p*(3-2*p) where p is progress 0→1
			pin := fmt.Sprintf("((t-%.3f)/%.3f)", enableStart, fade)
			fadeInExpr = fmt.Sprintf("%s*%s*(3-2*%s)", pin, pin, pin)
			pout := fmt.Sprintf("((%.3f-t)/%.3f)", end, fade)
			fadeOutExpr = fmt.Sprintf("%s*%s*(3-2*%s)", pout, pout, pout)
		default: // "linear"
			fadeInExpr = fmt.Sprintf("(t-%.3f)/%.3f", enableStart, fade)
			fadeOutExpr = fmt.Sprintf("(%.3f-t)/%.3f", end, fade)
		}

		alphaExpr := fmt.Sprintf(
			"if(lt(t,%.3f),%s,if(gt(t,%.3f),%s,1))",
			start, fadeInExpr, fadeOutStart, fadeOutExpr,
		)
		return fmt.Sprintf(
			"drawtext=%sfontsize=%d:fontcolor=%s:x=%s:y=%s:text='%s':enable='between(t,%.3f,%.3f)':alpha='%s'",
			fontSpec, fontSize, colorSpec, x, y, escapedText, enableStart, end, alphaExpr,
		)
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
