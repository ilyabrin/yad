//go:build !windows

package i18n

// systemLanguage has nothing to add on Unix: the locale variables are it.
func systemLanguage() string { return "" }
