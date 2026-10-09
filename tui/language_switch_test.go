package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/ilyabrin/yad/internal/i18n"
)

func TestLSwitchesTheLanguage(t *testing.T) {
	defer i18n.Set(i18n.English)
	i18n.Set(i18n.English)
	t.Setenv("YAD_LANG", "")

	a := appWithFiles()
	a.browser.width = 160 // wide enough for every hint, L included
	if a.ChosenLanguage() != "" {
		t.Fatal("nothing to save before L is pressed")
	}
	a = press(a, runeKey('L'))
	if i18n.Current() != i18n.Russian || a.ChosenLanguage() != "ru" {
		t.Fatalf("after L: language %v, to save %q; want ru", i18n.Current(), a.ChosenLanguage())
	}
	view := ansi.Strip(a.browser.View())
	if !strings.Contains(view, "RU") || !strings.Contains(view, "L язык") {
		t.Errorf("the screen does not show Russian:\n%s", view)
	}
	if a.browser.mode != modeNormal {
		t.Errorf("mode = %v; without YAD_LANG there is nothing to warn about", a.browser.mode)
	}

	// Shift+Д is Shift+L on a Russian layout.
	a = press(a, runeKey('Д'))
	if i18n.Current() != i18n.English || a.ChosenLanguage() != "en" {
		t.Errorf("after Shift+Д: language %v, to save %q; want en", i18n.Current(), a.ChosenLanguage())
	}
	if view := ansi.Strip(a.browser.View()); !strings.Contains(view, "EN") || !strings.Contains(view, "L lang") {
		t.Errorf("the screen does not show English:\n%s", view)
	}
}

func TestLWarnsWhenYADLANGWillWin(t *testing.T) {
	defer i18n.Set(i18n.English)
	i18n.Set(i18n.English)
	t.Setenv("YAD_LANG", "en")

	a := press(appWithFiles(), runeKey('L'))
	if a.browser.mode != modeMessage || !strings.Contains(a.browser.message, "YAD_LANG") {
		t.Errorf("mode %v, message %q; want a note that YAD_LANG decides the next start", a.browser.mode, a.browser.message)
	}
}

func TestLIsTextInAField(t *testing.T) {
	defer i18n.Set(i18n.English)
	i18n.Set(i18n.English)

	a := press(appWithFiles(), runeKey('/'), runeKey('L'))
	if i18n.Current() != i18n.English || a.browser.filterInput.Value() != "L" {
		t.Errorf("L in the filter switched the language or was lost: %v, %q", i18n.Current(), a.browser.filterInput.Value())
	}
}

func TestLanguageCodeLeavesANarrowTitle(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.path = "disk:/Documents/Projects/2026"
	m.width, m.height = 30, 10
	if title := ansi.Strip(m.viewTitleBar()); strings.Contains(title, "EN") {
		t.Errorf("at 30 columns the path needs the room, title %q", title)
	}
	m.width = 100
	if title := ansi.Strip(m.viewTitleBar()); !strings.Contains(title, "EN") {
		t.Errorf("at 100 columns the language belongs in the title, got %q", title)
	}
}
