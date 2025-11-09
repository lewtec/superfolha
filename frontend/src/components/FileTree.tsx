interface File {
  path: string
  content: string
}

interface FileTreeProps {
  files: File[]
  currentFile: string | null
  onFileSelect: (path: string) => void
  onNewFile: () => void
  onDeleteFile: (path: string) => void
}

export default function FileTree({ files, currentFile, onFileSelect, onNewFile, onDeleteFile }: FileTreeProps) {
  return (
    <div className="file-tree bg-base-200 p-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="font-bold">Files</h3>
        <button className="btn btn-xs btn-primary" onClick={onNewFile}>
          + New
        </button>
      </div>
      <ul className="menu bg-base-100 rounded-box">
        {files.map((file) => (
          <li key={file.path}>
            <div className="flex justify-between items-center">
              <a
                className={currentFile === file.path ? 'active' : ''}
                onClick={() => onFileSelect(file.path)}
              >
                {file.path}
              </a>
              {file.path !== 'main.tex' && (
                <button
                  className="btn btn-xs btn-ghost"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDeleteFile(file.path)
                  }}
                >
                  ×
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
