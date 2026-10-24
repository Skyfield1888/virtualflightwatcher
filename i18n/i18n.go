// Package i18n holds every user-facing string of the bot.
// To add a language: add a constant, add it to Supported, and register a map in its own
// lang_xx.go file (copy lang_en.go). Missing keys fall back to English.
package i18n

import "fmt"

type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
	ES Lang = "es"
	DE Lang = "de"
	PT Lang = "pt" // Brazilian Portuguese
	IT Lang = "it"
	RU Lang = "ru"
	TR Lang = "tr"

	Default = EN
)

// Supported languages with their native display name (used for the /config choices).
var Supported = []struct {
	Code Lang
	Name string
}{
	{EN, "English"},
	{FR, "Français"},
	{ES, "Español"},
	{DE, "Deutsch"},
	{PT, "Português (Brasil)"},
	{IT, "Italiano"},
	{RU, "Русский"},
	{TR, "Türkçe"},
}

func Valid(l Lang) bool {
	for _, s := range Supported {
		if s.Code == l {
			return true
		}
	}
	return false
}

// T returns the translation of key in lang, falling back to English, then to the key itself.
func T(lang Lang, key string, args ...any) string {
	s, ok := messages[lang][key]
	if !ok {
		s, ok = messages[EN][key]
		if !ok {
			return key
		}
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// messages is filled by the init() of each lang_xx.go file.
var messages = map[Lang]map[string]string{}
