package tui

import (
	"testing"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestEveryRussianLetterHasAKey(t *testing.T) {
	for _, r := range "абвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" {
		latin, ok := cyrillicKeys[r]
		if !ok {
			t.Errorf("%c has no QWERTY key", r)
			continue
		}
		if latin > unicode.MaxASCII {
			t.Errorf("%c maps to %c, which is not on a QWERTY keyboard", r, latin)
		}
		// Shift must stay Shift: D deletes, d downloads.
		if unicode.IsUpper(r) != unicode.IsUpper(latin) && unicode.IsLetter(latin) {
			t.Errorf("%c maps to %c: the case does not match", r, latin)
		}
	}
}

func TestLatinKey(t *testing.T) {
	cases := []struct {
		in   tea.KeyMsg
		want string
	}{
		{runeKey('в'), "d"},
		{runeKey('В'), "D"},
		{runeKey('.'), "/"},
		{runeKey('і'), "s"},
		{runeKey('x'), "x"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'в'}, Alt: true}, "alt+в"},
		{tea.KeyMsg{Type: tea.KeyEnter}, "enter"},
	}
	for _, tc := range cases {
		if got := latinKey(tc.in).String(); got != tc.want {
			t.Errorf("latinKey(%q) = %q, want %q", tc.in.String(), got, tc.want)
		}
	}
}

func appWithFiles() *App {
	a := newTestApp()
	a.browser = newBrowserWithEntries(makeEntries(3))
	a.browser.width, a.browser.height = 80, 24
	a.browser.filterInput = textinput.New() // as NewBrowserModel does
	return a
}

func press(a *App, keys ...tea.KeyMsg) *App {
	for _, k := range keys {
		m, _ := a.Update(k)
		a = m.(*App)
	}
	return a
}

func TestHotkeysWorkInARussianLayout(t *testing.T) {
	// Shift+В is Shift+D: delete, which asks first.
	a := press(appWithFiles(), runeKey('В'))
	if a.browser.mode != modeConfirmDelete {
		t.Fatalf("В: mode = %v, want the delete confirmation", a.browser.mode)
	}
	// т is n: no, keep the file.
	a = press(a, runeKey('т'))
	if a.browser.mode != modeNormal {
		t.Errorf("т in the confirmation: mode = %v, want back to the list", a.browser.mode)
	}
	// . is /: the filter.
	if a = press(a, runeKey('.')); a.browser.mode != modeFilter {
		t.Errorf(".: mode = %v, want the filter", a.browser.mode)
	}
}

func TestTextFieldsKeepCyrillic(t *testing.T) {
	a := press(appWithFiles(), runeKey('/'), runeKey('о'), runeKey('т'), runeKey('ч'))
	if a.browser.mode != modeFilter || a.browser.filterInput.Value() != "отч" {
		t.Errorf("filter = %q in mode %v, want the Cyrillic text as typed", a.browser.filterInput.Value(), a.browser.mode)
	}

	a = press(appWithFiles(), runeKey('n'), runeKey('в'))
	if a.browser.mode != modeInputNewDir || a.browser.inputDlg.Value() != "в" {
		t.Errorf("new folder name = %q in mode %v, want в", a.browser.inputDlg.Value(), a.browser.mode)
	}

	// The share dialog's password takes Cyrillic too.
	a = press(appWithFiles(), runeKey('з'), tea.KeyMsg{Type: tea.KeyTab}, runeKey('п'))
	if a.browser.mode != modeShare {
		t.Fatalf("з should open the share dialog, mode = %v", a.browser.mode)
	}
	if s := a.browser.shareDlg.Settings(time.Now()); s == nil || s.Password != "п" {
		t.Errorf("password = %+v, want п", s)
	}
}
