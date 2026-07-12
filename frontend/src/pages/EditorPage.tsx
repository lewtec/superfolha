import { useGetFileContent } from "../hooks/useGetFileContentQuery";
import { useSaveFileMutation } from "../hooks/useSaveFileMutation";
import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
import { useCommitProjectMutation } from "../hooks/useCommitProjectMutation";
import { useDebounce } from "../hooks/useDebounce";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { translateError, translateGraphQLErrors } from "../i18n/translateError";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useCallback, useEffect, useState } from "react";
import { Menu, X, Code, FileText, Terminal } from "feather-icons-react";
import FileTree from "../components/FileTree";
import Editor from "../components/Editor";
import PDFViewer from "../components/PDFViewer";
import BinaryFileViewer from "../components/BinaryFileViewer";
import Layout from "../components/Layout";
import { isBinaryContent } from "../utils/fileUtils";

interface File {
  path: string;
  content: string | null;
  isDirty: boolean;
  isBinary: boolean;
  size: number;
  isTooBig: boolean;
}

type EditorTab = "code" | "pdf" | "logs";

export default function EditorPage() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation(["editor", "common", "errors"]);

  const { project } = useGetProjectQuery({ id: id! });
  const fetchedFiles = project?.files;
  const { getFileContent } = useGetFileContent();

  const [files, setFiles] = useState<File[]>([]);
  const [currentFile, setCurrentFile] = useState<File | null>(null);
  const [activeTab, setActiveTab] = useState<EditorTab>("code");
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [logs, setLogs] = useState("");
  const [compiling, setCompiling] = useState(false);
  const [editorStatus, setEditorStatus] = useState<
    "clean" | "dirty" | "saving" | "saved" | "committed" | "error"
  >("clean");
  const [sidebarOpen, setSidebarOpen] = useState(() =>
    typeof window !== "undefined"
      ? window.matchMedia("(min-width: 768px)").matches
      : true,
  );

  const { saveFile } = useSaveFileMutation();
  const { deleteFile } = useDeleteFileMutation();
  const { commitProject } = useCommitProjectMutation();

  const debouncedCommit = useDebounce(() => {
    if (editorStatus === "saved") {
      commitProject(id!, "Auto-commit: changes saved.", {
        onCompleted: (response, errors) => {
          if (errors) {
            console.error(
              "Auto-commit failed:",
              translateGraphQLErrors(t, errors),
            );
            return;
          }
          setEditorStatus("committed");
        },
        onError: (err) => {
          console.error("Auto-commit failed:", translateError(t, err));
        },
      });
    }
  }, 10000);

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
        setActiveTab("code");
        // Collapse drawer on small screens after pick
        if (window.matchMedia("(max-width: 767px)").matches) {
          setSidebarOpen(false);
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

      const initialFiles: File[] = (fetchedFiles as FetchedFile[]).map(
        (file) => ({
          ...file,
          content: null,
          isDirty: false,
        }),
      );
      setFiles(initialFiles);
      if (initialFiles.length > 0) {
        handleFileSelect(initialFiles[0].path);
      } else {
        setCurrentFile(null);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only re-seed when server file list changes
  }, [fetchedFiles]);

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
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
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [files, commitProject, id]);

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
          onCompleted: (_response, errors) => {
            if (errors) {
              setEditorStatus("error");
              alert(translateGraphQLErrors(t, errors));
              return;
            }
            setEditorStatus("saved");
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
            alert(t("editor:save_failed"));
            console.error(err);
          },
        });
      }
    },
    [id, currentFile, saveFile, t],
  );

  const memoizedOnDeleteFile = useCallback(
    (path: string) => {
      deleteFile(id!, path, {
        onCompleted: (_response, errors) => {
          if (errors) {
            alert(translateGraphQLErrors(t, errors));
            return;
          }
          setFiles((prev) => prev.filter((f) => f.path !== path));
          if (currentFile?.path === path) {
            setCurrentFile(files.filter((f) => f.path !== path)[0] || null);
          }
        },
        onError: (err) => {
          alert(t("editor:delete_file_failed"));
          console.error(err);
        },
      });
    },
    [id, deleteFile, currentFile, files, t],
  );

  const handleNewFile = useCallback(() => {
    const fileName = prompt(t("editor:enter_file_name"));
    if (fileName) {
      const newFile: File = {
        path: fileName,
        content: "",
        isDirty: true,
        isBinary: false,
        size: 0,
        isTooBig: false,
      };
      setFiles((prev) => [...prev, newFile]);
      setCurrentFile(newFile);
      setActiveTab("code");
    }
  }, [t]);

  const handleLoadFile = useCallback(
    (fileName: string, content: string | null) => {
      let isBinary = false;
      let size = 0;
      let isTooBig = false;

      if (content !== null) {
        isBinary = isBinaryContent(content, fileName);
        size = content.length;
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
        return [
          ...prev,
          {
            path: fileName,
            content: content !== null ? content : null,
            isDirty: true,
            isBinary,
            size,
            isTooBig,
          },
        ];
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
        return (
          files.find((file) => file.path === fileName) || {
            path: fileName,
            content,
            isDirty: true,
            isBinary,
            size,
            isTooBig,
          }
        );
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
      setLogs(t("editor:no_file_compile"));
      setActiveTab("logs");
      return;
    }

    setCompiling(true);
    setLogs("Compiling...\n");

    try {
      const response = await fetch(
        `/api/compile?project=${id}&file=${encodeURIComponent(currentFile.path)}`,
        {
          method: "GET",
          credentials: "include",
        },
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const msg = translateError(t, errorData);
        throw Object.assign(new Error(msg), { code: errorData.code });
      }

      const data = await response.json();
      setLogs(data.logs || "Compilation completed");
      if (data.pdf) {
        setPdfData(data.pdf);
        setActiveTab("pdf");
      } else if (!data.success) {
        setLogs(
          data.logs || "Compilation failed with no specific error message.",
        );
        setActiveTab("logs");
      }
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : String(error);
      setLogs(`Error: ${errorMessage}`);
      setActiveTab("logs");
    } finally {
      setCompiling(false);
    }
  }, [id, currentFile, t]);

  const getStatusBadge = () => {
    const anyDirty = files.some((file) => file.isDirty);

    if (anyDirty) {
      return (
        <span className="badge badge-soft badge-error whitespace-nowrap">
          {t("editor:status_unsaved")}
        </span>
      );
    }

    switch (editorStatus) {
      case "saving":
        return (
          <span className="badge badge-soft badge-warning whitespace-nowrap">
            {t("editor:status_saving")}
          </span>
        );
      case "saved":
        return (
          <span className="badge badge-soft badge-success whitespace-nowrap">
            {t("editor:status_saved")}
          </span>
        );
      case "error":
        return (
          <span className="badge badge-soft badge-error whitespace-nowrap">
            {t("editor:status_error")}
          </span>
        );
      case "committed":
      case "clean":
      default:
        return (
          <span className="badge badge-soft badge-info whitespace-nowrap">
            {t("editor:status_committed")}
          </span>
        );
    }
  };

  const viewRows: { id: EditorTab; label: string; icon: typeof Code }[] = [
    { id: "code", label: t("editor:code"), icon: Code },
    { id: "pdf", label: t("editor:pdf"), icon: FileText },
    { id: "logs", label: t("editor:logs"), icon: Terminal },
  ];

  const selectView = (tab: EditorTab) => {
    setActiveTab(tab);
    if (window.matchMedia("(max-width: 767px)").matches) {
      setSidebarOpen(false);
    }
  };

  const sidebar = (
    <aside
      className={`
        editor-sidebar bg-base-200 border-r border-base-300 flex flex-col
        fixed md:static inset-y-0 left-0 z-40 w-[min(18rem,85vw)]
        transition-transform duration-150 ease-out
        ${sidebarOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0"}
        ${sidebarOpen ? "md:w-64" : "md:w-0 md:border-0 md:overflow-hidden"}
      `}
      style={{
        top: "var(--shell-height)",
        height: "calc(100dvh - var(--shell-height))",
      }}
      aria-hidden={!sidebarOpen}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-base-300 md:hidden shrink-0">
        <span className="font-semibold text-sm">{t("editor:menu")}</span>
        <button
          type="button"
          className="btn btn-ghost btn-square btn-sm min-h-[var(--touch-min)] min-w-[var(--touch-min)]"
          onClick={() => setSidebarOpen(false)}
          aria-label={t("editor:close_sidebar")}
        >
          <X size={20} />
        </button>
      </div>

      <nav
        className="p-2 border-b border-base-300 shrink-0"
        aria-label={t("editor:view")}
      >
        <p className="px-2 pb-1 text-xs font-medium uppercase tracking-wide text-base-content/60">
          {t("editor:view")}
        </p>
        <ul className="menu menu-md bg-transparent rounded-box w-full p-0 gap-0.5">
          {viewRows.map(({ id: tabId, label, icon: Icon }) => (
            <li key={tabId}>
              <button
                type="button"
                className={`min-h-[var(--touch-min)] ${activeTab === tabId ? "active" : ""}`}
                onClick={() => selectView(tabId)}
              >
                <Icon size={18} />
                {label}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
        <FileTree
          files={files as { path: string; content: string; isDirty: boolean }[]}
          currentFile={currentFile?.path || null}
          onFileSelect={handleFileSelect}
          onNewFile={handleNewFile}
          onDeleteFile={memoizedOnDeleteFile}
          onLoadFile={handleLoadFile}
          projectId={id!}
        />
      </div>
    </aside>
  );

  return (
    <Layout
      navStart={
        <button
          type="button"
          className="btn btn-ghost btn-square min-h-[var(--touch-min)] min-w-[var(--touch-min)]"
          onClick={() => setSidebarOpen((o) => !o)}
          aria-label={
            sidebarOpen ? t("editor:close_sidebar") : t("editor:open_sidebar")
          }
          aria-expanded={sidebarOpen}
        >
          {sidebarOpen ? <X size={22} /> : <Menu size={22} />}
        </button>
      }
      navEnd={
        <>
          {getStatusBadge()}
          <button
            type="button"
            className={`btn btn-primary btn-sm sm:btn-md min-h-[var(--touch-min)] ${compiling ? "btn-disabled" : ""}`}
            onClick={compile}
            disabled={compiling}
          >
            {compiling ? (
              <>
                <span className="loading loading-spinner loading-xs" />
                <span className="hidden sm:inline">
                  {t("editor:compiling")}
                </span>
              </>
            ) : (
              t("editor:compile")
            )}
          </button>
        </>
      }
    >
      <div className="editor-workspace relative flex flex-1 min-h-0 overflow-hidden">
        {/* Backdrop (mobile) */}
        {sidebarOpen ? (
          <button
            type="button"
            className="fixed inset-0 z-30 bg-neutral/40 md:hidden"
            style={{ top: "var(--shell-height)" }}
            aria-label={t("editor:close_sidebar")}
            onClick={() => setSidebarOpen(false)}
          />
        ) : null}

        {sidebar}

        <div className="flex-1 flex flex-col min-w-0 min-h-0 bg-base-100">
          {currentFile ? (
            <div className="px-3 py-1.5 text-sm border-b border-base-300 truncate text-base-content/80 shrink-0">
              {currentFile.path}
              {currentFile.isDirty ? (
                <span className="text-warning ml-1">•</span>
              ) : null}
            </div>
          ) : null}

          {/*
            Keep all panes mounted so tab switches do not remount CodeMirror
            or reload the PDF iframe. Inactive panes are only hidden.
          */}
          <div className="relative flex-1 min-h-0">
            <div
              className={`absolute inset-0 min-h-0 flex flex-col ${
                activeTab === "code"
                  ? "z-10"
                  : "z-0 invisible pointer-events-none"
              }`}
              aria-hidden={activeTab !== "code"}
            >
              {currentFile && currentFile.isBinary ? (
                <BinaryFileViewer fileName={currentFile.path} projectId={id!} />
              ) : currentFile ? (
                <Editor
                  key={currentFile.path}
                  value={currentFile.content}
                  onChange={handleEditorChange}
                  onSave={memoizedOnSave}
                />
              ) : (
                <div className="flex items-center justify-center h-full text-base-content/70 page-pad">
                  {t("editor:select_file")}
                </div>
              )}
            </div>

            <div
              className={`absolute inset-0 min-h-0 flex flex-col ${
                activeTab === "pdf"
                  ? "z-10"
                  : "z-0 invisible pointer-events-none"
              }`}
              aria-hidden={activeTab !== "pdf"}
            >
              <PDFViewer pdfData={pdfData} />
            </div>

            <div
              className={`absolute inset-0 min-h-0 flex flex-col ${
                activeTab === "logs"
                  ? "z-10"
                  : "z-0 invisible pointer-events-none"
              }`}
              aria-hidden={activeTab !== "logs"}
            >
              <Editor
                key="__logs__"
                value={logs}
                onChange={() => null}
                onSave={() => null}
              />
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
