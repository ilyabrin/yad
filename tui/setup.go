package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/yad/internal/auth"
	"github.com/ilyabrin/yad/internal/i18n"
	"github.com/ilyabrin/yad/internal/qr"
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

// --- SetupModel -------------------------------------------------------------

type SetupModel struct {
	oauthCfg *auth.Config

	// pkce is generated once per setup session. Its verifier lives only in
	// memory: it is never written to the config file, so quitting before the
	// code is entered simply means starting the sign-in again.
	pkce *auth.PKCE

	step        setupStep
	authURL     string
	qrLines     []string // the sign-in link as a QR code; nil if it could not be built
	browserOpen bool     // true if we successfully opened the browser

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
	var qrLines []string

	var pkce *auth.PKCE

	if fullOAuth {
		p, err := auth.NewPKCE()
		if err != nil {
			return SetupModel{
				step:    stepError,
				errMsg:  err.Error(),
				spinner: sp,
				input:   ti,
			}
		}
		pkce = p

		u, err := auth.AuthURL(oauthCfg, pkce.Challenge)
		if err != nil {
			return SetupModel{
				step:    stepError,
				errMsg:  err.Error(),
				spinner: sp,
				input:   ti,
			}
		}
		authURL = u
		if code, err := qr.Encode(u, qr.L); err == nil {
			qrLines = code.HalfBlocks(qrQuietZone)
		}
		firstStep = stepShowURL
		placeholder = i18n.T("setup.paste_the_verification_code_from")
	} else {
		firstStep = stepEnterCode
		placeholder = i18n.T("setup.paste_token_placeholder")
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}

	ti.Placeholder = placeholder
	ti.Focus()

	return SetupModel{
		oauthCfg:  oauthCfg,
		pkce:      pkce,
		step:      firstStep,
		authURL:   authURL,
		qrLines:   qrLines,
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
				var verifier string
				if m.pkce != nil {
					verifier = m.pkce.Verifier
				}
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()
						resp, err := auth.ExchangeCode(ctx, code, verifier, oauthCfg)
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

	titleText := i18n.T("setup.title")
	b.WriteString(StyleTitle.Render(titleText))
	if m.width > lipgloss.Width(titleText) {
		b.WriteString(strings.Repeat(" ", m.width-lipgloss.Width(titleText)))
	}
	b.WriteString("\n\n")

	switch m.step {
	case stepShowURL:
		b.WriteString(m.viewShowURL())
		b.WriteString(m.viewQR(b.String()))
	case stepEnterCode:
		b.WriteString(m.viewEnterCode())
	case stepExchange:
		b.WriteString(m.viewExchange())
	case stepDone:
		b.WriteString(StyleSuccess.Render(i18n.T("setup.authorised_starting_yad")))
	case stepError:
		b.WriteString(StyleError.Render("✗ " + m.errMsg))
		b.WriteString("\n\n")
		b.WriteString(StyleMuted(i18n.T("setup.press_any_key_to_exit")))
	}

	return b.String()
}

func (m SetupModel) viewShowURL() string {
	var b strings.Builder

	b.WriteString(StyleDialogTitle.Render(i18n.T("setup.step_1_authorise_yad")))
	b.WriteString("\n\n")

	if m.browserOpen {
		b.WriteString(StyleSuccess.Render(i18n.T("setup.browser_opened_automatically")))
	} else {
		b.WriteString(StyleMuted(i18n.T("setup.could_not_open_browser_automatically")))
		b.WriteString("\n")
		b.WriteString(i18n.T("setup.open_this_url_manually") + "\n")
	}

	b.WriteString("\n")

	// Render URL on its own line without wrapping
	urlStyle := lipgloss.NewStyle().Foreground(colorAccent)
	b.WriteString(urlStyle.Render(m.authURL))
	b.WriteString("\n\n")

	b.WriteString(StyleMuted(i18n.T("setup.log_in_and_grant_access")))
	b.WriteString("\n\n")
	b.WriteString(StyleStatusKey.Render(i18n.T("setup.any_key")) + i18n.T("setup.i_ve_authorised_show_code") +
		StyleStatusKey.Render("Esc") + i18n.T("hint.quit"))

	// No Width() here - let the terminal handle line length naturally
	return StyleDialog.Render(b.String())
}

func (m SetupModel) viewEnterCode() string {
	var b strings.Builder

	if m.fullOAuth {
		b.WriteString(StyleDialogTitle.Render(i18n.T("setup.step_2_enter_verification_code")))
		b.WriteString("\n")
		b.WriteString(StyleMuted(i18n.T("setup.paste_the_code_shown_on")))
	} else {
		b.WriteString(StyleDialogTitle.Render(i18n.T("setup.paste_token_title")))
		b.WriteString("\n")
		b.WriteString(StyleMuted(i18n.T("setup.get_a_token_at_https")))
	}
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(StyleStatusKey.Render("↵") + i18n.T("hint.confirm") +
		StyleStatusKey.Render("Esc") + i18n.T("hint.quit"))

	return StyleDialog.Width(wrapWidth(m.width)).Render(b.String())
}

func (m SetupModel) viewExchange() string {
	return StyleDialog.Width(wrapWidth(m.width)).Render(
		m.spinner.View() + i18n.T("setup.exchanging_code_for_token"),
	)
}

// --- Helpers ----------------------------------------------------------------

// cmdOpenBrowser tries to open url in the system default browser.

// viewQR shows the sign-in link as a QR code under the dialog, so it can be
// opened on a phone, which helps most when this machine has no browser.
// above is everything already on screen; the code is only drawn when it fits
// below that, because a QR code cut off by the terminal cannot be scanned.
func (m SetupModel) viewQR(above string) string {
	if len(m.qrLines) == 0 || m.width == 0 || m.height == 0 {
		return ""
	}

	// The dialog does not wrap the long URL, so count the rows the terminal
	// will really use, then a blank line and the caption.
	need := rowsOnScreen(above, m.width) + 2 + len(m.qrLines)
	if lipgloss.Width(m.qrLines[0]) > m.width || need > m.height {
		return "\n\n" + StyleMuted(i18n.T("setup.make_the_window_taller_or"))
	}

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(StyleMuted(i18n.T("setup.or_scan_this_with_your")))
	for _, line := range m.qrLines {
		b.WriteString("\n")
		b.WriteString(StyleQR.Render(line))
	}
	return b.String()
}

// rowsOnScreen counts the terminal rows s occupies once lines wider than the
// terminal wrap.
func rowsOnScreen(s string, width int) int {
	rows := 0
	for _, line := range strings.Split(s, "\n") {
		rows += max(1, (lipgloss.Width(line)+width-1)/width)
	}
	return rows
}

func wrapWidth(termWidth int) int {
	w := termWidth - setupWrapMargin
	if w > setupWrapMax {
		return setupWrapMax
	}
	if w < setupWrapMin {
		return setupWrapMin
	}
	return w
}
