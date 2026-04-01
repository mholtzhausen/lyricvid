# lyricvid – Project Guidelines

Go CLI tool that generates MP4 lyric videos from an MP3, an LRC/text lyrics file, and a background image. Delegates all encoding to FFmpeg via `exec.Command`.

## Build and Test

```bash
# Build
go build -o lyricvid .

# Run directly
go run main.go --audio song.mp3 --lyrics song.lrc --image cover.jpg

# Vet / lint
go vet ./...
```

No automated test suite exists yet. Manual testing requires FFmpeg + ffprobe in PATH.

## Architecture

```
main.go               CLI (Cobra), input validation, orchestration
internal/audio/       GetDuration() via ffprobe; CheckFFmpeg() PATH check
internal/lyrics/      ParseFile() auto-detects LRC vs plain text; Line{StartTime,EndTime,Text}
internal/video/       Render() builds FFmpeg filter_complex and runs ffmpeg
```

- All input validation happens in `main.go` before any sub-package is called.
- Each `internal/` sub-package is a **single file**; keep it that way unless the file genuinely needs splitting.
- Sub-packages have no inter-dependencies — only `main.go` imports them all.

## Conventions

- **Errors**: wrap with context using `fmt.Errorf("doing X %q: %w", val, err)`.
- **Progress**: `fmt.Printf` to stdout during workflow ("Analyzing…", "Parsing…", "Rendering…"). FFmpeg stderr is captured and returned only on failure.
- **Flags**: declared as package-level vars in `main.go`; registered on both root command and the explicit `generate` sub-command via `addFlags(cmd)`.
- **Platform font paths**: `runtime.GOOS` switch in `video.findFont()`; add new paths there for additional OS support.

## FFmpeg Filter Escaping — Critical Pitfall

Two distinct escape functions exist in `internal/video/renderer.go`:

| Function | Use | Escapes |
|---|---|---|
| `escapeDrawtext(text)` | Lyric text content | `\`, `'`→`\u2019`, `:`, `%`, `[`, `]`, `;` |
| `escapeDrawtextValue(text)` | Font file paths | `\`, `'`, `:` only |

**Always use the correct function.** Using `escapeDrawtext` on a font path will corrupt it. Order matters: backslash must be escaped first.

## External Requirements

FFmpeg and ffprobe must be on PATH. `audio.CheckFFmpeg()` is called early in `runGenerate()` and surfaces a helpful error with the download URL if missing. Do not remove this check.
