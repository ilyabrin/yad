// Package i18n gives yad's interface in the user's language.
//
// Translations live in locales/<code>.yaml, one file per language, and are
// built into the binary. English (en.yaml) is the source: it has every key,
// and anything a translation leaves out is shown in English. Adding a
// language means adding a file; see CONTRIBUTING.md.
package i18n

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var files embed.FS

// Lang is a language code, such as "en" or "ru": the name of its file.
type Lang string

// The languages the code refers to by name. Others come from their files.
const (
	English Lang = "en"
	Russian Lang = "ru"
)

func (l Lang) String() string { return string(l) }

// entry is one translated text: a plain string, or plural forms keyed by
// category ("one", "few", "many", "other").
type entry struct {
	text  string
	forms map[string]string
}

func (e *entry) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.MappingNode {
		return n.Decode(&e.forms)
	}
	return n.Decode(&e.text)
}

var (
	catalogs = mustLoad()
	current  = English
)

func mustLoad() map[Lang]map[string]entry {
	c, err := load(files)
	if err != nil {
		panic("i18n: " + err.Error())
	}
	return c
}

func load(fsys fs.FS) (map[Lang]map[string]entry, error) {
	names, err := fs.Glob(fsys, "locales/*.yaml")
	if err != nil {
		return nil, err
	}
	out := make(map[Lang]map[string]entry, len(names))
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		var c map[string]entry
		if err := yaml.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[Lang(strings.TrimSuffix(path.Base(name), ".yaml"))] = c
	}
	if _, ok := out[English]; !ok {
		return nil, fmt.Errorf("locales/en.yaml is missing")
	}
	return out, nil
}

// Languages lists every language yad has a translation for.
func Languages() []Lang {
	out := make([]Lang, 0, len(catalogs))
	for l := range catalogs {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Set chooses the language for everything shown from now on; one without a
// translation means English. Call it before building any screen.
func Set(l Lang) {
	if _, ok := catalogs[l]; !ok {
		l = English
	}
	current = l
}

// Current is the language in use.
func Current() Lang { return current }

func lookup(key string) entry {
	if e, ok := catalogs[current][key]; ok {
		return e
	}
	if e, ok := catalogs[English][key]; ok {
		return e
	}
	return entry{text: key} // a test makes sure this never ships
}

// T returns the text for key.
func T(key string) string { return lookup(key).text }

// F formats the text for key, like fmt.Sprintf.
func F(key string, args ...any) string { return fmt.Sprintf(T(key), args...) }

// N formats a count with the plural form the current language needs for n:
// 1 file, 2 files; 1 файл, 2 файла, 5 файлов. Each form has one %d.
func N(key string, n int) string {
	e := lookup(key)
	if e.forms == nil {
		return fmt.Sprintf(e.text, n)
	}
	for _, category := range []string{pluralCategory(current, n), "other", "many"} {
		if form, ok := e.forms[category]; ok {
			return fmt.Sprintf(form, n)
		}
	}
	return fmt.Sprintf(catalogs[English][key].forms["other"], n)
}

// pluralCategory picks the CLDR plural category of n in language l. Add a
// rule here when a new language counts differently from English.
func pluralCategory(l Lang, n int) string {
	if n < 0 {
		n = -n
	}
	switch l {
	case "ru", "uk", "be":
		m10, m100 := n%10, n%100
		switch {
		case m10 == 1 && m100 != 11:
			return "one"
		case m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14):
			return "few"
		default:
			return "many"
		}
	default:
		if n == 1 {
			return "one"
		}
		return "other"
	}
}

// requiredCategories are the plural forms a translation into l must give.
func requiredCategories(l Lang) []string {
	switch l {
	case "ru", "uk", "be":
		return []string{"one", "few", "many"}
	default:
		return []string{"one", "other"}
	}
}

// Detect chooses the language: YAD_LANG, which is meant for a single run,
// then the configured one, then the LC_ALL, LC_MESSAGES and LANG variables,
// then the system language where there is one (Windows). A setting naming a
// language without a translation counts as English.
func Detect(configured string, getenv func(string) string) Lang {
	candidates := []string{getenv("YAD_LANG"), configured, getenv("LC_ALL"), getenv("LC_MESSAGES"), getenv("LANG"), systemLanguage()}
	for _, c := range candidates {
		if l, ok := parse(c); ok {
			if _, have := catalogs[l]; have {
				return l
			}
			return English
		}
	}
	return English
}

// parse reads a setting such as "ru", "ru_RU.UTF-8" or "en-US" into a
// language code. Empty, "auto", "C" and "POSIX" say nothing.
func parse(v string) (Lang, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" || v == "auto" || v == "c" || v == "posix" || strings.HasPrefix(v, "c.") {
		return "", false
	}
	if i := strings.IndexAny(v, "_-.@"); i > 0 {
		v = v[:i]
	}
	return Lang(v), true
}
