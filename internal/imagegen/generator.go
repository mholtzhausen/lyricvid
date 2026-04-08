package imagegen

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"google.golang.org/genai"
)

//go:embed prompts/*.txt
var promptFS embed.FS

const (
	modelText  = "gemini-3.1-flash-lite-preview"
	modelImage = "gemini-3.1-flash-image-preview"
)

// QualityPreset maps a quality label to image dimensions and Gemini ImageSize.
type QualityPreset struct {
	Width     int
	Height    int
	ImageSize string
}

// QualityPresets is the set of supported quality values for the --quality flag.
var QualityPresets = map[string]QualityPreset{
	"480p":  {854, 480, "1K"},
	"720p":  {1280, 720, "1K"},
	"1080p": {1920, 1080, "2K"},
	"1440p": {2560, 1440, "4K"},
}

// Config holds all parameters for image generation.
type Config struct {
	APIKey          string
	InspirationText string
	Count           int
	OutputDir       string // absolute path; created if absent
	ImageSize       string // Gemini ImageConfig.ImageSize: "1K", "2K", "4K"
	Style           string // optional visual style injected into every prompt
	AspectRatio     string // e.g. "16:9"; passed directly to Gemini ImageConfig.AspectRatio
}

// Generate builds a story arc and produces Count images, saving them to OutputDir.
func Generate(ctx context.Context, cfg Config) error {
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %q: %w", cfg.OutputDir, err)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  cfg.APIKey,
	})
	if err != nil {
		return fmt.Errorf("creating Gemini client: %w", err)
	}

	fmt.Printf("Generating story arc for %d scenes...\n", cfg.Count)
	scenes, arcPrompt, err := generateStoryArc(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("generating story arc: %w", err)
	}
	fmt.Printf("Story arc prompt:\n%s\n\n", arcPrompt)

	for i, scene := range scenes {
		label := scene
		if len(label) > 60 {
			label = label[:60] + "..."
		}
		fmt.Printf("Generating image %d/%d: %s\n", i+1, cfg.Count, label)

		data, mimeType, imagePrompt, err := generateImage(ctx, client, cfg, scene)
		if err != nil {
			return fmt.Errorf("scene %d: %w", i+1, err)
		}
		fmt.Printf("Image prompt:\n%s\n\n", imagePrompt)

		ext := mimeToExt(mimeType)
		filename := fmt.Sprintf("image_%03d%s", i+1, ext)
		path := filepath.Join(cfg.OutputDir, filename)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("saving image %q: %w", path, err)
		}
		fmt.Printf("Image saved: %s\n", filepath.Join(filepath.Base(cfg.OutputDir), filename))
	}

	fmt.Printf("Image generation complete: %d images saved to %s\n", cfg.Count, cfg.OutputDir)
	return nil
}

func generateStoryArc(ctx context.Context, client *genai.Client, cfg Config) ([]string, string, error) {
	prompt, err := renderPrompt("story_arc.txt", map[string]any{
		"Count":           cfg.Count,
		"InspirationText": cfg.InspirationText,
		"Style":           cfg.Style,
	})
	if err != nil {
		return nil, "", fmt.Errorf("rendering story arc prompt: %w", err)
	}

	systemInstruction, err := loadSystemInstruction("story_arc_system.txt", map[string]any{
		"Count":           cfg.Count,
		"InspirationText": cfg.InspirationText,
		"Style":           cfg.Style,
	})
	if err != nil {
		return nil, "", err
	}

	result, err := client.Models.GenerateContent(ctx, modelText,
		[]*genai.Content{{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		}},
		&genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			ResponseModalities: []string{"TEXT"},
		},
	)
	if err != nil {
		return nil, "", err
	}
	if len(result.Candidates) == 0 {
		return nil, "", fmt.Errorf("no candidates in story arc response")
	}

	raw := result.Text()
	scenes, err := parseScenes(raw, cfg.Count)
	if err != nil {
		return nil, "", err
	}
	return scenes, prompt, nil
}

func generateImage(ctx context.Context, client *genai.Client, cfg Config, scene string) ([]byte, string, string, error) {
	prompt, err := renderPrompt("image_scene.txt", map[string]any{
		"Scene": scene,
		"Style": cfg.Style,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("rendering image prompt: %w", err)
	}

	systemInstruction, err := loadSystemInstruction("story_arc_system.txt", map[string]any{
		"Scene": scene,
		"Style": cfg.Style,
	})
	if err != nil {
		return nil, "", "", err
	}

	result, err := client.Models.GenerateContent(ctx, modelImage,
		[]*genai.Content{{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		}},
		&genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			ResponseModalities: []string{"IMAGE"},
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingLevel: genai.ThinkingLevelMinimal,
			},
			ImageConfig: &genai.ImageConfig{
				AspectRatio: cfg.AspectRatio,
				ImageSize:   cfg.ImageSize,
			},
		},
	)
	if err != nil {
		return nil, "", "", err
	}
	if len(result.Candidates) == 0 {
		return nil, "", "", fmt.Errorf("no candidates in image response")
	}

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil {
			return part.InlineData.Data, part.InlineData.MIMEType, prompt, nil
		}
	}
	return nil, "", "", fmt.Errorf("no image data in response")
}

var sceneLineRe = regexp.MustCompile(`^\d+\.\s+(.+)$`)

func parseScenes(text string, count int) ([]string, error) {
	var scenes []string
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if m := sceneLineRe.FindStringSubmatch(line); m != nil {
			scenes = append(scenes, m[1])
		}
	}
	if len(scenes) < count {
		return nil, fmt.Errorf("story arc yielded only %d/%d scenes; raw response: %q", len(scenes), count, text)
	}
	return scenes[:count], nil
}

func loadSystemInstruction(name string, data any) (*genai.Content, error) {
	text, err := renderPrompt(name, data)
	if err != nil {
		return nil, fmt.Errorf("rendering system prompt: %w", err)
	}
	return &genai.Content{
		Role:  "system",
		Parts: []*genai.Part{{Text: strings.TrimSpace(text)}},
	}, nil
}

func renderPrompt(name string, data any) (string, error) {
	raw, err := promptFS.ReadFile("prompts/" + name)
	if err != nil {
		return "", fmt.Errorf("reading prompt %q: %w", name, err)
	}
	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parsing prompt template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing prompt template %q: %w", name, err)
	}
	return buf.String(), nil
}

func mimeToExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}
