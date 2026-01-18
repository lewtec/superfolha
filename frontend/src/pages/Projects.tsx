import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useProjectsQuery } from "../hooks/useProjectsQuery";
import { useCreateProjectMutation } from "../hooks/useCreateProjectMutation";
import { useDeleteProjectMutation } from "../hooks/useDeleteProjectMutation";

export default function Projects() {
  const navigate = useNavigate();
  const [newProjectName, setNewProjectName] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [error, setError] = useState("");

  const { projects, ...queryData } = useProjectsQuery();

  const { createProject, isInFlight: isCreatingProject } =
    useCreateProjectMutation({
      onCompleted: (response, errors) => {
        if (errors) {
          setError(errors[0].message);
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
        setError("Failed to create project");
        console.error(err);
      },
    });

  const { deleteProject, isInFlight: isDeletingProject } =
    useDeleteProjectMutation({
      onCompleted: (response, errors) => {
        if (errors) {
          setError(errors[0].message);
          return;
        }
      },
      onError: (err) => {
        setError("Failed to delete project");
        console.error(err);
      },
      updater: (store, response) => {
        if (response?.deleteProject?.id) {
          const deletedId = response.deleteProject.id;
          const root = store.getRoot();
          const projects = root.getLinkedRecords("projects");
          if (projects) {
            const newProjects = projects.filter(
              (project) => project.getValue("id") !== deletedId,
            );
            root.setLinkedRecords(newProjects, "projects");
          }
        }
      },
    });

  const handleCreateProject = () => {
    if (!newProjectName.trim()) {
      setError("Project name is required");
      return;
    }
    setError("");
    createProject(newProjectName);
  };

  const handleDeleteProject = (id: string) => {
    if (!confirm("Are you sure you want to delete this project?")) {
      return;
    }
    setError("");
    deleteProject(id);
  };

  return (
    <div className="min-h-screen bg-base-200">
      {/* Content */}
      <div className="container mx-auto p-8">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-3xl font-bold">My Projects</h1>
          <button
            className="btn btn-primary"
            onClick={() => setShowModal(true)}
            disabled={isCreatingProject}
          >
            + New Project
          </button>
        </div>

        {error && (
          <div className="mb-4 alert alert-error">
            <span>{error}</span>
          </div>
        )}

        {queryData.loading ? (
          <div className="flex h-64 items-center justify-center">
            <span className="loading loading-lg loading-spinner"></span>
          </div>
        ) : projects && projects.length === 0 ? (
          <div className="py-16 text-center">
            <p className="mb-4 text-lg text-base-content/70">No projects yet</p>
            <button
              className="btn btn-primary"
              onClick={() => setShowModal(true)}
              disabled={isCreatingProject}
            >
              Create your first project
            </button>
          </div>
        ) : (
          <div
            className="
              grid grid-cols-1 gap-6
              md:grid-cols-2
              lg:grid-cols-3
            "
          >
            {projects &&
              projects.map((project) => (
                <div key={project.id} className="card bg-base-100 shadow-xl">
                  <div className="card-body">
                    <h2 className="card-title">{project.name}</h2>
                    <p className="text-sm text-base-content/70">
                      Updated:{" "}
                      {new Date(project.updatedAt).toLocaleDateString()}
                    </p>
                    <div className="mt-4 card-actions justify-end">
                      <button
                        className="btn btn-sm btn-primary"
                        onClick={() => navigate(`/editor/${project.id}`)}
                      >
                        Open
                      </button>
                      <button
                        className="btn btn-sm btn-error"
                        onClick={() => handleDeleteProject(project.id)}
                        disabled={isDeletingProject}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              ))}
          </div>
        )}
      </div>

      {/* Create Project Modal */}
      {showModal && (
        <div className="modal-open modal">
          <div className="modal-box">
            <h3 className="mb-4 text-lg font-bold">Create New Project</h3>
            <div className="form-control">
              <label className="label">
                <span className="label-text">Project Name</span>
              </label>
              <input
                type="text"
                placeholder="My LaTeX Document"
                className="input-bordered input"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                onKeyPress={(e) => e.key === "Enter" && handleCreateProject()}
                disabled={isCreatingProject}
              />
            </div>
            <div className="modal-action">
              <button
                className="btn"
                onClick={() => {
                  setShowModal(false);
                  setNewProjectName("");
                }}
              >
                Cancel
              </button>
              <button
                className="btn btn-primary"
                onClick={handleCreateProject}
                disabled={isCreatingProject}
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
