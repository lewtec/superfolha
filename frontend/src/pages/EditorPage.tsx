import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Editor from "../components/Editor";
import FileTree from "../components/FileTree";
import PDFViewer from "../components/PDFViewer";
import Navbar from "../components/Navbar"; // Re-added Navbar import
import LogsPanel from "../components/LogsPanel";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useGetFilesQuery } from "../hooks/useGetFilesQuery";
import { useSaveFileMutation } from "../hooks/useSaveFileMutation";
import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
// import { EditorProvider } from "../contexts/EditorContext"; // Removed EditorProvider import

interface File {
  path: string;
  content: string;
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate(); // Moved up

  // Query hooks must be called unconditionally
  const { project, ...projectQueryData } = useGetProjectQuery({ id: id! });
  const { files: fetchedFiles, ...filesQueryData } = useGetFilesQuery({
    projectId: id!,
  });

  // Conditional return based on loading state - MUST be after all hooks are called
  if (projectQueryData.loading || filesQueryData.loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="loading loading-spinner loading-lg"></span>
      </div>
    );
  }

  // All other hooks and state declarations can follow
  const [files, setFiles] = useState<File[]>([]);
  const [currentFile, setCurrentFile] = useState<File | null>(null);
  const [view, setView] = useState<"code" | "pdf">("code");
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [logs, setLogs] = useState("");
  const [compileSuccess, setCompileSuccess] = useState(false);
  const [compiling, setCompiling] = useState(false);
  const [editorStatus, setEditorStatus] = useState<'clean' | 'dirty' | 'saving' | 'saved' | 'error'>('clean');

  const { saveFile, isInFlight: isSavingFile } = useSaveFileMutation(); // No config here anymore

  const { deleteFile, isInFlight: isDeletingFile } = useDeleteFileMutation({
    onCompleted: (response, errors) => {
      if (errors) {
        alert(errors[0].message);
        return;
      }
      const newFiles = files.filter((f) => f.path !== currentFile?.path); // Use currentFile?.path here
      setFiles(newFiles);
      if (currentFile?.path && newFiles.length > 0) {
        setCurrentFile(newFiles[0]);
      } else if (newFiles.length === 0) {
        setCurrentFile(null);
      }
    },
    onError: (err) => {
      alert("Failed to delete file");
      console.error(err);
    },
  });

  useEffect(() => {
    if (fetchedFiles) {
      setFiles(fetchedFiles as File[]);
      if (fetchedFiles.length > 0) {
        setCurrentFile(fetchedFiles[0] as File);
      }
    }
  }, [fetchedFiles]);

  const compile = async () => {
    setCompiling(true);
    try {
      // Create tarball from files
      const formData = new FormData();
      const tarball = await createTarball(files);
      formData.append("tarball", new Blob([tarball]), "project.tar.gz");

      const response = await fetch("/api/compile", {
        method: "POST",
        body: formData,
      });

      const result = await response.json();
      setPdfData(result.pdf);
      setLogs(result.logs);
      setCompileSuccess(result.success);

      if (result.success) {
        setView("pdf");
      }
    } catch (err) {
      alert("Compilation failed");
    } finally {
      setCompiling(false);
    }
  };

  const createTarball = async (files: File[]): Promise<Uint8Array> => {
    // Simple tarball creation - in production use a proper library
    const encoder = new TextEncoder();
    const parts: Uint8Array[] = [];

    for (const file of files) {
      const content = encoder.encode(file.content);
      parts.push(content);
    }

    // This is a simplified version - proper implementation would create actual tar.gz
    return new Uint8Array(parts.flatMap((p) => Array.from(p)));
  };

  const handleFileSelect = (path: string) => {
    const file = files.find((f) => f.path === path);
    if (file) {
      setCurrentFile(file);
      setView("code");
    }
  };

  const handleNewFile = () => {
    const filename = prompt("Enter file name (e.g., chapter1.tex):");
    if (!filename) return;

    const newFile: File = { path: filename, content: "" };
    setFiles([...files, newFile]);
    setCurrentFile(newFile);
  };

  const handleEditorChange = useCallback(
    (value: string) => {
      if (currentFile) {
        setCurrentFile({ ...currentFile, content: value });
        setFiles(
          files.map((f) =>
            f.path === currentFile.path
              ? { ...currentFile, content: value }
              : f,
          ),
        );
        setEditorStatus('dirty'); // Mark as dirty
      }
    },
    [currentFile, files],
  ); // Dependencies for useCallback

  const memoizedOnSave = useCallback((content: string) => { // Accepts content now
    if (currentFile) {
      setEditorStatus('saving'); // Mark as saving
      saveFile(id!, currentFile.path, content, { // Pass the received content and callbacks
        onCompleted: (response, errors) => {
          if (errors) {
            setEditorStatus('error');
            alert(errors[0].message);
            return;
          }
          setEditorStatus('saved'); // Mark as saved
        },
        onError: (err) => {
          setEditorStatus('error');
          alert('Failed to save file');
          console.error(err);
        },
      });
    }
  }, [id, currentFile, saveFile]);

  const memoizedOnDeleteFile = useCallback(
    (path: string) => deleteFile(id!, path),
    [id, deleteFile],
  );

  return (
    <div className="h-screen flex flex-col">
      <Navbar
        projectName={project?.name || "Loading..."}
        onCompile={compile}
        compiling={compiling}
        editorStatus={editorStatus} // Pass editorStatus
      />

      <div className="flex flex-1 overflow-hidden">
        {/* File Tree Sidebar */}
        <div className="w-64 border-r border-base-300">
          <FileTree
            files={files}
            currentFile={currentFile?.path || null}
            onFileSelect={handleFileSelect}
            onNewFile={handleNewFile}
            onDeleteFile={memoizedOnDeleteFile}
          />
        </div>

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col">
          {/* Toolbar */}
          <div className="tabs tabs-boxed bg-base-200 p-2 flex items-center"> {/* Added flex items-center */}
            <a
              className={`tab ${view === "code" ? "tab-active" : ""}`}
              onClick={() => setView("code")}
            >
              Code
            </a>
            <a
              className={`tab ${view === "pdf" ? "tab-active" : ""}`}
              onClick={() => setView("pdf")}
            >
              PDF
            </a>
            {/* Editor Status Indicator as Text */}
            <span className={`ml-2 text-sm font-medium ${
              editorStatus === 'clean' ? 'text-base-content/70' :
              editorStatus === 'dirty' ? 'text-warning' :
              editorStatus === 'saving' ? 'text-info' :
              editorStatus === 'saved' ? 'text-success' :
              'text-error'
            }`}>
              {editorStatus === 'clean' ? 'No changes' :
              editorStatus === 'dirty' ? 'Unsaved changes' :
              editorStatus === 'saving' ? 'Saving...' :
              editorStatus === 'saved' ? 'Saved' :
              'Error'}
            </span>
          </div>

          {/* Editor/PDF View */}
          <div className="flex-1 overflow-hidden">
            {view === "code" && currentFile ? (
              <Editor
                value={currentFile.content}
                onChange={handleEditorChange}
                onSave={memoizedOnSave}
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
  );
}
