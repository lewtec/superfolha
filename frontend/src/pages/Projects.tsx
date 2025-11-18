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
        <div className="flex justify-between items-center mb-6">
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
          <div className="alert alert-error mb-4">
            <span>{error}</span>
          </div>
        )}

        {queryData.loading ? (
          <div className="flex justify-center items-center h-64">
            <span className="loading loading-spinner loading-lg"></span>
          </div>
        ) : projects && projects.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-lg text-base-content/70 mb-4">No projects yet</p>
            <button
              className="btn btn-primary"
              onClick={() => setShowModal(true)}
              disabled={isCreatingProject}
            >
              Create your first project
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
                      Updated:{" "}
                      {new Date(project.updatedAt).toLocaleDateString()}
                    </p>
                    <div className="card-actions justify-end mt-4">
                      <button
                        className="btn btn-primary btn-sm"
                        onClick={() => navigate(`/editor/${project.id}`)}
                      >
                        Open
                      </button>
                      <button
                        className="btn btn-error btn-sm"
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
        <div className="modal modal-open">
          <div className="modal-box">
            <h3 className="font-bold text-lg mb-4">Create New Project</h3>
            <div className="form-control">
              <label className="label">
                <span className="label-text">Project Name</span>
              </label>
              <input
                type="text"
                placeholder="My LaTeX Document"
                className="input input-bordered"
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
