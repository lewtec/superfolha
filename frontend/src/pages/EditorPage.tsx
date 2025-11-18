import { useGetFileContent } from "../hooks/useGetFileContentQuery";
import { useSaveFileMutation } from "../hooks/useSaveFileMutation";
import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
import { useCommitProjectMutation } from "../hooks/useCommitProjectMutation";
import { useDebounce } from "../hooks/useDebounce";
import { useParams } from "react-router-dom";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useCallback, useEffect, useState } from "react";
import FileTree from "../components/FileTree";
import Editor from "../components/Editor";
import PDFViewer from "../components/PDFViewer";
import BinaryFileViewer from "../components/BinaryFileViewer"; // Import BinaryFileViewer
import { isBinaryContent } from "../utils/fileUtils"; // Import isBinaryContent

interface File {
  path: string;
  content: string | null; // Content can be null now
  isDirty: boolean;
  isBinary: boolean; // Added isBinary property
  size: number; // Added size property
  isTooBig: boolean; // Added isTooBig property
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>();

  // All hooks must be called unconditionally before any returns
  const { project, ...projectQueryData } = useGetProjectQuery({ id: id! });
  const fetchedFiles = project?.files;
  const { getFileContent } = useGetFileContent();

  const [files, setFiles] = useState<File[]>([]);
  const [currentFile, setCurrentFile] = useState<File | null>(null);
  const [activeTab, setActiveTab] = useState<"code" | "pdf" | "logs">("code");
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [logs, setLogs] = useState("");
  const [compiling, setCompiling] = useState(false);
  const [editorStatus, setEditorStatus] = useState<
    "clean" | "dirty" | "saving" | "saved" | "committed" | "error"
  >("clean");

  const { saveFile } = useSaveFileMutation();
  const { deleteFile } = useDeleteFileMutation();
  const { commitProject } = useCommitProjectMutation();

  const debouncedCommit = useDebounce(() => {
    if (editorStatus === "saved") {
      commitProject(id!, "Auto-commit: changes saved.", {
        onCompleted: (response, errors) => {
          if (errors) {
            console.error("Auto-commit failed:", errors[0].message);
            return;
          }
          console.log("Auto-commit successful:", response);
          setEditorStatus("committed");
        },
        onError: (err) => {
          console.error("Auto-commit failed:", err);
        },
      });
    }
  }, 10000);

