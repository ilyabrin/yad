package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// --- ConfirmDialog ---

func TestConfirmDialog_Y_ConfirmsAndDone(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	_, confirmed, done := d.Update(keyMsg("y"))
	if !confirmed || !done {
		t.Errorf("y: confirmed=%v done=%v, want true/true", confirmed, done)
	}
}

func TestConfirmDialog_CapitalY_ConfirmsAndDone(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	_, confirmed, done := d.Update(keyMsg("Y"))
	if !confirmed || !done {
		t.Errorf("Y: confirmed=%v done=%v, want true/true", confirmed, done)
	}
}

func TestConfirmDialog_N_CancelsAndDone(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	_, confirmed, done := d.Update(keyMsg("n"))
	if confirmed || !done {
		t.Errorf("n: confirmed=%v done=%v, want false/true", confirmed, done)
	}
}

func TestConfirmDialog_Esc_CancelsAndDone(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	_, confirmed, done := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if confirmed || !done {
		t.Errorf("esc: confirmed=%v done=%v, want false/true", confirmed, done)
	}
}

func TestConfirmDialog_Enter_WhenNotFocused_Cancels(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?") // focused=false → No is active
	_, confirmed, done := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if confirmed || !done {
		t.Errorf("enter (unfocused): confirmed=%v done=%v, want false/true", confirmed, done)
	}
}

func TestConfirmDialog_Enter_WhenFocused_Confirms(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	d.focused = true // Yes is active
	_, confirmed, done := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !confirmed || !done {
		t.Errorf("enter (focused): confirmed=%v done=%v, want true/true", confirmed, done)
	}
}

func TestConfirmDialog_Tab_TogglesFocus(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	before := d.focused
	d2, _, done := d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if done {
		t.Error("tab should not close the dialog")
	}
	if d2.focused == before {
		t.Error("tab should toggle focused")
	}
}

func TestConfirmDialog_OtherKey_NoOp(t *testing.T) {
	d := NewConfirmDialog("Delete", "Sure?")
	_, confirmed, done := d.Update(keyMsg("x"))
	if confirmed || done {
		t.Errorf("x: confirmed=%v done=%v, want false/false", confirmed, done)
	}
}
