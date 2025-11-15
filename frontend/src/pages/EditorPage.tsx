import { useSaveFileMutation } from "../hooks/useSaveFileMutation";
import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
import { useCommitProjectMutation } from "../hooks/useCommitProjectMutation";
import { useDebounce } from "../hooks/useDebounce";
import { useNavigate, useParams } from "react-router-dom";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useGetFilesQuery } from "../hooks/useGetFilesQuery";
import { useCallback, useEffect, useState } from "react";
import FileTree from "../components/FileTree";
import Editor from "../components/Editor";
import PDFViewer from "../components/PDFViewer";
import LogsPanel from "../components/LogsPanel";

interface File {
  path: string;
  content: string;
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  // All hooks must be called unconditionally before any returns
  const { project, ...projectQueryData } = useGetProjectQuery({ id: id! });
  const { files: fetchedFiles, ...filesQueryData } = useGetFilesQuery({
    projectId: id!,
  });

  const [files, setFiles] = useState<File[]>([]);
  const [currentFile, setCurrentFile] = useState<File | null>(null);
  const [view, setView] = useState<"code" | "pdf">("code");
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [logs, setLogs] = useState("");
  const [compileSuccess, setCompileSuccess] = useState(false);
  const [compiling, setCompiling] = useState(false);
  const [editorStatus, setEditorStatus] = useState<
    "clean" | "dirty" | "saving" | "saved" | "committed" | "error"
  >("clean");
  const [isCommitted, setIsCommitted] = useState(false);

  const { saveFile, isInFlight: isSavingFile } = useSaveFileMutation();
  const { deleteFile } = useDeleteFileMutation();
  const { commitProject, isInFlight: isCommittingProject } =
    useCommitProjectMutation();

  const debouncedCommit = useDebounce(() => {
    if (editorStatus === "saved") {
      commitProject(id!, "Auto-commit: changes saved.", {
        onCompleted: (response, errors) => {
          if (errors) {
            console.error("Auto-commit failed:", errors[0].message);
            return;
          }
          console.log("Auto-commit successful:", response);
          setIsCommitted(true);
          setEditorStatus("committed");
        },
        onError: (err) => {
          console.error("Auto-commit failed:", err);
        },
      });
    }
  }, 10000);

  // Conditional return based on loading state - MUST be after all hooks are called
  if (projectQueryData.loading || filesQueryData.loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="loading loading-spinner loading-lg"></span>
      </div>
    );
  }

  useEffect(() => {
    if (fetchedFiles) {
      setFiles(fetchedFiles as File[]);
      if (fetchedFiles.length > 0) {
        setCurrentFile(fetchedFiles[0] as File);
      } else {
        // If no files are fetched, ensure currentFile is null
        setCurrentFile(null);
      }
    }
  }, [fetchedFiles]);

  // Commit on page close
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (editorStatus === "dirty" || editorStatus === "saving") {
        event.preventDefault();
        event.returnValue = "";

        commitProject(id!, "Auto-commit: changes before unload.", {
          onCompleted: () => {
            console.log("Final commit successful before unload.");
          },
          onError: (err) => {
            console.error("Final commit failed before unload:", err);
          },
        });
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [editorStatus, commitProject, id]);

  // Trigger auto-commit when status changes to 'saved'
  useEffect(() => {
    if (editorStatus === "saved") {
      debouncedCommit();
    }
  }, [editorStatus, debouncedCommit]);

  const memoizedOnSave = useCallback(
    (content: string) => {
      if (currentFile) {
        setEditorStatus("saving");
        setIsCommitted(false);
        saveFile(id!, currentFile.path, content, {
          onCompleted: (response, errors) => {
            if (errors) {
              setEditorStatus("error");
              alert(errors[0].message);
              return;
            }
            setEditorStatus("saved");
          },
          onError: (err) => {
            setEditorStatus("error");
            alert("Failed to save file");
            console.error(err);
          },
        });
      }
    },
    [id, currentFile, saveFile],
  );

  const memoizedOnDeleteFile = useCallback(
    (path: string) => {
      deleteFile(id!, path, {
        onCompleted: (response, errors) => {
          if (errors) {
            alert(errors[0].message);
            return;
          }
          setFiles((prev) => prev.filter((f) => f.path !== path));
          if (currentFile?.path === path) {
            setCurrentFile(files[0] || null);
          }
        },
        onError: (err) => {
          alert("Failed to delete file");
          console.error(err);
        },
      });
    },
    [id, deleteFile, currentFile, files],
  );

  const handleFileSelect = useCallback((path: string) => {
    setFiles((prevFiles) => {
      const file = prevFiles.find((f) => f.path === path);
      if (file) {
        setCurrentFile(file);
      }
      return prevFiles;
    });
  }, []);

  const handleNewFile = useCallback(() => {
    const fileName = prompt("Enter file name:");
    if (fileName) {
      const newFile: File = { path: fileName, content: "" };
      setFiles((prev) => [...prev, newFile]);
      setCurrentFile(newFile);
    }
  }, []);

  const handleEditorChange = useCallback((content: string) => {
    if (currentFile) {
      setCurrentFile({ ...currentFile, content });
      setEditorStatus("dirty");
    }
  }, [currentFile]);

  const compile = useCallback(async () => {
    setCompiling(true);
    setLogs("Compiling...\n");
    try {
      const response = await fetch(`/api/projects/${id}/compile`, {
        method: "POST",
      });
      const data = await response.json();
      setLogs(data.logs || "Compilation completed");
      setCompileSuccess(data.success || false);
      if (data.pdfData) {
        setPdfData(data.pdfData);
      }
    } catch (error) {
      setLogs(`Error: ${error}`);
      setCompileSuccess(false);
    } finally {
      setCompiling(false);
    }
  }, [id]);

  const getStatusBadge = () => {
    switch (editorStatus) {
      case "saving":
        return <span className="badge badge-warning gap-2">Saving...</span>;
      case "saved":
        return <span className="badge badge-success gap-2">Saved</span>;
      case "committed":
      case "clean":
        return <span className="badge badge-info gap-2">Committed</span>;
      case "dirty":
        return <span className="badge badge-error gap-2">Unsaved</span>;
      case "error":
        return <span className="badge badge-error gap-2">Error</span>;
      default:
        return <span className="badge badge-info gap-2">Committed</span>;
    }
  };

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
            onDeleteFile={memoizedOnDeleteFile}
          />
        </div>

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col">
          {/* Toolbar */}
          <div className="flex items-center justify-between bg-base-200 p-2 gap-4">
            <div className="tabs tabs-boxed">
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
            </div>

            <div className="flex items-center gap-3">
              {getStatusBadge()}
              <button
                className={`btn btn-sm ${compiling ? "btn-disabled" : "btn-primary"}`}
                onClick={compile}
                disabled={compiling}
              >
                {compiling ? (
                  <>
                    <span className="loading loading-spinner loading-xs"></span>
                    Compiling...
                  </>
                ) : (
                  "Compile"
                )}
              </button>
            </div>
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