  // Conditional return based on loading state - MUST be after all hooks are called
  if (projectQueryData.loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="loading loading-spinner loading-lg"></span>
      </div>
    );
  }

  const handleFileSelect = useCallback(
    async (path: string) => {
      const file = files.find((f) => f.path === path);
      if (file) {
        if (file.content == null && !file.isBinary) {
          const response = await getFileContent({ id: id!, path });
          const content = response?.project?.file?.content;
          const updatedFile = { ...file, content };
          setFiles(files.map((f) => (f.path === path ? updatedFile : f)));
          setCurrentFile(updatedFile);
        } else {
          setCurrentFile(file);
        }
      }
    },
    [files, getFileContent, id],
  );

  useEffect(() => {
    if (fetchedFiles) {
      type FetchedFile = {
        path: string;
        isBinary: boolean;
        size: number;
        isTooBig: boolean;
      };

      const initialFiles: File[] = (fetchedFiles as FetchedFile[]).map((file) => {
        return {
          ...file,
          content: null,
          isDirty: false,
        };
      });
      setFiles(initialFiles);
      if (initialFiles.length > 0) {
        handleFileSelect(initialFiles[0].path);
      } else {
        setCurrentFile(null);
      }
    }
  }, [fetchedFiles]);

  // Commit on page close
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      // Check if any file is dirty
      const anyDirty = files.some((file) => file.isDirty);
      if (anyDirty) {
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
  }, [files, commitProject, id]); // Added files to dependency array

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
        saveFile(id!, currentFile.path, content, {
          onCompleted: (response, errors) => {
            if (errors) {
              setEditorStatus("error");
              alert(errors[0].message);
              return;
            }
            setEditorStatus("saved");
            // Update the isDirty status of the saved file
            setFiles((prevFiles) =>
              prevFiles.map((file) =>
                file.path === currentFile.path
                  ? { ...file, content, isDirty: false }
                  : file,
              ),
            );
            setCurrentFile((prev) =>
              prev ? { ...prev, content, isDirty: false } : null,
            );
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
            // Try to select the first file if the current one was deleted
            setCurrentFile(files.filter((f) => f.path !== path)[0] || null);
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

  const handleNewFile = useCallback(() => {
    const fileName = prompt("Enter file name:");
    if (fileName) {
      const newFile: File = { path: fileName, content: "", isDirty: true }; // New files are dirty
      setFiles((prev) => [...prev, newFile]);
      setCurrentFile(newFile);
    }
  }, []);

  const handleLoadFile = useCallback(
    (fileName: string, content: string | null) => {
      let isBinary = false;
      let size = 0;
      let isTooBig = false;

      if (content !== null) {
        isBinary = isBinaryContent(content, fileName);
        size = content.length;
        // For client-side loaded files, we assume they are not "too big" if content is provided
        // The backend determines "isTooBig" for fetched files.
        isTooBig = false;
      } else {
        // If content is null, we can't determine binary status or size client-side.
        // We might assume it's binary or too big if it came from a context that implies it.
        // For now, we'll default to not binary and size 0 if content is null.
        // This might need refinement based on how `handleLoadFile` is called for null content.
        isBinary = false;
        size = 0;
        isTooBig = false;
      }

      setFiles((prev) => {
        const existingFileIndex = prev.findIndex(
          (file) => file.path === fileName,
        );
        if (existingFileIndex > -1) {
          const updatedFiles = [...prev];
          updatedFiles[existingFileIndex] = {
            ...updatedFiles[existingFileIndex],
            content:
              content !== null
                ? content
                : updatedFiles[existingFileIndex].content,
            isDirty: true,
            isBinary,
            size,
            isTooBig,
          };
          return updatedFiles;
        }
        const newFile: File = {
          path: fileName,
          content: content !== null ? content : null,
          isDirty: true,
          isBinary,
          size,
          isTooBig,
        };
        return [...prev, newFile];
      });
      setCurrentFile((prev) => {
        if (prev && prev.path === fileName) {
          return {
            ...prev,
            content: content !== null ? content : prev.content,
            isDirty: true,
            isBinary,
            size,
            isTooBig,
          };
        }
        const loadedFile = files.find((file) => file.path === fileName);
        return loadedFile || null;
      });

      if (content !== null && !isBinary) {
        memoizedOnSave(content);
      }
    },
    [files, memoizedOnSave],
  );

  const handleEditorChange = useCallback(
    (content: string) => {
      setFiles((prevFiles) =>
        prevFiles.map((file) =>
          file.path === currentFile?.path
            ? { ...file, content, isDirty: true }
            : file,
        ),
      );
      setCurrentFile((prev) =>
        prev ? { ...prev, content, isDirty: true } : null,
      );
      setEditorStatus("dirty");
    },
    [currentFile],
  );

  const compile = useCallback(async () => {
    if (!currentFile || !currentFile.path) {
      setLogs("Error: No file selected for compilation.");
      return;
    }

    setCompiling(true);
    setLogs("Compiling...\n");

    // No explicit authToken check here, as it's expected in a cookie
    // No Authorization header needed, as the token is in a cookie

    try {
      const response = await fetch(
        `/api/compile?project=${id}&file=${currentFile.path}`,
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json", // Keep this if sending a body, but for GET it's often optional
          },
          credentials: "include", // Essential for sending cookies
        },
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Compilation failed");
      }

      const data = await response.json();
      setLogs(data.logs || "Compilation completed");
      if (data.pdf) {
        // Changed from data.pdfData to data.pdf
        setPdfData(data.pdf);
      } else if (!data.success) {
        // If compilation failed and no PDF, ensure error is shown
        setLogs(
          data.logs || "Compilation failed with no specific error message.",
        );
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      setLogs(`Error: ${errorMessage}`);
    } finally {
      setCompiling(false);
    }
  }, [id, currentFile]); // Added currentFile to dependencies

  const getStatusBadge = () => {
    // Check if any file is dirty
    const anyDirty = files.some((file) => file.isDirty);

    if (anyDirty) {
      return <span className="badge badge-error gap-2">Unsaved</span>;
    }

    switch (editorStatus) {
      case "saving":
        return <span className="badge badge-warning gap-2">Saving...</span>;
      case "saved":
        return <span className="badge badge-success gap-2">Saved</span>;
      case "committed":
      case "clean":
        return <span className="badge badge-info gap-2">Committed</span>;
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
            onLoadFile={handleLoadFile}
            projectId={id!} // Pass projectId here
          />
        </div>

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col">
          {/* Toolbar */}
          <div className="flex items-center justify-between bg-base-200 p-2 gap-4">
            <div className="tabs tabs-boxed">
              <a
                className={`tab ${activeTab === "code" ? "tab-active" : ""}`}
                onClick={() => setActiveTab("code")}
              >
                Code
              </a>
              <a
                className={`tab ${activeTab === "pdf" ? "tab-active" : ""}`}
                onClick={() => setActiveTab("pdf")}
              >
                PDF
              </a>
              <a
                className={`tab ${activeTab === "logs" ? "tab-active" : ""}`}
                onClick={() => setActiveTab("logs")}
              >
                Logs
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

          {/* Content Area based on activeTab */}
          <div className="flex-1 overflow-hidden">
            {activeTab === "code" && currentFile && currentFile.isBinary ? (
              <BinaryFileViewer fileName={currentFile.path} projectId={id!} />
            ) : activeTab === "code" && currentFile ? (
              <Editor
                key={currentFile.path}
                value={currentFile.content}
                onChange={handleEditorChange}
                onSave={memoizedOnSave}
              />
            ) : activeTab === "pdf" ? (
              <PDFViewer pdfData={pdfData} />
            ) : activeTab === "logs" ? (
              <Editor value={logs} onChange={() => null} onSave={() => null} />
            ) : (
              <div className="flex items-center justify-center h-full text-base-content/70">
                Select a file or tab to view content.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
