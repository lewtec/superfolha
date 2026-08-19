import { useDeleteFileMutation } from "../hooks/useDeleteFileMutation";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { translateError, translateGraphQLErrors } from "../i18n/translateError";
import { useGetProjectQuery } from "../hooks/useGetProjectQuery";
import { useAuthStatus } from "../hooks/useAuthStatus";
import { useProjectCollab } from "../hooks/useProjectCollab";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Menu,
  X,
  Code,
  FileText,
  Terminal,
  MessageSquare,
} from "feather-icons-react";
import FileTree from "../components/FileTree";
import CollabEditor from "../components/CollabEditor";
import PDFViewer from "../components/PDFViewer";
import BinaryFileViewer from "../components/BinaryFileViewer";
import Layout from "../components/Layout";
import { isBinaryContent } from "../utils/fileUtils";
import type { ChatMessage, SyncStatus } from "../collab/ProjectCollab";

interface FileRow {
  path: string;
  content: string | null;
  isDirty: boolean;
  isBinary: boolean;
  size: number;
  isTooBig: boolean;
}

type EditorTab = "code" | "pdf" | "logs";

type ChatToast = ChatMessage & { id: number };

const CHAT_TOAST_MS = 5000;
const CHAT_TOAST_MAX = 3;
const CHAT_OPEN_KEY = "superfolha-chat-open";

function ChatToastStack({
  toasts,
  onOpen,
}: {
  toasts: ChatToast[];
  onOpen: () => void;
}) {
  if (toasts.length === 0) return null;
  return (
    <div
      className="toast toast-end toast-bottom z-[60]"
      aria-live="polite"
      aria-relevant="additions"
    >
      {toasts.map((toast) => (
        <button
          key={toast.id}
          type="button"
          role="alert"
          className="alert max-w-xs text-left cursor-pointer"
          onClick={onOpen}
        >
          <span className="flex min-w-0 flex-col gap-0.5">
            <span className="font-semibold text-primary truncate">
              {toast.from}
            </span>
            <span className="line-clamp-2 break-words">{toast.text}</span>
          </span>
        </button>
      ))}
    </div>
  );
}

