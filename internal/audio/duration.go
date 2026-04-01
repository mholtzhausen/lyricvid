package audio

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GetDuration uses ffprobe to extract the duration of an audio file in seconds.
func GetDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("ffprobe failed: %s", string(exitErr.Stderr))
		}
		return 0, fmt.Errorf("running ffprobe: %w", err)
	}

	durationStr := strings.TrimSpace(string(out))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing duration %q: %w", durationStr, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("invalid audio duration: %f seconds", duration)
	}

	return duration, nil
}

// CheckFFmpeg verifies that ffmpeg and ffprobe are installed and accessible.
func CheckFFmpeg() error {
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s not found in PATH. Please install FFmpeg: https://ffmpeg.org/download.html", tool)
		}
	}
	return nil
}
