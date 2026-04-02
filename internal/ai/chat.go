package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/genai"
)

const modelName = "gemini-3.1-flash-lite-preview"

var (
	goldColor  = lipgloss.Color("#FFD700")
	blueColor  = lipgloss.Color("#88AAFF")
	dimColor   = lipgloss.Color("#555555")
	errColor   = lipgloss.Color("#FF5555")
	whiteColor = lipgloss.Color("#DDDDDD")

	userLabelStyle = lipgloss.NewStyle().Foreground(goldColor).Bold(true)
	userTextStyle  = lipgloss.NewStyle().Foreground(goldColor)
	aiLabelStyle   = lipgloss.NewStyle().Foreground(blueColor).Bold(true)
	aiTextStyle    = lipgloss.NewStyle().Foreground(whiteColor)
	errStyle       = lipgloss.NewStyle().Foreground(errColor)
	dimStyle       = lipgloss.NewStyle().Foreground(dimColor)
	sepStyle       = lipgloss.NewStyle().Foreground(dimColor)
)

type aiResponseMsg string
type errMsg struct{ error }

type chatMessage struct {
	role string // "user", "assistant", "error"
	text string
}

type model struct {
	viewport     viewport.Model
	input        textinput.Model
	spinner      spinner.Model
	messages     []chatMessage
	history      []*genai.Content
	client       *genai.Client
	systemPrompt string
	waiting      bool
	ready        bool
	width        int
	height       int
}

func newModel(client *genai.Client, systemPrompt string) model {
	ti := textinput.New()
	ti.Placeholder = "Ask something about lyricvid..."
	ti.Focus()
	ti.CharLimit = 500

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(goldColor)

	return model{
		input:        ti,
		spinner:      sp,
		client:       client,
		systemPrompt: systemPrompt,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.width - 6

		headerH := lipgloss.Height(m.headerView())
		// footer: sep + inputRow + hints = 3 lines; plus the sep after header = 1 → total 4
		vpH := m.height - headerH - 4
		if vpH < 1 {
			vpH = 1
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, vpH)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpH
			m.viewport.SetContent(m.renderMessages())
		}

	case aiResponseMsg:
		m.waiting = false
		text := string(msg)
		m.messages = append(m.messages, chatMessage{role: "assistant", text: text})
		m.history = append(m.history, &genai.Content{
			Role:  "model",
			Parts: []*genai.Part{{Text: text}},
		})
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}

	case errMsg:
		m.waiting = false
		m.messages = append(m.messages, chatMessage{role: "error", text: msg.Error()})
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if !m.waiting {
				text := strings.TrimSpace(m.input.Value())
				if text != "" {
					m.input.SetValue("")
					m.messages = append(m.messages, chatMessage{role: "user", text: text})

					snapshot := make([]*genai.Content, len(m.history), len(m.history)+1)
					copy(snapshot, m.history)
					snapshot = append(snapshot, &genai.Content{
						Role:  "user",
						Parts: []*genai.Part{{Text: text}},
					})
					m.history = snapshot

					if m.ready {
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()
					}
					m.waiting = true
					cmds = append(cmds, sendMessage(m.client, m.systemPrompt, snapshot))
				}
			}
			return m, tea.Batch(cmds...)

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Viewport handles mouse and scroll keys
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return dimStyle.Render("Initializing...")
	}

	sep := sepStyle.Render(strings.Repeat("─", m.width))

	var inputRow string
	if m.waiting {
		inputRow = "  " + m.spinner.View() + " " + dimStyle.Render("Thinking...")
	} else {
		inputRow = "  > " + m.input.View()
	}
	hints := dimStyle.Render("  ↵ send  •  ctrl+c quit")

	return strings.Join([]string{
		m.headerView(),
		sep,
		m.viewport.View(),
		sep,
		inputRow,
		hints,
	}, "\n")
}

func (m model) headerView() string {
	boxW := m.width - 4
	if boxW > 66 {
		boxW = 66
	}
	if boxW < 24 {
		boxW = 24
	}

	title := lipgloss.NewStyle().Foreground(goldColor).Bold(true).
		Render("♪  lyricvid AI  ·  Gemini Chat  ♪")
	sub := dimStyle.Render("Ask anything about lyricvid — flags, config, usage")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(goldColor).
		Padding(0, 2).
		Width(boxW).
		Align(lipgloss.Center).
		MarginLeft(1).
		MarginTop(0).
		Render(title + "\n" + sub)
}

func (m model) renderMessages() string {
	if len(m.messages) == 0 {
		return "\n" + dimStyle.Render("  Ask a question to get started...")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString("\n")
		switch msg.role {
		case "user":
			b.WriteString(userLabelStyle.Render("  You") + "\n")
			for _, line := range strings.Split(msg.text, "\n") {
				b.WriteString(userTextStyle.Render("  "+line) + "\n")
			}
		case "assistant":
			b.WriteString(aiLabelStyle.Render("  lyricvid AI") + "\n")
			for _, line := range strings.Split(msg.text, "\n") {
				b.WriteString(aiTextStyle.Render("  "+line) + "\n")
			}
		case "error":
			b.WriteString(errStyle.Render("  ✖  "+msg.text) + "\n")
		}
	}
	return b.String()
}

func sendMessage(client *genai.Client, systemPrompt string, history []*genai.Content) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		contents := make([]*genai.Content, 0, len(history)+2)
		contents = append(contents,
			&genai.Content{
				Role:  "user",
				Parts: []*genai.Part{{Text: systemPrompt}},
			},
			&genai.Content{
				Role:  "model",
				Parts: []*genai.Part{{Text: "Understood. I have read the lyricvid documentation and I'm ready to help."}},
			},
		)
		contents = append(contents, history...)

		result, err := client.Models.GenerateContent(ctx, modelName, contents,
			&genai.GenerateContentConfig{
				ResponseModalities: []string{"TEXT"},
			},
		)
		if err != nil {
			return errMsg{err}
		}
		if len(result.Candidates) == 0 {
			return errMsg{fmt.Errorf("no response from model")}
		}
		return aiResponseMsg(result.Text())
	}
}

// Run starts the interactive AI chat TUI.
func Run(ctx context.Context, apiKey, helpContent string) error {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		return fmt.Errorf("creating Gemini client: %w", err)
	}

	systemPrompt := buildSystemPrompt(helpContent)
	m := newModel(client, systemPrompt)

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = prog.Run()
	return err
}

func buildSystemPrompt(helpContent string) string {
	return `You are a helpful assistant embedded in the lyricvid CLI tool.
lyricvid generates animated MP4 music videos with synchronized lyrics overlays, driven entirely by FFmpeg.

Answer questions about lyricvid usage: flags, subcommands, config files, supported values, and workflow.
Be concise. Use plain text; avoid markdown formatting since the output is a terminal.
If asked about something unrelated to lyricvid, politely redirect to lyricvid topics.

Here is the complete help documentation for this version of lyricvid:

---
` + helpContent + `
---`
}
