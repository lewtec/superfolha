import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import Editor from '../components/Editor'
import FileTree from '../components/FileTree'
import PDFViewer from '../components/PDFViewer'
import Navbar from '../components/Navbar'
import LogsPanel from '../components/LogsPanel'

interface File {
  path: string
  content: string
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [project, setProject] = useState<any>(null)
  const [files, setFiles] = useState<File[]>([])
  const [currentFile, setCurrentFile] = useState<File | null>(null)
  const [view, setView] = useState<'code' | 'pdf'>('code')
  const [pdfData, setPdfData] = useState<string | null>(null)
  const [logs, setLogs] = useState('')
  const [compileSuccess, setCompileSuccess] = useState(false)
  const [compiling, setCompiling] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadProject()
    loadFiles()
  }, [id])

  const loadProject = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: JSON.stringify({
          query: `
            query GetProject($id: ID!) {
              project(id: $id) {
                id
                name
              }
            }
          `,
          variables: { id },
        }),
      })

      const data = await response.json()
      if (data.errors) {
        alert(data.errors[0].message)
        navigate('/projects')
        return
      }

      setProject(data.data.project)
    } catch (err) {
      console.error(err)
    }
  }

  const loadFiles = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: JSON.stringify({
          query: `
            query GetFiles($projectId: ID!) {
              files(projectId: $projectId) {
                path
                content
              }
            }
          `,
          variables: { projectId: id },
        }),
      })

      const data = await response.json()
      if (data.errors) {
        alert(data.errors[0].message)
        return
      }

      const loadedFiles = data.data.files || []
      setFiles(loadedFiles)
      if (loadedFiles.length > 0) {
        setCurrentFile(loadedFiles[0])
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const saveFile = async () => {
    if (!currentFile) return

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: JSON.stringify({
          query: `
            mutation SaveFile($projectId: ID!, $path: String!, $content: String!) {
              saveFile(projectId: $projectId, path: $path, content: $content) {
                path
                content
              }
            }
          `,
          variables: {
            projectId: id,
            path: currentFile.path,
            content: currentFile.content,
          },
        }),
      })

      const data = await response.json()
      if (data.errors) {
        alert(data.errors[0].message)
      }
    } catch (err) {
      alert('Failed to save file')
    }
  }

  const compile = async () => {
    setCompiling(true)
    try {
      // Create tarball from files
      const formData = new FormData()
      const tarball = await createTarball(files)
      formData.append('tarball', new Blob([tarball]), 'project.tar.gz')

      const response = await fetch('/api/compile', {
        method: 'POST',
        body: formData,
      })

      const result = await response.json()
      setPdfData(result.pdf)
      setLogs(result.logs)
      setCompileSuccess(result.success)

      if (result.success) {
        setView('pdf')
      }
    } catch (err) {
      alert('Compilation failed')
    } finally {
      setCompiling(false)
    }
  }

  const createTarball = async (files: File[]): Promise<Uint8Array> => {
    // Simple tarball creation - in production use a proper library
    const encoder = new TextEncoder()
    const parts: Uint8Array[] = []

    for (const file of files) {
      const content = encoder.encode(file.content)
      parts.push(content)
    }

    // This is a simplified version - proper implementation would create actual tar.gz
    return new Uint8Array(parts.flatMap(p => Array.from(p)))
  }

  const handleFileSelect = (path: string) => {
    const file = files.find(f => f.path === path)
    if (file) {
      setCurrentFile(file)
      setView('code')
    }
  }

  const handleNewFile = () => {
    const filename = prompt('Enter file name (e.g., chapter1.tex):')
    if (!filename) return

    const newFile: File = { path: filename, content: '' }
    setFiles([...files, newFile])
    setCurrentFile(newFile)
  }

  const handleDeleteFile = async (path: string) => {
    if (!confirm(`Delete ${path}?`)) return

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: JSON.stringify({
          query: `
            mutation DeleteFile($projectId: ID!, $path: String!) {
              deleteFile(projectId: $projectId, path: $path)
            }
          `,
          variables: { projectId: id, path },
        }),
      })

      const data = await response.json()
      if (data.errors) {
        alert(data.errors[0].message)
        return
      }

      const newFiles = files.filter(f => f.path !== path)
      setFiles(newFiles)
      if (currentFile?.path === path && newFiles.length > 0) {
        setCurrentFile(newFiles[0])
      }
    } catch (err) {
      alert('Failed to delete file')
    }
  }

  const handleEditorChange = (value: string) => {
    if (currentFile) {
      setCurrentFile({ ...currentFile, content: value })
      setFiles(files.map(f => f.path === currentFile.path ? { ...currentFile, content: value } : f))
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="loading loading-spinner loading-lg"></span>
      </div>
    )
  }

  return (
    <div className="h-screen flex flex-col">
      <div className="flex flex-1 overflow-hidden">
        {/* File Tree Sidebar */}
        <div className="w-64 border-r border-base-300">
          <FileTree
            files={files}
            currentFile={currentFile?.path || null}
            onFileSelect={handleFileSelect}
            onNewFile={handleNewFile}
            onDeleteFile={handleDeleteFile}
          />
        </div>

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col">
          {/* Toolbar */}
          <div className="tabs tabs-boxed bg-base-200 p-2">
            <a
              className={`tab ${view === 'code' ? 'tab-active' : ''}`}
              onClick={() => setView('code')}
            >
              Code
            </a>
            <a
              className={`tab ${view === 'pdf' ? 'tab-active' : ''}`}
              onClick={() => setView('pdf')}
            >
              PDF
            </a>
          </div>

          {/* Editor/PDF View */}
          <div className="flex-1 overflow-hidden">
            {view === 'code' && currentFile ? (
              <Editor
                value={currentFile.content}
                onChange={handleEditorChange}
                onSave={saveFile}
              />
            ) : (
              <PDFViewer pdfData={pdfData} />
            )}
          </div>

          {/* Logs Panel */}
          <LogsPanel logs={logs} success={compileSuccess} />
        </div>
      </div>
    </div>
  )
}
