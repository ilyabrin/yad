package tui

import tea "github.com/charmbracelet/bubbletea"

// A terminal sends the character a key types, not the key itself, so with a
// Russian or Ukrainian layout on, d arrives as в and the hotkeys do nothing.
// cyrillicKeys maps each ЙЦУКЕН character to what the same key types on a
// QWERTY layout, so hotkeys work whichever layout is on. Typing text, as in
// the filter or a file name, is left alone.
var cyrillicKeys = map[rune]rune{
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y', 'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p', 'х': '[', 'ъ': ']',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h', 'о': 'j', 'л': 'k', 'д': 'l', 'ж': ';', 'э': '\'',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n', 'ь': 'm', 'б': ',', 'ю': '.', 'ё': '`',
	'Й': 'Q', 'Ц': 'W', 'У': 'E', 'К': 'R', 'Е': 'T', 'Н': 'Y', 'Г': 'U', 'Ш': 'I', 'Щ': 'O', 'З': 'P', 'Х': '{', 'Ъ': '}',
	'Ф': 'A', 'Ы': 'S', 'В': 'D', 'А': 'F', 'П': 'G', 'Р': 'H', 'О': 'J', 'Л': 'K', 'Д': 'L', 'Ж': ':', 'Э': '"',
	'Я': 'Z', 'Ч': 'X', 'С': 'C', 'М': 'V', 'И': 'B', 'Т': 'N', 'Ь': 'M', 'Б': '<', 'Ю': '>', 'Ё': '~',
	// Ukrainian letters on the keys where Russian has ы, ъ and э.
	'і': 's', 'ї': ']', 'є': '\'', 'І': 'S', 'Ї': '}', 'Є': '"',
	// The key that types / on QWERTY types . on ЙЦУКЕН, so . opens the
	// filter too; nothing else uses it.
	'.': '/',
}

// latinKey turns a key typed in a Cyrillic layout into its QWERTY key.
// Anything else, including keys with Ctrl or Alt, comes back unchanged.
func latinKey(msg tea.KeyMsg) tea.KeyMsg {
	if msg.Type != tea.KeyRunes || msg.Alt || len(msg.Runes) != 1 {
		return msg
	}
	if r, ok := cyrillicKeys[msg.Runes[0]]; ok {
		msg.Runes = []rune{r}
	}
	return msg
}
