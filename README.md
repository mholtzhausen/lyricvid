# lyricvid

Turn an MP3 into a proper lyrics video — synchronized, animated, and looking like you actually put effort in. Built on FFmpeg, driven by a config file, optionally powered by Google Gemini AI for background image generation.

## Prerequisites

- **Go 1.22+** to build from source
- **FFmpeg + ffprobe** in your PATH
  - Ubuntu/Debian: `sudo apt install ffmpeg`
  - macOS: `brew install ffmpeg`
  - Windows: grab it from [ffmpeg.org](https://ffmpeg.org/download.html)

## Installation

```bash
git clone https://github.com/user/lyricvid.git
cd lyricvid
go build -o lyricvid .
```

Or if you just want the binary:

```bash
go install github.com/user/lyricvid@latest
```

---

## The Easiest Way to Get Started

Drop your MP3 in its own folder and `cd` into it. That's your project home — everything else lives alongside it.

```bash
mkdir my-song && cp song.mp3 my-song/ && cd my-song
```

Now run `init` to scaffold the whole project in one go:

```bash
lyricvid init song.mp3
```

This creates:
- `images/` — a folder where you'll put your background images
- `images.yml` — a pre-filled config file with sensible defaults and fade-in/out already wired up
- `song.lrc` — an empty LRC lyrics file ready to fill in

Then the tool prints step-by-step instructions for what to do next. It's genuinely pretty helpful.

You can also give the project a custom name (useful if you're managing multiple videos from one folder):

```bash
lyricvid init song.mp3 cinematic
# creates: cinematic/, cinematic.yml, song.lrc
```

Once your LRC is filled in and you've got some images in the folder, render:

```bash
lyricvid song.mp3
```

That's it. Output lands at `song.mp4`.

---

## Lyrics Files

### LRC (timestamped) — the good stuff

```
[00:12.34] First line of lyrics
[00:16.00] Second line
[00:20.50] Third line
[02:30.00] Last line
```

Each line lights up exactly when the timestamp hits. The active line gets the highlight color and a larger size; the two lines before and after sit in a dimmer context color. Metadata tags like `[ti:Title]` are skipped automatically.

### Plain text — no timestamps, no problem

```
First line of lyrics
Second line
Third line
Last line
```

lyricvid divides the audio duration evenly between lines. Not as precise, but fine for rough cuts.

---

## All the Commands

### `generate` (or just run the binary directly)

The main event. Renders your video.

```bash
lyricvid song.mp3
lyricvid generate song.mp3
lyricvid generate song.mp3 --quality 1080p --drift zoom-in,right --fade-in-seconds 4
```

Picks up `lyricvid.yml` and `<song-stem>.yml` config files from the same directory automatically.

---

### `init`

Scaffold a new project. Covered above — use this first.

```bash
lyricvid init song.mp3
lyricvid init song.mp3 my-project-name
```

---

### `config-file`

Writes a fully-commented YAML template with every single option and its default. Great starting point for a config file if you don't want to use `init`.

```bash
lyricvid config-file
lyricvid config-file ./song.yml
```

If the file already exists, the command reads it first and carries all current values over into the fresh template — keeping your settings while picking up any new fields or updated comments added in later versions of lyricvid.

---

### `save`

Renders nothing — just saves your CLI flags to a config file. Useful for locking in settings you've been tweaking on the command line.

```bash
lyricvid save song.mp3 --quality 720p --drift left,zoom-in --bg-dim 0.5
```

Merges with any existing config file (preserves comments and existing values). Only writes what you explicitly passed.

---

### `image-gen`

Generate background images from your lyrics using Google Gemini. Gives you scene images that actually match the song — or at least something more interesting than a stock photo.

```bash
lyricvid image-gen song.mp3
lyricvid image-gen song.mp3 --count 8 --quality 1080p --style "cinematic film noir, high contrast"
lyricvid image-gen song.mp3 my-images/ --aspect-ratio 16:9
```

Reads the lyrics file (auto-detected alongside the MP3) for scene inspiration. First time you run it, pass `--api-key` once and it'll be saved for all future runs.

```bash
lyricvid image-gen song.mp3 --api-key YOUR_GEMINI_API_KEY
```

---

### `set-gemini`

Store your Gemini API key without running image-gen. Saves it encrypted to `~/.lyricvid.yaml`.

```bash
lyricvid set-gemini YOUR_GEMINI_API_KEY
```

---

### `ai`

An interactive chat assistant pre-loaded with the full lyricvid help as context. Useful when you can't remember which flag does what or want to figure out a filtergraph setting.

```bash
lyricvid ai
lyricvid ai --api-key YOUR_GEMINI_API_KEY
```

---

## Config Files

lyricvid looks for YAML config files next to the audio file, in this order:

1. `lyricvid.yml` — applies to every song in the folder
2. `<audio-stem>.yml` — per-song overrides
3. `--config <path>` — explicit override from the CLI
4. CLI flags — always win

Values in later files override earlier ones. CLI flags always take highest priority.

**Example `song.yml`:**

```yaml
image: images
quality: 1080p
aspect-ratio: "16:9"

font-color: "#FFFFFF"
highlight-color: "#FFD700"
bg-dim: 0.4

fade-in-seconds: 5
fade-in-title: "Song Title|Artist Name"
fade-in-font-size: "80|60"
fade-in-font-color: "#FFFFFF"

fade-out-seconds: 5
fade-out-title: "made with lyricvid"
fade-out-font-color: "#FF0000"

drift: left,right,zoom-in
drift-easing: cubic

visualizer-type: waveform
visualizer-color: "#FFD700"
visualizer-height: 0.15
visualizer-position: bottom
visualizer-opacity: 0.8

enable-cuda: true
framerate: 30
```

Relative image paths in config files are resolved from the audio file's directory, not wherever you ran the command from.

---

## Flag Reference

### Input / Output

| Flag | Default | Description |
|------|---------|-------------|
| `--lyrics` | auto-detected | Path to `.lrc` or `.txt` lyrics file; auto-detected from audio directory if omitted |
| `--image` | auto-detected | Path to background image or folder; looks in `<audio-dir>/images/` by default |
| `--output` | `<stem>.mp4` | Output path; defaults to same folder and name as the audio file |
| `--config` | — | Path to a YAML config file; overrides auto-detected `lyricvid.yml` / `<stem>.yml` |

### Quality & Dimensions

| Flag | Default | Description |
|------|---------|-------------|
| `--quality` | — | Preset: `480p`, `720p`, `1080p`, `1440p`; overrides `--width`/`--height` |
| `--aspect-ratio` | `16:9` | Used with `--quality` to compute final dimensions (e.g. `4:3`, `21:9`, `1:1`) |
| `--width` | `1920` | Video width in pixels (ignored when `--quality` is set) |
| `--height` | `1080` | Video height in pixels (ignored when `--quality` is set) |

### Typography

| Flag | Default | Description |
|------|---------|-------------|
| `--font-size` | auto-scaled | Base font size; auto-scales proportionally to width if not set |
| `--font-size-reference` | `38` | Font size (pt) at the 1920px reference width, used for auto-scaling |
| `--font-color` | `#FFFFFF` | Color for context lyric lines |
| `--highlight-color` | `#FFD700` | Color for the active lyric line |

Colors accept hex (`#FF6600`) or FFmpeg named colors (`white`, `yellow`, etc).

### Background & Transitions

| Flag | Default | Description |
|------|---------|-------------|
| `--bg-dim` | `0.4` | Background dimming (0.0 = pure black, 1.0 = original image, no dimming) |
| `--transition` | `fade` | Transition effect between images; see below for options |
| `--transition-duration` | `3.0` | Duration of transition in seconds |

**Transition types:** `fade`, `fadeblack`, `fadewhite`, `dissolve`, `wipeleft`, `wiperight`, `wipeup`, `wipedown`, `slideleft`, `slideright`, `radial`, `pixelize`, `none`

### Lyric Positioning & Timing

| Flag | Default | Description |
|------|---------|-------------|
| `--lyric-position` | `0.65` | Vertical position of the active line (0.0 = top, 1.0 = bottom) |
| `--lyric-fade` | `0.3` | Cross-fade duration in seconds between lyric lines; `0` = hard cut |
| `--lyric-fade-style` | `linear` | Fade curve: `linear` or `smooth` |

### Fade-In

| Flag | Default | Description |
|------|---------|-------------|
| `--fade-in-seconds` | `0` | Seconds to fade in from black at the start |
| `--fade-in-title` | — | Title text during fade-in; use `\|` to separate lines (`"Song Title\|Artist"`) |
| `--fade-in-title-fade-out` | `1.0` | Seconds to fade the title out after the fade-in ends |
| `--fade-in-font-size` | `60` | Font size(s) for the fade-in title; use `\|` for per-line sizes (`"80\|60"`) |
| `--fade-in-font-color` | font-color | Color(s) for the fade-in title; use `\|` for per-line colors |

### Fade-Out

| Flag | Default | Description |
|------|---------|-------------|
| `--fade-out-seconds` | `0` | Seconds to fade to black at the end |
| `--fade-out-title` | — | Title text during fade-out; use `\|` for multiple lines |
| `--fade-out-title-fade-out` | `1.0` | Seconds to fade in the title before the fade-out begins |
| `--fade-out-font-size` | `60` | Font size(s) for the fade-out title |
| `--fade-out-font-color` | font-color | Color(s) for the fade-out title |

**Example with multi-line title and per-line colors:**

```bash
lyricvid song.mp3 \
  --fade-in-seconds 6 \
  --fade-in-title "My Song|By Some Band" \
  --fade-in-font-size "80|50" \
  --fade-in-font-color "#FFD700|#FFFFFF"
```

### Drift Animation

Slowly pans or zooms the background image while it's on screen. Subtle, but it makes a static image feel alive.

| Flag | Default | Description |
|------|---------|-------------|
| `--drift` | disabled | Comma-separated motion types: `left`, `right`, `up`, `down`, `zoom-in`, `zoom-out`, `random` |
| `--drift-max` | `60` | Maximum drift distance in pixels |
| `--drift-min` | `10` | Minimum drift distance in pixels |
| `--drift-duration-percentage` | `90` | % of each image's display time used for animation (holds at final position after) |
| `--drift-easing` | `quad` | Easing curve: `linear`, `quad`, `cubic`, `smooth` |

```bash
# Gentle zoom in with a slow horizontal drift
lyricvid song.mp3 --drift zoom-in,right --drift-easing cubic --drift-max 40

# Unpredictable, lively
lyricvid song.mp3 --drift random
```

### Audio Visualizer

Overlays a real-time audio visualization on the video — waveform, spectrum, or frequency bars.

| Flag | Default | Description |
|------|---------|-------------|
| `--visualizer-type` | `none` | Visualizer type: `waveform`, `spectrum`, `freqs`, or `none` to disable |
| `--visualizer-color` | `white` | Color of the visualizer; hex or FFmpeg named color |
| `--visualizer-height` | `0.15` | Height as a fraction of video height (e.g. `0.15` = bottom 15%) |
| `--visualizer-position` | `bottom` | Placement: `top` or `bottom` |
| `--visualizer-opacity` | `0.8` | Overlay opacity (0.0 = invisible, 1.0 = fully opaque) |
| `--visualizer-mode` | `line` | Waveform drawing mode (only applies to `waveform`): `line`, `point`, `p2p`, `cline` |

**Visualizer types:**
- `waveform` — animated amplitude waveform; `--visualizer-mode` controls the drawing style
- `spectrum` — scrolling frequency spectrogram; colors map to audio channels
- `freqs` — real-time frequency bar graph

```bash
# Subtle waveform at the bottom
lyricvid song.mp3 --visualizer-type waveform --visualizer-color "#FFD700"

# Full-width spectrum, top of frame, semi-transparent
lyricvid song.mp3 --visualizer-type spectrum --visualizer-position top --visualizer-opacity 0.6

# Taller frequency bars covering the bottom quarter
lyricvid song.mp3 --visualizer-type freqs --visualizer-height 0.25 --visualizer-color cyan
```

### Hardware Acceleration

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-cuda` | `true` | Use CUDA/NVENC if available; automatically falls back to libx264 if not |

### Output

| Flag | Default | Description |
|------|---------|-------------|
| `--framerate` | `0` (FFmpeg default) | Output frame rate in fps; `0` lets FFmpeg choose its default |

### `image-gen` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--inspiration` | auto-detected | Path to lyrics/text file for scene inspiration |
| `--count` | `5` | Number of images to generate |
| `--api-key` | — | Gemini API key (saved on first use) |
| `--quality` | `480p` | Image size preset: `480p`, `720p`, `1080p`, `1440p` |
| `--style` | — | Visual style applied to every image (e.g. `"oil painting, warm tones"`) |
| `--aspect-ratio` | `16:9` | Image aspect ratio passed to Gemini |

---

## How it Works Under the Hood

1. `ffprobe` reads the audio file to get its exact duration
2. The lyrics file is parsed (LRC vs plain text is auto-detected)
3. A complex FFmpeg filtergraph is assembled:
   - Background image is scaled, padded, and dimmed via `colorchannelmixer`
   - Each lyric line gets a set of `drawtext` filters: the active line at center in highlight color, plus up to 4 context lines at reduced opacity (±1 at 70%, ±2 at 40%)
   - All `drawtext` filters use `enable='between(t,start,end)'` for precise timing
   - Drift animation is applied via `zoompan` per image segment
   - Fade-in/out and title overlays are composed on top
   - If a visualizer is enabled, the audio stream is routed through `showwaves`, `showspectrum`, or `showfreqs`; the black background is keyed out and the result is composited over the video at the chosen position and opacity
4. FFmpeg encodes H.264 video + AAC audio, with real-time progress in your terminal

**Output format:** H.264, CRF 23, medium preset · AAC 192 kbps · MP4

---

## Example: Full Kitchen Sink

```bash
lyricvid generate song.mp3 \
  --lyrics song.lrc \
  --image images/ \
  --output my-video.mp4 \
  --quality 1080p \
  --aspect-ratio 16:9 \
  --font-color "#FFFFFF" \
  --highlight-color "#FF6600" \
  --bg-dim 0.5 \
  --lyric-position 0.7 \
  --lyric-fade 0.4 \
  --lyric-fade-style smooth \
  --transition dissolve \
  --transition-duration 2.5 \
  --drift zoom-in,left \
  --drift-easing cubic \
  --drift-max 50 \
  --fade-in-seconds 5 \
  --fade-in-title "My Song|The Artist" \
  --fade-in-font-size "80|55" \
  --fade-in-font-color "#FFD700|#FFFFFF" \
  --fade-out-seconds 4 \
  --fade-out-title "thanks for listening" \
  --visualizer-type waveform \
  --visualizer-color "#FFD700" \
  --visualizer-mode cline \
  --visualizer-height 0.12 \
  --visualizer-opacity 0.75 \
  --framerate 30
```

---

## Supported Formats

| Type | Extensions |
|------|-----------|
| Audio | `.mp3` `.m4a` `.flac` `.wav` |
| Lyrics | `.lrc` `.txt` |
| Images | `.jpg` `.jpeg` `.png` `.webp` |
