package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	appi18n "github.com/lewtec/superfolha/internal/i18n"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/ui/pages"
	"os"
)

func (s *Server) handleProjectsGet(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	items, err := s.repo.GetUserProjects(r.Context(), user.UserID)
	if err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	loc := s.loc(r)
	updated := make([]string, len(items))
	for i, p := range items {
		updated[i] = appi18n.TData(loc, "projects.updated", map[string]any{
			"Date": p.UpdatedAt.Format("2006-01-02"),
		})
	}
	c := s.chrome(r, appi18n.T(loc, "projects.title"))
	s.render(w, r, pages.Projects(c, items, updated))
}

func (s *Server) handleProjectsPost(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INVALID_INPUT"), http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, paths.ProjectsError("projects.name_required"), http.StatusSeeOther)
		return
	}
	projectUUID, err := uuid.NewV7()
	if err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	projectID := projectUUID.String()
	projectPath := s.projectService.GetProjectPath(projectID)
	_, err = s.repo.CreateProject(r.Context(), db.CreateProjectParams{
		ID:      projectID,
		UserID:  user.UserID,
		Name:    name,
		GitPath: projectPath,
	})
	if err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.Editor(projectID), http.StatusSeeOther)
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, projectPath, _, err := s.resolver.getAndCheckProject(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	if s.hubs != nil {
		s.hubs.CloseProject(project.ID)
	}
	if err := s.repo.DeleteProject(r.Context(), project.ID); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	if err := os.RemoveAll(projectPath); err != nil {
		http.Redirect(w, r, paths.ProjectsError("errors.INTERNAL"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paths.ProjectsFlash("ok"), http.StatusSeeOther)
}

func (s *Server) handleEditorGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, _, user, err := s.resolver.getAndCheckProject(r.Context(), id)
	if err != nil {
		if coded, ok := apierrors.As(err); ok && coded.Code == apierrors.CodeUnauthenticated {
			http.Redirect(w, r, paths.LoginNext(r.URL.RequestURI()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, paths.ProjectsError(errorID(err)), http.StatusSeeOther)
		return
	}
	c := s.chrome(r, project.Name)
	s.render(w, r, pages.Editor(c, project.ID, project.Name, user.Email, pages.MarshalI18n(appi18n.Map(s.bundle, s.lang(r)))))
}
