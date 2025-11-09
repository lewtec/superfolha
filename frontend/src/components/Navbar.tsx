import { useNavigate } from 'react-router-dom'

interface NavbarProps {
  projectName: string
  onCompile: () => void
  compiling: boolean
}

export default function Navbar({ projectName, onCompile, compiling }: NavbarProps) {
  const navigate = useNavigate()

  return (
    <div className="navbar bg-base-100 shadow-md">
      <div className="flex-1">
        <button className="btn btn-ghost normal-case text-xl" onClick={() => navigate('/projects')}>
          LaTeX Editor
        </button>
        <span className="text-base-content/70 ml-4">/ {projectName}</span>
      </div>
      <div className="flex-none gap-2">
        <button
          className={`btn btn-primary ${compiling ? 'loading' : ''}`}
          onClick={onCompile}
          disabled={compiling}
        >
          {compiling ? 'Compiling...' : 'Compile'}
        </button>
      </div>
    </div>
  )
}
