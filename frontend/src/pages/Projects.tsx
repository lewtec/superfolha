import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useProjectsQuery } from "../hooks/useProjectsQuery";
import { useCreateProjectMutation } from "../hooks/useCreateProjectMutation";
import { useDeleteProjectMutation } from "../hooks/useDeleteProjectMutation";
import {
  translateError,
  translateGraphQLErrors,
} from "../i18n/translateError";

export default function Projects() {
  const navigate = useNavigate();
  const { t, i18n } = useTranslation(["projects", "common", "errors"]);
  const [newProjectName, setNewProjectName] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [error, setError] = useState("");

  const { projects } = useProjectsQuery();

  const { createProject, isInFlight: isCreatingProject } =
    useCreateProjectMutation({
      onCompleted: (response, errors) => {
        if (errors) {
          setError(translateGraphQLErrors(t, errors));
          return;
        }
        const newProject = response.createProject;
        if (newProject) {
          setShowModal(false);
          setNewProjectName("");
          navigate(`/editor/${newProject.id}`);
        }
      },
      onError: (err) => {
        setError(translateError(t, err));
        console.error(err);
      },
    });

  const { deleteProject, isInFlight: isDeletingProject } =
    useDeleteProjectMutation({
      onCompleted: (_response, errors) => {
        if (errors) {
          setError(translateGraphQLErrors(t, errors));
          return;
        }
      },
      onError: (err) => {
        setError(translateError(t, err));
        console.error(err);
      },
      updater: (store, response) => {
        if (response?.deleteProject?.id) {
          const deletedId = response.deleteProject.id;
          const root = store.getRoot();
          const projectsRec = root.getLinkedRecords("projects");
          if (projectsRec) {
            const newProjects = projectsRec.filter(
              (project) => project.getValue("id") !== deletedId,
            );
            root.setLinkedRecords(newProjects, "projects");
          }
        }
      },
    });

  const handleCreateProject = () => {
    if (!newProjectName.trim()) {
      setError(t("projects:name_required"));
      return;
    }
    setError("");
    createProject(newProjectName);
  };

  const handleDeleteProject = (id: string) => {
    if (!confirm(t("projects:delete_confirm"))) {
      return;
    }
    setError("");
    deleteProject(id);
  };

  return (
    <div className="page-fill bg-base-200">
      <div className="container mx-auto page-pad py-8">
        <div className="flex justify-between items-center mb-6 gap-2 flex-wrap">
          <h1 className="text-3xl font-bold">{t("projects:title")}</h1>
          <button
            className="btn btn-primary min-h-[var(--touch-min)]"
            onClick={() => setShowModal(true)}
            disabled={isCreatingProject}
          >
            {t("projects:new_project")}
          </button>
        </div>

        {error && (
          <div className="alert alert-error mb-4">
            <span>{error}</span>
          </div>
        )}

        {projects && projects.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-lg text-base-content/70 mb-4">
              {t("projects:empty")}
            </p>
            <button
              className="btn btn-primary min-h-[var(--touch-min)]"
              onClick={() => setShowModal(true)}
              disabled={isCreatingProject}
            >
              {t("projects:create_first")}
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {projects &&
              projects.map((project) => (
                <div key={project.id} className="card bg-base-100 shadow-xl">
                  <div className="card-body">
                    <h2 className="card-title">{project.name}</h2>
                    <p className="text-sm text-base-content/70">
                      {t("projects:updated", {
                        date: new Date(project.updatedAt).toLocaleDateString(
                          i18n.language,
                        ),
                      })}
                    </p>
                    <div className="card-actions justify-end mt-4">
                      <button
                        className="btn btn-primary btn-sm min-h-[var(--touch-min)]"
                        onClick={() => navigate(`/editor/${project.id}`)}
                      >
                        {t("common:open")}
                      </button>
                      <button
                        className="btn btn-error btn-sm min-h-[var(--touch-min)]"
                        onClick={() => handleDeleteProject(project.id)}
                        disabled={isDeletingProject}
                      >
                        {t("common:delete")}
                      </button>
                    </div>
                  </div>
                </div>
              ))}
          </div>
        )}
      </div>

      {showModal && (
        <div className="modal modal-open">
          <div className="modal-box">
            <h3 className="font-bold text-lg mb-4">
              {t("projects:create_modal_title")}
            </h3>
            <div className="form-control">
              <label className="label">
                <span className="label-text">{t("projects:name_label")}</span>
              </label>
              <input
                type="text"
                placeholder={t("projects:name_placeholder")}
                className="input input-bordered"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreateProject()}
                disabled={isCreatingProject}
              />
            </div>
            <div className="modal-action">
              <button
                className="btn min-h-[var(--touch-min)]"
                onClick={() => {
                  setShowModal(false);
                  setNewProjectName("");
                }}
              >
                {t("common:cancel")}
              </button>
              <button
                className="btn btn-primary min-h-[var(--touch-min)]"
                onClick={handleCreateProject}
                disabled={isCreatingProject}
              >
                {t("common:create")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
