package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	appi18n "github.com/lewtec/superfolha/internal/i18n"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/ui/layout"
	"github.com/lewtec/superfolha/internal/ui/pages"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (s *Server) lang(r *http.Request) string {
	return appi18n.FromRequest(r)
}

func (s *Server) loc(r *http.Request) *i18n.Localizer {
	return appi18n.Localizer(s.bundle, s.lang(r))
}

func (s *Server) chrome(r *http.Request, title string) layout.Chrome {
	loc := s.loc(r)
	c := layout.Chrome{
		Title: title,
		Lang:  s.lang(r),
		Flash: r.URL.Query().Get("flash"),
		Error: r.URL.Query().Get("error"),
		T:     func(id string) string { return appi18n.T(loc, id) },
	}
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		c.LoggedIn = true
		c.Email = u.Email
	}
	if c.Error != "" {
		c.Error = localizeMessage(loc, c.Error)
	}
	if c.Flash != "" {
		c.Flash = localizeMessage(loc, c.Flash)
	}
	return c
}

func localizeMessage(loc *i18n.Localizer, raw string) string {
	if translated := appi18n.T(loc, raw); translated != raw {
		return translated
	}
	if translated := appi18n.T(loc, "errors."+raw); translated != "errors."+raw {
		return translated
	}
	return raw
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("templ render", "err", err)
	}
}

func (s *Server) requirePageUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.GetUserFromContext(r.Context()); !ok {
			http.Redirect(w, r, paths.LoginNext(r.URL.RequestURI()), http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	c := s.chrome(r, "")
	s.render(w, r, pages.Landing(c))
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.GetUserFromContext(r.Context()); ok {
		http.Redirect(w, r, paths.Projects(), http.StatusSeeOther)
		return
	}
	c := s.chrome(r, appi18n.T(s.loc(r), "auth.login_button"))
	s.render(w, r, pages.Login(c, r.URL.Query().Get("next")))
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, paths.Login(), http.StatusSeeOther)
}

func (s *Server) handleRegisterGet(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, paths.Login(), http.StatusSeeOther)
}

func (s *Server) handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, paths.Login(), http.StatusSeeOther)
}

func (s *Server) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if !s.github.Enabled() {
		http.Redirect(w, r, paths.LoginError("errors.GITHUB_NOT_CONFIGURED"), http.StatusSeeOther)
		return
	}
	next := safeNext(r.URL.Query().Get("next"))
	state, err := auth.MintOAuthState(next)
	if err != nil {
		http.Redirect(w, r, paths.LoginError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	redirectURI := githubCallbackURL(r)
	http.Redirect(w, r, s.github.AuthCodeURL(state, redirectURI), http.StatusSeeOther)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if !s.github.Enabled() {
		http.Redirect(w, r, paths.LoginError("errors.GITHUB_NOT_CONFIGURED"), http.StatusSeeOther)
		return
	}
	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, paths.LoginError("errors.GITHUB_OAUTH"), http.StatusSeeOther)
		return
	}
	next, err := auth.ParseOAuthState(r.URL.Query().Get("state"))
	if err != nil {
		http.Redirect(w, r, paths.LoginError("errors.GITHUB_OAUTH"), http.StatusSeeOther)
		return
	}
	id, err := s.github.Exchange(r.URL.Query().Get("code"), githubCallbackURL(r))
	if err != nil {
		http.Redirect(w, r, paths.LoginError("errors.GITHUB_OAUTH"), http.StatusSeeOther)
		return
	}
	token, err := auth.GenerateToken(fmt.Sprintf("%d", id.ID), id.Login)
	if err != nil {
		http.Redirect(w, r, paths.LoginError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	auth.SetAuthCookie(w, r, token)
	http.Redirect(w, r, safeNext(next), http.StatusSeeOther)
}

func githubCallbackURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + r.Host + paths.GitHubCallback()
}

func (s *Server) handleLogoutPage(w http.ResponseWriter, r *http.Request) {
	auth.ClearAuthCookie(w, r)
	http.Redirect(w, r, paths.Landing, http.StatusSeeOther)
}

func (s *Server) handleLang(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.Landing, http.StatusSeeOther)
		return
	}
	appi18n.SetLangCookie(w, r.FormValue("lang"))
	http.Redirect(w, r, safeNext(r.FormValue("next")), http.StatusSeeOther)
}

func safeNext(next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return paths.Projects()
	}
	u, err := url.Parse(next)
	if err != nil || u.IsAbs() || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return paths.Projects()
	}
	return u.RequestURI()
}

func errorID(err error) string {
	if coded, ok := apierrors.As(err); ok {
		return "errors." + string(coded.Code)
	}
	return "errors.UNKNOWN"
}
