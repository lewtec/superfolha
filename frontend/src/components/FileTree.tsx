import { useState, useRef } from "react";
import { buildFileTree, FileTreeNode } from "../utils/fileTree";
import {
  Folder,
  File as FileIcon,
  ChevronRight,
  ChevronDown,
  Trash2,
  Upload,
} from "feather-icons-react";

interface File {
  path: string;
  content: string;
  isDirty: boolean; // Added isDirty
}

interface FileTreeProps {
  files: File[];
  currentFile: string | null;
  onFileSelect: (path: string) => void;
  onNewFile: () => void;
  onDeleteFile: (path: string) => void;
  onLoadFile: (fileName: string, content: string | null) => void; // content can be null for binary files
  projectId: string; // New prop
}

interface FileTreeItemProps {
  node: FileTreeNode;
  currentFile: string | null;
  onFileSelect: (path: string) => void;
  onDeleteFile: (path: string) => void;
  level: number;
}

const FileTreeItem: React.FC<FileTreeItemProps> = ({
  node,
  currentFile,
  onFileSelect,
  onDeleteFile,
  level,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleToggleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (node.type === "directory") {
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
          {node.type === "directory" ? (
            isExpanded ? (
              <ChevronDown size={16} />
            ) : (
              <ChevronRight size={16} />
            )
          ) : (
            <div style={{ width: "16px" }}></div> // Spacer for file icon alignment
          )}
          {node.type === "directory" ? (
            <Folder size={16} />
          ) : (
            <FileIcon size={16} />
          )}
          <span
            className={
              currentFile === node.path && node.type === "file"
                ? "font-bold text-primary"
                : ""
            }
          >
            {node.name}{" "}
            {node.isDirty && <span className="text-warning">*</span>}
          </span>
        </div>
        {node.type === "file" && node.path !== "main.tex" && (
          <button className="btn btn-xs btn-ghost" onClick={handleDelete}>
            <Trash2 size={16} />
          </button>
        )}
      </div>
      {node.type === "directory" && isExpanded && node.children && (
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

export default function FileTree({
  files,
  currentFile,
  onFileSelect,
  onNewFile,
  onDeleteFile,
  onLoadFile,
  projectId,
}: FileTreeProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const filePaths = files.map((file) => ({
    path: file.path,
    isDirty: file.isDirty,
  }));
  const tree = buildFileTree(filePaths);

  const handleFileChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append("file", file);

    try {
      const response = await fetch(`/api/projects/${projectId}/upload-file`, {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Failed to upload file");
      }

      const result = await response.json();
      const uploadedFilePath = result.path;

      // For text files, read content on frontend to update editor immediately
      // For binary files, content will be null, and editor won't display it
      const isTextFile =
        file.type.startsWith("text/") ||
        file.name.endsWith(".tex") ||
        file.name.endsWith(".md");
      let fileContent: string | null = null;

      if (isTextFile) {
        fileContent = await file.text();
      }

      onLoadFile(uploadedFilePath, fileContent);
    } catch (error) {
      console.error("Error uploading file:", error);
      alert(
        `Error uploading file: ${error instanceof Error ? error.message : String(error)}`,
      );
    } finally {
      // Clear the input so the same file can be selected again
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const triggerFileInput = () => {
    fileInputRef.current?.click();
  };

  return (
    <div className="file-tree bg-base-200 p-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="font-bold">Files</h3>
        <div className="flex gap-2">
          <button className="btn btn-xs btn-primary" onClick={triggerFileInput}>
            <Upload size={16} /> Load
          </button>
          <button className="btn btn-xs btn-primary" onClick={onNewFile}>
            + New
          </button>
        </div>
      </div>
      <input
        type="file"
        ref={fileInputRef}
        className="hidden"
        onChange={handleFileChange}
        accept="*/*" // Accept all file types
      />
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
