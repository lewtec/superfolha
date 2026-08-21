package i18n

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/lewtec/superfolha/internal/paths"

	_ "embed"
)

//go:embed locales/en.json
var enJSON []byte

//go:embed locales/pt.json
var ptJSON []byte

//go:embed locales/es.json
var esJSON []byte

const (
	LangEN = "en"
	LangPT = "pt"
	LangES = "es"
)

var supported = []string{LangEN, LangPT, LangES}

func NewBundle() *i18n.Bundle {
	b := i18n.NewBundle(language.English)
	b.RegisterUnmarshalFunc("json", json.Unmarshal)
	if _, err := b.ParseMessageFileBytes(enJSON, "en.json"); err != nil {
		panic("i18n: en: " + err.Error())
	}
	if _, err := b.ParseMessageFileBytes(ptJSON, "pt.json"); err != nil {
		panic("i18n: pt: " + err.Error())
	}
	if _, err := b.ParseMessageFileBytes(esJSON, "es.json"); err != nil {
		panic("i18n: es: " + err.Error())
	}
	return b
}

func Normalize(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		lang = lang[:i]
	}
	for _, s := range supported {
		if lang == s {
			return s
		}
	}
	return LangEN
}

func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(paths.LangCookie); err == nil {
		return Normalize(c.Value)
	}
	if al := r.Header.Get("Accept-Language"); al != "" {
		if tags, _, err := language.ParseAcceptLanguage(al); err == nil {
			matcher := language.NewMatcher([]language.Tag{
				language.English, language.Portuguese, language.Spanish,
			})
			_, idx, _ := matcher.Match(tags...)
			return supported[idx]
		}
	}
	return LangEN
}

func SetLangCookie(w http.ResponseWriter, lang string) {
	http.SetCookie(w, &http.Cookie{
		Name:     paths.LangCookie,
		Value:    Normalize(lang),
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false,
	})
}

func Localizer(bundle *i18n.Bundle, lang string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, Normalize(lang), LangEN)
}

func T(loc *i18n.Localizer, id string) string {
	s, err := loc.Localize(&i18n.LocalizeConfig{MessageID: id})
	if err != nil {
		return id
	}
	return s
}

func TData(loc *i18n.Localizer, id string, data map[string]any) string {
	s, err := loc.Localize(&i18n.LocalizeConfig{MessageID: id, TemplateData: data})
	if err != nil {
		return id
	}
	return s
}

// Map returns every message for lang as a flat id→text map (editor island).
func Map(bundle *i18n.Bundle, lang string) map[string]string {
	loc := Localizer(bundle, lang)
	out := make(map[string]string)
	// Parse the file again is wasteful; walk known tags from the English file.
	var keys map[string]string
	if err := json.Unmarshal(enJSON, &keys); err != nil {
		return out
	}
	for id := range keys {
		out[id] = T(loc, id)
	}
	return out
}

func Supported() []string { return append([]string(nil), supported...) }
