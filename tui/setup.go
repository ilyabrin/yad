package tui

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/yad/internal/auth"
)

// setupStep tracks which stage of the OAuth flow we're in.
type setupStep int

const (
	stepShowURL   setupStep = iota // display the auth URL, wait for user to open it
	stepEnterCode                  // user pastes the verification code
	stepExchange                   // exchanging code for token (spinner)
	stepDone                       // success - ready to launch browser
	stepError                      // unrecoverable error
)

// --- Messages ---------------------------------------------------------------

// SetupDoneMsg is published when setup completes successfully.
type SetupDoneMsg struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

type exchangeResultMsg struct {
	resp *auth.TokenResponse
	err  error
}

type browserOpenedMsg struct{ err error }

// --- SetupModel -------------------------------------------------------------

type SetupModel struct {
	oauthCfg *auth.Config

	step        setupStep
	authURL     string
	browserOpen bool // true if we successfully opened the browser

	input   textinput.Model
	spinner spinner.Model
	errMsg  string

	width  int
	height int

	fullOAuth bool
}

func NewSetupModel(oauthCfg *auth.Config) SetupModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	ti := textinput.New()
	ti.Width = 56

	fullOAuth := auth.IsConfigured(oauthCfg)

	var firstStep setupStep
	var placeholder, authURL string

	if fullOAuth {
		u, err := auth.AuthURL(oauthCfg)
		if err != nil {
			return SetupModel{
				step:    stepError,
				errMsg:  err.Error(),
				spinner: sp,
				input:   ti,
			}
		}
		authURL = u
		firstStep = stepShowURL
		placeholder = "Paste the verification code from Yandex…"
	} else {
		firstStep = stepEnterCode
		placeholder = "Paste your Yandex OAuth token…"
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}

	ti.Placeholder = placeholder
	ti.Focus()

	return SetupModel{
		oauthCfg:  oauthCfg,
		step:      firstStep,
		authURL:   authURL,
		input:     ti,
		spinner:   sp,
		fullOAuth: fullOAuth,
	}
}

func (m SetupModel) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.spinner.Tick}
	if m.step == stepShowURL && m.authURL != "" {
		cmds = append(cmds, cmdOpenBrowser(m.authURL))
	}
	return tea.Batch(cmds...)
}

// --- Update -----------------------------------------------------------------

func (m SetupModel) Update(msg tea.Msg) (SetupModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case spinner.TickMsg:
		if m.step == stepExchange {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case browserOpenedMsg:
		m.browserOpen = msg.err == nil

	case exchangeResultMsg:
		if msg.err != nil {
			m.step = stepError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.step = stepDone
		return m, func() tea.Msg {
			return SetupDoneMsg{
				AccessToken:  msg.resp.AccessToken,
				RefreshToken: msg.resp.RefreshToken,
				Expiry:       msg.resp.ExpiresAt(),
			}
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m SetupModel) handleKey(msg tea.KeyMsg) (SetupModel, tea.Cmd) {
	switch m.step {

	case stepShowURL:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		default:
			m.step = stepEnterCode
			m.input.Focus()
		}
		return m, nil

	case stepEnterCode:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				return m, nil
			}
			if m.fullOAuth {
				m.step = stepExchange
				oauthCfg := m.oauthCfg
				code := value
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()
						resp, err := auth.ExchangeCode(ctx, code, oauthCfg)
						return exchangeResultMsg{resp: resp, err: err}
					},
				)
			}
			return m, func() tea.Msg {
				return SetupDoneMsg{AccessToken: value}
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case stepError:
		return m, tea.Quit
	}

	return m, nil
}

// --- View -------------------------------------------------------------------

func (m SetupModel) View() string {
	var b strings.Builder

	titleText := "  YaD  ·  Setup"
	b.WriteString(StyleTitle.Render(titleText))
	if m.width > lipgloss.Width(titleText) {
		b.WriteString(strings.Repeat(" ", m.width-lipgloss.Width(titleText)))
	}
	b.WriteString("\n\n")

	switch m.step {
	case stepShowURL:
		b.WriteString(m.viewShowURL())
	case stepEnterCode:
		b.WriteString(m.viewEnterCode())
	case stepExchange:
		b.WriteString(m.viewExchange())
	case stepDone:
		b.WriteString(StyleSuccess.Render("✓ Authorised! Starting YaD…"))
	case stepError:
		b.WriteString(StyleError.Render("✗ " + m.errMsg))
		b.WriteString("\n\n")
		b.WriteString(StyleMuted("Press any key to exit."))
	}

	return b.String()
}

func (m SetupModel) viewShowURL() string {
	var b strings.Builder

	b.WriteString(StyleDialogTitle.Render("Step 1 - Authorise YaD"))
	b.WriteString("\n\n")

	if m.browserOpen {
		b.WriteString(StyleSuccess.Render("✓ Browser opened automatically"))
	} else {
		b.WriteString(StyleMuted("Could not open browser automatically."))
		b.WriteString("\n")
		b.WriteString("Open this URL manually:\n")
	}

	b.WriteString("\n")

	// Render URL on its own line without wrapping
	urlStyle := lipgloss.NewStyle().Foreground(colorAccent)
	b.WriteString(urlStyle.Render(m.authURL))
	b.WriteString("\n\n")

	b.WriteString(StyleMuted("Log in and grant access. Yandex will show a verification code."))
	b.WriteString("\n\n")
	b.WriteString(StyleStatusKey.Render("any key") + " - I've authorised, show code input   " +
		StyleStatusKey.Render("Esc") + " quit")

	// No Width() here - let the terminal handle line length naturally
	return StyleDialog.Render(b.String())
}

func (m SetupModel) viewEnterCode() string {
	var b strings.Builder

	if m.fullOAuth {
		b.WriteString(StyleDialogTitle.Render("Step 2 - Enter verification code"))
		b.WriteString("\n")
		b.WriteString(StyleMuted("Paste the code shown on the Yandex page:"))
	} else {
		b.WriteString(StyleDialogTitle.Render("Paste your Yandex OAuth token"))
		b.WriteString("\n")
		b.WriteString(StyleMuted("Get a token at https://oauth.yandex.ru"))
	}
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(StyleStatusKey.Render("↵") + " confirm   " +
		StyleStatusKey.Render("Esc") + " quit")

	return StyleDialog.Width(wrapWidth(m.width)).Render(b.String())
}

func (m SetupModel) viewExchange() string {
	return StyleDialog.Width(wrapWidth(m.width)).Render(
		m.spinner.View() + "  Exchanging code for token…",
	)
}

// --- Helpers ----------------------------------------------------------------

// cmdOpenBrowser tries to open url in the system default browser.
func cmdOpenBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			// cmd /c start truncates URLs at '&'.
			// rundll32 handles full URLs with query parameters correctly.
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		return browserOpenedMsg{err: cmd.Start()}
	}
}

func wrapWidth(termWidth int) int {
	w := termWidth - 8
	if w > 72 {
		return 72
	}
	if w < 40 {
		return 40
	}
	return w
}
