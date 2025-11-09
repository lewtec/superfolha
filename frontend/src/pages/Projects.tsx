import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

interface Project {
  id: string
  name: string
  createdAt: string
  updatedAt: string
}

export default function Projects() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [newProjectName, setNewProjectName] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    loadProjects()
  }, [])

  const loadProjects = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { 'Authorization': `Bearer ${token}` })
        },
        body: JSON.stringify({
          query: `
            query {
              projects {
                id
                name
                createdAt
                updatedAt
              }
            }
          `,
        }),
      })

      const data = await response.json()

      if (data.errors) {
        if (data.errors[0].message.includes('not authenticated')) {
          localStorage.removeItem('token')
          navigate('/login')
          return
        }
        setError(data.errors[0].message)
        return
      }

      setProjects(data.data.projects || [])
    } catch (err) {
      setError('Failed to load projects')
    } finally {
      setLoading(false)
    }
  }

  const createProject = async () => {
    if (!newProjectName.trim()) {
      setError('Project name is required')
      return
    }

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { 'Authorization': `Bearer ${token}` })
        },
        body: JSON.stringify({
          query: `
            mutation CreateProject($name: String!) {
              createProject(name: $name) {
                id
                name
                createdAt
                updatedAt
              }
            }
          `,
          variables: { name: newProjectName },
        }),
      })

      const data = await response.json()

      if (data.errors) {
        setError(data.errors[0].message)
        return
      }

      const newProject = data.data.createProject
      setProjects([newProject, ...projects])
      setShowModal(false)
      setNewProjectName('')
      navigate(`/editor/${newProject.id}`)
    } catch (err) {
      setError('Failed to create project')
    }
  }

  const deleteProject = async (id: string) => {
    if (!confirm('Are you sure you want to delete this project?')) {
      return
    }

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { 'Authorization': `Bearer ${token}` })
        },
        body: JSON.stringify({
          query: `
            mutation DeleteProject($id: ID!) {
              deleteProject(id: $id)
            }
          `,
          variables: { id },
        }),
      })

      const data = await response.json()

      if (data.errors) {
        setError(data.errors[0].message)
        return
      }

      setProjects(projects.filter(p => p.id !== id))
    } catch (err) {
      setError('Failed to delete project')
    }
  }

  const logout = () => {
    localStorage.removeItem('token')
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-base-200">
      {/* Navbar */}
      <div className="navbar bg-base-100 shadow-md">
        <div className="flex-1">
          <a className="btn btn-ghost normal-case text-xl">LaTeX Editor</a>
        </div>
        <div className="flex-none">
          <button className="btn btn-ghost" onClick={logout}>
            Logout
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto p-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold">My Projects</h1>
          <button className="btn btn-primary" onClick={() => setShowModal(true)}>
            + New Project
          </button>
        </div>

        {error && (
          <div className="alert alert-error mb-4">
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <span className="loading loading-spinner loading-lg"></span>
          </div>
        ) : projects.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-lg text-base-content/70 mb-4">No projects yet</p>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              Create your first project
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {projects.map((project) => (
              <div key={project.id} className="card bg-base-100 shadow-xl">
                <div className="card-body">
                  <h2 className="card-title">{project.name}</h2>
                  <p className="text-sm text-base-content/70">
                    Updated: {new Date(project.updatedAt).toLocaleDateString()}
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
                      onClick={() => deleteProject(project.id)}
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
                onKeyPress={(e) => e.key === 'Enter' && createProject()}
              />
            </div>
            <div className="modal-action">
              <button className="btn" onClick={() => {
                setShowModal(false)
                setNewProjectName('')
              }}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={createProject}>
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
