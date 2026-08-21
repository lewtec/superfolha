package server

import (
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
		// Allow either a raw message or an errors.* / auth.* message id.
		if translated := appi18n.T(loc, c.Error); translated != c.Error {
			c.Error = translated
		} else if translated := appi18n.T(loc, "errors."+c.Error); translated != "errors."+c.Error {
			c.Error = translated
		}
	}
	return c
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
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.LoginError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	resp, err := s.authService.Login(r.Context(), email, password)
	if err != nil {
		http.Redirect(w, r, paths.LoginError(errorID(err)), http.StatusSeeOther)
		return
	}
	auth.SetAuthCookie(w, resp.Token)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleRegisterGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.GetUserFromContext(r.Context()); ok {
		http.Redirect(w, r, paths.Projects(), http.StatusSeeOther)
		return
	}
	c := s.chrome(r, appi18n.T(s.loc(r), "auth.register_button"))
	s.render(w, r, pages.Register(c))
}

func (s *Server) handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.RegisterError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")
	if password != confirm {
		http.Redirect(w, r, paths.RegisterError("auth.passwords_do_not_match"), http.StatusSeeOther)
		return
	}
	resp, err := s.authService.Register(r.Context(), email, password)
	if err != nil {
		http.Redirect(w, r, paths.RegisterError(errorID(err)), http.StatusSeeOther)
		return
	}
	auth.SetAuthCookie(w, resp.Token)
	http.Redirect(w, r, paths.Projects(), http.StatusSeeOther)
}

func (s *Server) handleLogoutPage(w http.ResponseWriter, r *http.Request) {
	auth.ClearAuthCookie(w)
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
