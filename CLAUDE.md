# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`lyricvid` is a Go CLI tool that generates MP4 music videos with synchronized lyrics overlays, driven entirely by FFmpeg under the hood. It requires FFmpeg and ffprobe to be available in PATH.

## Commands

```bash
# Build
go build -o lyricvid .

# Run directly (generate is also the default subcommand)
go run main.go --audio song.mp3 --lyrics song.lrc --image cover.jpg

# Vet
go vet ./...
```

There is no automated test suite.

## Architecture

All orchestration lives in `main.go` (CLI setup, flag parsing, input validation, pipeline sequencing). The three `internal/` packages are fully decoupled from each other — they never import one another. `main.go` wires them together.

**Pipeline**:
1. `audio.CheckFFmpeg()` — verify ffmpeg/ffprobe are in PATH
2. `audio.GetDuration()` — calls `ffprobe` to get audio length in seconds
3. `lyrics.ParseFile()` — auto-detects LRC vs plain text; returns `[]lyrics.Line`
4. `lyrics.SetEndTimes()` or `lyrics.DistributeEvenly()` — assigns end timestamps
5. `video.Render()` — builds an FFmpeg filtergraph, runs FFmpeg, monitors stderr for progress

**Key types**:
- `lyrics.Line{StartTime, EndTime float64; Text string}` — the unit passed from lyrics → video
- `video.Config` — flat struct holding all rendering parameters plus `[]lyrics.Line`
  - `Framerate int` — output frame rate (fps); 0 means FFmpeg default (currently 25 for the `color` source). When set, `-r <fps>` is passed to FFmpeg and the `color` source rate is set accordingly.

**Filtergraph structure** (`internal/video/renderer.go:buildFilterComplex`):
- Base chain: scale → pad → format=yuv420p → `colorchannelmixer` (for bg dim) → `[bg]`
- For each lyric line `i`, generates multiple `drawtext` filters: one active/highlighted line at center + up to 4 context lines (±1 at 70% opacity, ±2 at 40% opacity), all gated by `enable='between(t,start,end)'`

## FFmpeg escaping — critical gotcha

There are two distinct escape functions for the `drawtext` filter:

- `escapeDrawtext(text)` — for lyric content: escapes `\`, replaces `'` with Unicode right-quote (U+2019), escapes `:`, `%`, `[`, `]`, `;`
- `escapeDrawtextValue(text)` — for font file paths: escapes `\`, `'`, `:`

**Apostrophes are handled differently in each**: lyric text swaps them for a Unicode lookalike (avoiding shell quoting issues inside single-quoted FFmpeg values); font paths escape them with a backslash. Do not swap these functions or mix their logic.

**Backslash must always be escaped first** before any other replacements.

## Conventions

- Errors: `fmt.Errorf("doing X %q: %w", val, err)` — always wrap with context
- Progress messages go to stdout via `fmt.Printf` ("Analyzing…", "Parsing…", "Rendering…")
- FFmpeg stderr is captured silently; only shown to the user on failure
- Each `internal/` sub-package is a single file — split only when genuinely needed