function statusBadge(status: SyncStatus, t: (k: string) => string) {
  switch (status) {
    case "connecting":
    case "syncing":
      return (
        <span className="badge badge-soft badge-warning whitespace-nowrap">
          {t("editor:status_syncing")}
        </span>
      );
    case "dirty":
      return (
        <span className="badge badge-soft badge-warning whitespace-nowrap">
          {t("editor:status_saving")}
        </span>
      );
    case "synced":
      return (
        <span className="badge badge-soft badge-success whitespace-nowrap">
          {t("editor:status_synced")}
        </span>
      );
    case "committing":
      return (
        <span className="badge badge-soft badge-warning whitespace-nowrap">
          {t("editor:status_committing")}
        </span>
      );
    case "committed":
      return (
        <span className="badge badge-soft badge-info whitespace-nowrap">
          {t("editor:status_committed")}
        </span>
      );
    case "flush_error":
    case "commit_error":
    case "error":
      return (
        <span className="badge badge-soft badge-error whitespace-nowrap">
          {t("editor:status_error")}
        </span>
      );
    case "offline":
      return (
        <span className="badge badge-soft badge-error whitespace-nowrap">
          {t("editor:status_offline")}
        </span>
      );
    default:
      return null;
  }
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation(["editor", "common", "errors"]);
  const { email } = useAuthStatus();
  const {
    collab,
    status,
    files: collabFiles,
    chat,
    peers,
  } = useProjectCollab(id, email);

  const { project } = useGetProjectQuery({ id: id! });
  const fetchedFiles = project?.files;
  const { deleteFile } = useDeleteFileMutation();

  const [currentPath, setCurrentPath] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<EditorTab>("code");
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [logs, setLogs] = useState("");
  const [compiling, setCompiling] = useState(false);
  const [chatInput, setChatInput] = useState("");
  const [chatToasts, setChatToasts] = useState<ChatToast[]>([]);
  const [chatOpen, setChatOpen] = useState(() => {
    try {
      return localStorage.getItem(CHAT_OPEN_KEY) === "1";
    } catch {
      return false;
    }
  });
  const chatOpenRef = useRef(chatOpen);
  const chatLogRef = useRef<HTMLDivElement>(null);
  const chatInputRef = useRef<HTMLInputElement>(null);
  const toastIdRef = useRef(0);
  const toastTimersRef = useRef<number[]>([]);
  const pendingOwnChatRef = useRef<string[]>([]);
  const [sidebarOpen, setSidebarOpen] = useState(() =>
    typeof window !== "undefined"
      ? window.matchMedia("(min-width: 768px)").matches
      : true,
  );
  chatOpenRef.current = chatOpen;

  // Merge GraphQL bootstrap list with live tree.snapshot
  const files: FileRow[] = useMemo(() => {
    if (collabFiles.length > 0) {
      return collabFiles.map((f) => {
        const path = f.path;
        const lower = path.toLowerCase();
        const isBinary =
          isBinaryContent("", path) ||
          // extension-only when no content sample
          /\.(png|jpe?g|gif|pdf|zip|woff2?|mp[34]|ttf|otf)$/i.test(lower);
        return {
          path,
          content: null,
          isDirty: false,
          isBinary,
          size: f.size,
          isTooBig: f.size > 5 * 1024 * 1024,
        };
      });
    }
    if (fetchedFiles) {
      return fetchedFiles.map((file) => ({
        path: file.path,
        isBinary: file.isBinary,
        size: file.size,
        isTooBig: file.isTooBig,
        content: null,
        isDirty: false,
      }));
    }
    return [];
  }, [collabFiles, fetchedFiles]);

  const currentFile = currentPath
    ? files.find((f) => f.path === currentPath) || null
    : null;

  useEffect(() => {
    if (!currentPath && files.length > 0) {
      setCurrentPath(files[0]!.path);
    }
  }, [files, currentPath]);

  const handleFileSelect = useCallback((path: string) => {
    setCurrentPath(path);
    setActiveTab("code");
    if (window.matchMedia("(max-width: 767px)").matches) {
      setSidebarOpen(false);
    }
  }, []);

  const memoizedOnDeleteFile = useCallback(
    (path: string) => {
      if (collab) {
        collab.deleteFile(path);
        if (currentPath === path) {
          setCurrentPath(null);
        }
        return;
      }
      deleteFile(id!, path, {
        onCompleted: (_response, errors) => {
          if (errors) {
            alert(translateGraphQLErrors(t, errors));
            return;
          }
          if (currentPath === path) setCurrentPath(null);
        },
        onError: (err) => {
          alert(t("editor:delete_file_failed"));
          console.error(err);
        },
      });
    },
    [collab, id, deleteFile, currentPath, t],
  );

  const handleNewFile = useCallback(() => {
    const fileName = prompt(t("editor:enter_file_name"));
    if (!fileName) return;
    if (collab) {
      collab.createFile(fileName, "");
      setCurrentPath(fileName);
      setActiveTab("code");
      return;
    }
    setCurrentPath(fileName);
  }, [collab, t]);

  const handleLoadFile = useCallback(
    (fileName: string, content: string | null) => {
      // Uploads go through REST; refresh comes from tree.event / re-list.
      // For text uploads with content, push into collab if live.
      if (content !== null && collab && !isBinaryContent(content, fileName)) {
        collab.createFile(fileName, content);
      }
      setCurrentPath(fileName);
      setActiveTab("code");
    },
    [collab],
  );

  const compile = useCallback(async () => {
    const path = currentPath;
    if (!path) {
      setLogs(t("editor:no_file_compile"));
      setActiveTab("logs");
      return;
    }

    setCompiling(true);
    setLogs("Compiling...\n");

    try {
      const response = await fetch(
        `/api/compile?project=${id}&file=${encodeURIComponent(path)}`,
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
  }, [id, currentPath, t]);

  const sendChat = useCallback(() => {
    const text = chatInput.trim();
    if (!text || !collab) return;
    pendingOwnChatRef.current.push(text);
    collab.sendChat(text);
    setChatInput("");
  }, [chatInput, collab]);

  const persistChatOpen = useCallback((open: boolean) => {
    setChatOpen(open);
    try {
      localStorage.setItem(CHAT_OPEN_KEY, open ? "1" : "0");
    } catch {
      /* ignore quota / private mode */
    }
    if (open) setChatToasts([]);
  }, []);

  const selectView = useCallback((tab: EditorTab) => {
    setActiveTab(tab);
    if (window.matchMedia("(max-width: 767px)").matches) {
      setSidebarOpen(false);
    }
  }, []);

  useEffect(() => {
    if (!collab) return;
    const unsub = collab.subscribeChat((message) => {
      const pending = pendingOwnChatRef.current;
      const ownIdx =
        email && message.from === email ? pending.indexOf(message.text) : -1;
      if (ownIdx !== -1) {
        pending.splice(ownIdx, 1);
        return;
      }
      if (chatOpenRef.current && !document.hidden) return;
      const id = ++toastIdRef.current;
      setChatToasts((prev) => [
        ...prev.slice(1 - CHAT_TOAST_MAX),
        { ...message, id },
      ]);
      const timer = window.setTimeout(() => {
        setChatToasts((prev) => prev.filter((toast) => toast.id !== id));
      }, CHAT_TOAST_MS);
      toastTimersRef.current.push(timer);
    });
    return () => {
      unsub();
      for (const timer of toastTimersRef.current) window.clearTimeout(timer);
      toastTimersRef.current = [];
    };
  }, [collab, email]);

  useEffect(() => {
    const log = chatLogRef.current;
    if (log) log.scrollTop = log.scrollHeight;
  }, [chat, chatOpen]);

  useEffect(() => {
    if (chatOpen) chatInputRef.current?.focus();
  }, [chatOpen]);

  const viewRows: { id: EditorTab; label: string; icon: typeof Code }[] = [
    { id: "code", label: t("editor:code"), icon: Code },
    { id: "pdf", label: t("editor:pdf"), icon: FileText },
    { id: "logs", label: t("editor:logs"), icon: Terminal },
  ];

  const treeFiles = files.map((f) => ({
    path: f.path,
    content: f.content ?? "",
    isDirty: false,
  }));

  const sidebar = (
    <aside
      className={`
        editor-sidebar bg-base-200 border-r border-base-300 flex flex-col
        fixed md:static inset-y-0 left-0 z-40 w-[min(18rem,85vw)]
        transition-transform duration-150 ease-out
        ${sidebarOpen ? "" : "-translate-x-full md:translate-x-0"}
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

      {peers.length > 0 ? (
        <div className="px-3 py-2 border-b border-base-300 flex flex-wrap gap-1">
          {peers.map((p) => (
            <span
              key={p.clientId}
              className="badge badge-sm gap-1"
              style={{ borderColor: p.color, color: p.color }}
              title={p.name}
            >
              <span
                className="w-2 h-2 rounded-full"
                style={{ background: p.color }}
              />
              {p.name}
            </span>
          ))}
        </div>
      ) : null}

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
          files={treeFiles}
          currentFile={currentPath}
          onFileSelect={handleFileSelect}
          onNewFile={handleNewFile}
          onDeleteFile={memoizedOnDeleteFile}
          onLoadFile={handleLoadFile}
          projectId={id!}
        />
      </div>
    </aside>
  );

  const ytext =
    collab && currentPath && currentFile && !currentFile.isBinary
      ? collab.getYText(currentPath)
      : null;

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
          {statusBadge(status, t)}
          <button
            type="button"
            className="btn btn-ghost btn-sm hidden sm:inline-flex"
            onClick={() => collab?.commitNow()}
            title={t("editor:commit_now")}
          >
            {t("editor:commit_now")}
          </button>
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
          <button
            type="button"
            className={`btn btn-ghost btn-square min-h-[var(--touch-min)] min-w-[var(--touch-min)] ${chatOpen ? "btn-active" : ""}`}
            onClick={() => persistChatOpen(!chatOpen)}
            aria-controls="chat-panel"
            aria-expanded={chatOpen}
            aria-label={
              chatOpen ? t("editor:chat_close") : t("editor:chat_open")
            }
            title={t("editor:chat")}
          >
            <MessageSquare size={18} />
          </button>
        </>
      }
    >
      <div className="editor-workspace relative flex flex-1 min-h-0 overflow-hidden">
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

        <ChatToastStack
          toasts={chatToasts}
          onOpen={() => persistChatOpen(true)}
        />

        <div className="flex-1 flex flex-col min-w-0 min-h-0 bg-base-100">
          {currentFile ? (
            <div className="px-3 py-1.5 text-sm border-b border-base-300 truncate text-base-content/80 shrink-0">
              {currentFile.path}
            </div>
          ) : null}

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
              ) : currentFile && ytext && collab && collab.initialSynced ? (
                <CollabEditor
                  // Remount after initial sync so CM seeds from real Y.Text
                  key={`${currentFile.path}:${collab.sessionId}:synced`}
                  path={currentFile.path}
                  ytext={ytext}
                  awareness={collab.awareness}
                />
              ) : currentFile && collab && !collab.initialSynced ? (
                <div className="flex items-center justify-center h-full text-base-content/70 page-pad">
                  {t("editor:status_syncing")}
                </div>
              ) : currentFile && !collab ? (
                <div className="flex items-center justify-center h-full text-base-content/70 page-pad">
                  {t("editor:status_connecting")}
                </div>
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
              className={`absolute inset-0 min-h-0 flex flex-col p-3 ${
                activeTab === "logs"
                  ? "z-10"
                  : "z-0 invisible pointer-events-none"
              }`}
              aria-hidden={activeTab !== "logs"}
            >
              <pre className="flex-1 overflow-auto text-xs font-mono whitespace-pre-wrap bg-base-200 rounded-box p-3">
                {logs || t("editor:logs_empty")}
              </pre>
            </div>
          </div>
        </div>
      </div>

      <aside
        id="chat-panel"
        className={`fixed inset-x-0 z-40 flex justify-end pointer-events-none ${
          chatOpen ? "visible" : "invisible"
        }`}
        style={{
          top: "var(--shell-height)",
          height: "calc(100dvh - var(--shell-height))",
        }}
        aria-label={t("editor:chat")}
        aria-hidden={!chatOpen}
      >
        <div
          className={`pointer-events-auto flex flex-col h-full w-full md:w-96 max-w-full bg-base-100 border-l border-base-300 shadow-lg motion-safe:transition-transform motion-safe:duration-200 motion-safe:ease-out ${
            chatOpen ? "" : "translate-x-full"
          }`}
        >
          <div className="flex items-center justify-between gap-2 min-h-12 px-3 border-b border-base-300 shrink-0">
            <div className="min-w-0">
              <div className="text-sm font-semibold leading-tight">
                {t("editor:chat")}
              </div>
              <div className="text-xs text-base-content/50 leading-tight">
                {t("editor:chat_session_hint")}
              </div>
            </div>
            <button
              type="button"
              className="btn btn-ghost btn-square btn-sm min-h-[var(--touch-min)] min-w-[var(--touch-min)]"
              onClick={() => persistChatOpen(false)}
              aria-label={t("editor:chat_close")}
              title={t("editor:chat_close")}
            >
              <X size={18} />
            </button>
          </div>
          <div
            ref={chatLogRef}
            className="flex-1 min-h-0 overflow-y-auto px-4 py-3 text-sm"
            aria-live="polite"
          >
            {chat.length === 0 ? (
              <p className="text-base-content/60">{t("editor:chat_empty")}</p>
            ) : (
              chat.map((m, i) => (
                <div key={`${m.at}-${i}`} className="py-1">
                  <span className="font-semibold text-primary mr-1.5">
                    {m.from}
                  </span>
                  {m.text}
                </div>
              ))
            )}
          </div>
          <form
            className="flex flex-col gap-2 p-3 border-t border-base-300 shrink-0"
            onSubmit={(e) => {
              e.preventDefault();
              sendChat();
            }}
          >
            <input
              ref={chatInputRef}
              type="text"
              className="input input-bordered input-sm w-full"
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              placeholder={t("editor:chat_placeholder")}
              aria-label={t("editor:chat_message_label")}
              autoComplete="off"
            />
            <button className="btn btn-primary btn-sm" type="submit">
              {t("editor:chat_send")}
            </button>
          </form>
        </div>
      </aside>
    </Layout>
  );
}
