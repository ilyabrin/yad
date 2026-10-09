package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Every translation must have exactly the keys of en.yaml, of the same kind,
// with the plural forms its language needs.
func TestTranslationsAreComplete(t *testing.T) {
	en := catalogs[English]
	for _, l := range Languages() {
		c := catalogs[l]
		for key, want := range en {
			got, ok := c[key]
			switch {
			case !ok:
				t.Errorf("%s.yaml: missing %s", l, key)
			case (want.forms == nil) != (got.forms == nil):
				t.Errorf("%s.yaml: %s should be %s", l, key, kind(want))
			case got.forms != nil:
				for _, cat := range requiredCategories(l) {
					if got.forms[cat] == "" {
						t.Errorf("%s.yaml: %s lacks the %q form", l, key, cat)
					}
				}
			}
		}
		for key := range c {
			if _, ok := en[key]; !ok {
				t.Errorf("%s.yaml: %s is not in en.yaml", l, key)
			}
		}
	}
}

func kind(e entry) string {
	if e.forms != nil {
		return "plural forms"
	}
	return "plain text"
}

var verb = regexp.MustCompile(`%[-+# 0]*[0-9]*(\.[0-9]+)?[a-zA-Z%]`)

// A translation must take the same values in the same order, or Sprintf
// prints %!d(string=...) on screen.
func TestTranslationsKeepFormatVerbs(t *testing.T) {
	for _, l := range Languages() {
		for key, e := range catalogs[l] {
			want := verbs(catalogs[English][key])
			for _, text := range texts(e) {
				if got := verb.FindAllString(text, -1); !slices.Equal(got, want) {
					t.Errorf("%s.yaml: %s uses %v, en.yaml %v", l, key, got, want)
				}
			}
		}
	}
}

func texts(e entry) []string {
	if e.forms == nil {
		return []string{e.text}
	}
	var out []string
	for _, f := range e.forms {
		out = append(out, f)
	}
	return out
}

func verbs(e entry) []string {
	if e.forms != nil {
		return verb.FindAllString(e.forms["other"], -1)
	}
	return verb.FindAllString(e.text, -1)
}

// Every key the code asks for exists, and every key in en.yaml is used.
func TestKeysMatchTheCode(t *testing.T) {
	root := filepath.Join("..", "..")
	used := map[string]string{} // key -> where
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" && sel.Sel.Name != "F" && sel.Sel.Name != "N" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "i18n" {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: i18n.%s with a computed key; keys must be literals so they can be checked", p, sel.Sel.Name)
				return true
			}
			key, _ := strconv.Unquote(lit.Value)
			used[key] = p
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, where := range used {
		if _, ok := catalogs[English][key]; !ok {
			t.Errorf("%s asks for %q, which en.yaml does not have", where, key)
		}
	}
	for key := range catalogs[English] {
		if _, ok := used[key]; !ok {
			t.Errorf("en.yaml has %q, which no code uses", key)
		}
	}
}

func TestPlurals(t *testing.T) {
	defer Set(English)
	Set(Russian)
	ru := map[int]string{0: "many", 1: "one", 2: "few", 4: "few", 5: "many", 11: "many", 12: "many",
		14: "many", 21: "one", 22: "few", 101: "one", 111: "many", 1000: "many"}
	for n, want := range ru {
		if got := pluralCategory(Russian, n); got != want {
			t.Errorf("ru %d: %s, want %s", n, got, want)
		}
	}
	if got := N("trash.items", 3); got != "3 объекта" {
		t.Errorf("N = %q", got)
	}
	Set(English)
	if got := N("trash.items", 1); got != "1 item" {
		t.Errorf("N = %q", got)
	}
}

func TestDetect(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}
	cases := []struct {
		name       string
		configured string
		vars       map[string]string
		want       Lang
	}{
		{"config beats the locale", "en", map[string]string{"LANG": "ru_RU.UTF-8"}, English},
		{"YAD_LANG beats the config", "en", map[string]string{"YAD_LANG": "ru"}, Russian},
		{"config ru", "ru", nil, Russian},
		{"auto falls through to LANG", "auto", map[string]string{"LANG": "ru_RU.UTF-8"}, Russian},
		{"LC_ALL before LANG", "", map[string]string{"LC_ALL": "ru_RU.UTF-8", "LANG": "en_US.UTF-8"}, Russian},
		{"C locale says nothing", "", map[string]string{"LC_ALL": "C", "LANG": "ru_RU.UTF-8"}, Russian},
		{"a language without a translation gets English", "", map[string]string{"LANG": "de_DE.UTF-8"}, English},
		{"Windows-style names", "ru-RU", nil, Russian},
	}
	for _, tc := range cases {
		if got := Detect(tc.configured, env(tc.vars)); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMissingTranslationFallsBackToEnglish(t *testing.T) {
	defer Set(English)
	Set("xx") // no such file
	if Current() != English {
		t.Errorf("an unknown language should mean English, got %v", Current())
	}
}

// The locales folder must hold nothing but translations: a stray file would
// be built into every binary.
func TestLocalesFolderHoldsOnlyTranslations(t *testing.T) {
	entries, err := os.ReadDir("locales")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			t.Errorf("locales/%s is not a translation", e.Name())
		}
	}
}
