# lyricvid

Generate a music video with synchronized lyrics from an MP3, a lyrics file, and a background image.

## Prerequisites

- **Go 1.22+** for building from source
- **FFmpeg** and **ffprobe** must be installed and available in your PATH
  - Install from [https://ffmpeg.org/download.html](https://ffmpeg.org/download.html)
  - On Ubuntu/Debian: `sudo apt install ffmpeg`
  - On macOS: `brew install ffmpeg`
  - On Windows: download from the FFmpeg website

## Installation

### From source

```bash
git clone https://github.com/user/lyricvid.git
cd lyricvid
go build -o lyricvid .
```

### With go install

```bash
go install github.com/user/lyricvid@latest
```

## Usage

```bash
lyricvid generate \
  --audio    song.mp3 \
  --lyrics   song.lrc \
  --image    cover.jpg \
  --output   output.mp4
```

The `generate` subcommand is also the default, so you can omit it:

```bash
lyricvid \
  --audio song.mp3 \
  --lyrics song.lrc \
  --image cover.jpg
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--audio` | *(required)* | Path to audio file (.mp3, .m4a, .flac, .wav) |
| `--lyrics` | *(required)* | Path to lyrics file (.lrc or .txt) |
| `--image` | *(required)* | Path to background image (.jpg, .jpeg, .png, .webp) |
| `--output` | `output.mp4` | Output video file path |
| `--width` | `1920` | Video width in pixels |
| `--height` | `1080` | Video height in pixels |
| `--font-size` | `64` | Base font size for lyrics text |
| `--font-color` | `#FFFFFF` | Color for context lyric lines |
| `--highlight-color` | `#FFD700` | Color for the active lyric line |
| `--bg-dim` | `0.4` | Background dimming (0.0 = fully black, 1.0 = no dimming) |
| `--lyric-fade` | `0.3` | Seconds to cross-fade between lyric lines; 0 = hard cut |
| `--lyric-fade-style` | `linear` | Alpha curve for lyric cross-fade: `linear` or `smooth` |

### Example with all options

```bash
lyricvid generate \
  --audio    song.mp3 \
  --lyrics   song.lrc \
  --image    cover.jpg \
  --output   my_video.mp4 \
  --width    1280 \
  --height   720 \
  --font-size 48 \
  --font-color "#FFFFFF" \
  --highlight-color "#FF6600" \
  --bg-dim   0.5
```

## Lyrics File Formats

### LRC format (timestamped)

LRC files contain timestamps for each lyric line, enabling precise synchronization:

```
[00:12.34] First line of lyrics
[00:16.00] Second line of lyrics
[00:20.50] Third line of lyrics
[02:30.00] Last line
```

Timestamps use the format `[mm:ss.xx]` or `[mm:ss.xxx]`. The video will:
- Display the current lyric line in the highlight color with a larger font
- Show up to 2 lines before and after as context in a dimmer color
- Transition between lines at each timestamp

LRC metadata tags (like `[ti:Title]`, `[ar:Artist]`) are automatically skipped.

### Plain text format (no timestamps)

If your lyrics file has no LRC timestamps, simply put each line of lyrics on its own line:

```
First line of lyrics
Second line of lyrics
Third line of lyrics
Last line
```

The lyrics will be distributed evenly across the audio duration. Each line gets an equal share of the total time.

## How it works

1. The audio file is analyzed with `ffprobe` to determine its exact duration
2. The lyrics file is parsed, auto-detecting LRC vs plain text format
3. An FFmpeg command is built with a complex filtergraph that:
   - Scales the background image to the target resolution (maintaining aspect ratio)
   - Dims the background by the specified factor
   - Overlays `drawtext` filters for each lyric line with time-based enable/disable
4. FFmpeg encodes the output as H.264 video with AAC audio
5. Progress is shown in real-time on the terminal

## Output format

- Video: H.264, CRF 23, medium preset
- Audio: AAC, 192 kbps
- Container: MP4
