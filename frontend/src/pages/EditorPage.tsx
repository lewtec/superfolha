import { useSaveFileMutation } from "../hooks/useSaveFileMutation";
import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
import { useCommitProjectMutation } from "../hooks/useCommitProjectMutation"; // New import
import { useDebounce } from "../hooks/useDebounce"; // New import
import { useNavigate, useParams } from "react-router-dom";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useGetFilesQuery } from "../hooks/useGetFilesQuery";
import { useCallback, useEffect, useState } from "react";
import Navbar from "../components/Navbar";
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
  const [editorStatus, setEditorStatus] = useState<
    "clean" | "dirty" | "saving" | "saved" | "committed" | "error"
  >("clean");
  const [isCommitted, setIsCommitted] = useState(false);

  const { saveFile, isInFlight: isSavingFile } = useSaveFileMutation();
  const { commitProject, isInFlight: isCommittingProject } =
    useCommitProjectMutation();

  const debouncedCommit = useDebounce(() => {
    if (editorStatus === "saved") {
      // Only commit if saved and no new changes
      commitProject(id!, "Auto-commit: changes saved.", {
        onCompleted: (response, errors) => {
          if (errors) {
            console.error("Auto-commit failed:", errors[0].message);
            // Optionally set editorStatus to error or show a different indicator
            return;
          }
          console.log("Auto-commit successful:", response);
          setIsCommitted(true); // Mark as committed
          setEditorStatus("committed"); // Set status to committed
        },
        onError: (err) => {
          console.error("Auto-commit failed:", err);
          // Optionally set editorStatus to error
        },
      });
    }
  }, 10000); // 10 seconds debounce for commit

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
        // Check for dirty or saving state
        // Prevent immediate unload to allow for a final commit attempt
        event.preventDefault();
        event.returnValue = ""; // Required for Chrome to show the prompt

        // Trigger a commit immediately
        commitProject(id!, "Auto-commit: changes before unload.", {
          onCompleted: () => {
            console.log("Final commit successful before unload.");
            // No need to set editorStatus here as the page is closing
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
  }, [editorStatus, commitProject, id]); // Depend on editorStatus, commitProject, and id

  const memoizedOnSave = useCallback(
    (content: string) => {
      // Accepts content now
      if (currentFile) {
        setEditorStatus("saving"); // Mark as saving
        setIsCommitted(false); // Reset committed status on new changes
        saveFile(id!, currentFile.path, content, {
          // Pass the received content and callbacks
          onCompleted: (response, errors) => {
            if (errors) {
              setEditorStatus("error");
              alert(errors[0].message);
              return;
            }
            setEditorStatus("saved"); // Mark as saved
          },
          onError: (err) => {
            setEditorStatus("error");
            alert("Failed to save file");
            console.error(err);
          },
        });
      }
    },
    [id, currentFile, saveFile, commitProject, editorStatus],
  ); // Added commitProject and editorStatus to dependencies

  return (
    <div className="h-screen flex flex-col">
      <Navbar
        projectName={project?.name || "Loading..."}
        onCompile={compile}
        compiling={compiling}
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
          <div className="tabs tabs-boxed bg-base-200 p-2">
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
