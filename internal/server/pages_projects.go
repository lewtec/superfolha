package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lewtec/superfolha/internal/auth"
	appi18n "github.com/lewtec/superfolha/internal/i18n"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/remote"
	"github.com/lewtec/superfolha/internal/session"
	"github.com/lewtec/superfolha/internal/ui/pages"
)

func (s *Server) handleProjectsGet(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	items := s.hubs.ListFor(user.Email)
	c := s.chrome(r, appi18n.T(s.loc(r), "sessions.title"))
	s.render(w, r, pages.Sessions(c, items))
}

func (s *Server) handleProjectsPost(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	raw := strings.TrimSpace(r.FormValue("remote"))
	branch := strings.TrimSpace(r.FormValue("branch"))
	if branch == "" {
		branch = "main"
	}
	pub := strings.TrimSpace(r.FormValue("ssh_public"))
	if err := remote.Validate(raw); err != nil {
		http.Redirect(w, r, paths.ProjectsError("sessions.remote_required"), http.StatusSeeOther)
		return
	}
	if _, err := s.hubs.Create(user.Email, raw, branch, pub); err != nil {
		if errors.Is(err, session.ErrAlreadyLive) {
			http.Redirect(w, r, paths.ProjectsError("sessions.already_live"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, paths.ProjectsError("sessions.ssh_seed_invalid"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.ProjectsFlash("sessions.add_ssh_key"), http.StatusSeeOther)
}

func (s *Server) handleSessionRetry(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, paths.ProjectsFlash("sessions.add_ssh_key"), http.StatusSeeOther)
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	id := r.PathValue(paths.ParamID)
	if err := s.hubs.End(id, user.Email); err != nil {
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.ProjectsFlash("sessions.ended"), http.StatusSeeOther)
}

func (s *Server) handleSessionPreauth(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	id := r.PathValue(paths.ParamID)
	info, ok := s.hubs.Live(id)
	if !ok || info.HostLogin != user.Email {
		http.Redirect(w, r, paths.ProjectsError("errors.UNAUTHORIZED"), http.StatusSeeOther)
		return
	}
	tok, err := session.MintPreauth(id)
	if err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.Editor(id)+"?preauth="+tok+"&flash=sessions.preauth_ready", http.StatusSeeOther)
}

func (s *Server) handleSessionKnock(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	id := r.PathValue(paths.ParamID)
	if err := s.hubs.Knock(id, user.Email); err != nil {
		http.Redirect(w, r, paths.ProjectsError("sessions.knock_closed"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.ProjectsFlash("sessions.knock_sent"), http.StatusSeeOther)
}

func (s *Server) handleSessionAdmit(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	id := r.PathValue(paths.ParamID)
	if err := s.hubs.Admit(id, user.Email, strings.TrimSpace(r.FormValue("login"))); err != nil {
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.Editor(id), http.StatusSeeOther)
}

func (s *Server) handleSessionKick(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	id := r.PathValue(paths.ParamID)
	if err := s.hubs.KickAll(id, user.Email); err != nil {
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.Editor(id), http.StatusSeeOther)
}

func (s *Server) handleSessionKnockMode(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	id := r.PathValue(paths.ParamID)
	on := r.FormValue("on") == "1"
	if err := s.hubs.SetKnock(id, user.Email, on); err != nil {
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.Editor(id), http.StatusSeeOther)
}

func (s *Server) handleEditorGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue(paths.ParamID)
	user, _ := auth.GetUserFromContext(r.Context())
	if tok := r.URL.Query().Get("preauth"); tok != "" {
		if err := s.hubs.RedeemPreauth(id, user.Email, tok); err != nil {
			http.Redirect(w, r, paths.ProjectsError("sessions.preauth_invalid"), http.StatusSeeOther)
			return
		}
	}
	info, ok := s.hubs.Live(id)
	if !ok {
		http.Redirect(w, r, paths.ProjectsError("errors.PROJECT_NOT_FOUND"), http.StatusSeeOther)
		return
	}
	if !s.hubs.CanOpen(id, user.Email) {
		http.Redirect(w, r, paths.ProjectsError("errors.UNAUTHORIZED"), http.StatusSeeOther)
		return
	}
	if !info.Ready {
		http.Redirect(w, r, paths.ProjectsFlash("sessions.add_ssh_key"), http.StatusSeeOther)
		return
	}
	c := s.chrome(r, info.Remote)
	s.render(w, r, pages.Editor(c, info.ID, info.CloneURL, info.Branch, user.Email, info.SSHPublic, pages.MarshalI18n(appi18n.Map(s.bundle, s.lang(r)))))
}
