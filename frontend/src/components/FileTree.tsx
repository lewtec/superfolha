import { useState } from 'react';
import { buildFileTree, FileTreeNode } from '../utils/fileTree';
import { Folder, FileText, ChevronRight, ChevronDown, Trash2 } from 'feather-icons-react';

interface File {
  path: string;
  content: string;
}

interface FileTreeProps {
  files: File[];
  currentFile: string | null;
  onFileSelect: (path: string) => void;
  onNewFile: () => void;
  onDeleteFile: (path: string) => void;
}

interface FileTreeItemProps {
  node: FileTreeNode;
  currentFile: string | null;
  onFileSelect: (path: string) => void;
  onDeleteFile: (path: string) => void;
  level: number;
}

const FileTreeItem: React.FC<FileTreeItemProps> = ({ node, currentFile, onFileSelect, onDeleteFile, level }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleToggleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (node.type === 'directory') {
      setIsExpanded(!isExpanded);
    } else {
      onFileSelect(node.path);
    }
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDeleteFile(node.path);
  };

  const indent = level * 16; // 16px per level

  return (
    <li>
      <div
        className="flex items-center justify-between py-1 px-2 cursor-pointer hover:bg-base-300 rounded-md"
        style={{ paddingLeft: `${indent}px` }}
        onClick={handleToggleExpand}
      >
        <div className="flex items-center gap-2 flex-grow">
          {node.type === 'directory' ? (
            isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />
          ) : (
            <div style={{ width: '16px' }}></div> // Spacer for file icon alignment
          )}
          {node.type === 'directory' ? <Folder size={16} /> : <FileText size={16} />}
          <span className={currentFile === node.path && node.type === 'file' ? 'font-bold text-primary' : ''}>
            {node.name}
          </span>
        </div>
        {node.type === 'file' && node.path !== 'main.tex' && (
          <button className="btn btn-xs btn-ghost" onClick={handleDelete}>
            <Trash2 size={16} />
          </button>
        )}
      </div>
      {node.type === 'directory' && isExpanded && node.children && (
        <ul className="pl-4">
          {node.children.map((childNode) => (
            <FileTreeItem
              key={childNode.path}
              node={childNode}
              currentFile={currentFile}
              onFileSelect={onFileSelect}
              onDeleteFile={onDeleteFile}
              level={level + 1}
            />
          ))}
        </ul>
      )}
    </li>
  );
};

export default function FileTree({ files, currentFile, onFileSelect, onNewFile, onDeleteFile }: FileTreeProps) {
  const filePaths = files.map(file => file.path);
  const tree = buildFileTree(filePaths);

  return (
    <div className="file-tree bg-base-200 p-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="font-bold">Files</h3>
        <button className="btn btn-xs btn-primary" onClick={onNewFile}>
          + New
        </button>
      </div>
      <ul className="menu bg-base-100 rounded-box w-full">
        {tree.map((node) => (
          <FileTreeItem
            key={node.path}
            node={node}
            currentFile={currentFile}
            onFileSelect={onFileSelect}
            onDeleteFile={onDeleteFile}
            level={0}
          />
        ))}
      </ul>
    </div>
  );
}
