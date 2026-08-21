import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { autocompletion } from "@codemirror/autocomplete";
import { yCollab } from "y-codemirror.next";
import * as Y from "yjs";
import type { Awareness } from "y-protocols/awareness";
import { ProjectCollab, type ChatMessage, type SyncStatus } from "./collab/ProjectCollab";
import { latexCompletions, latexLanguage } from "./latexCompletions";
import { buildFileTree, type FileTreeNode } from "./utils/fileTree";
import { isBinaryContent } from "./utils/fileUtils";

type Tab = "code" | "pdf" | "logs";

function t(msgs: Record<string, string>, id: string, data?: Record<string, string>): string {
  let s = msgs[id] ?? id;
  if (data) {
    for (const [k, v] of Object.entries(data)) {
      s = s.replaceAll(`{{.${k}}}`, v);
    }
  }
  return s;
}

function icon(path: string): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="${path}"/></svg>`;
}

const ICO = {
  menu: "M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5",
  x: "M6 18 18 6M6 6l12 12",
  chat: "M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.76 9.76 0 0 1-2.53-.33L3 21l1.08-3.24A8.25 8.25 0 0 1 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z",
};

function statusClass(status: SyncStatus): string {
  switch (status) {
    case "synced":
    case "committed":
      return "badge-success";
    case "error":
    case "flush_error":
    case "commit_error":
    case "offline":
      return "badge-error";
    default:
      return "badge-warning";
  }
}

function statusKey(status: SyncStatus): string {
  switch (status) {
    case "connecting":
      return "editor.status_connecting";
    case "syncing":
      return "editor.status_syncing";
    case "dirty":
      return "editor.status_saving";
    case "synced":
      return "editor.status_synced";
    case "committing":
      return "editor.status_committing";
    case "committed":
      return "editor.status_committed";
    case "offline":
      return "editor.status_offline";
    default:
      return "editor.status_error";
  }
}

function renderTree(
  nodes: FileTreeNode[],
  current: string | null,
  expanded: Set<string>,
): string {
  return nodes
    .map((node) => {
      if (node.type === "directory") {
        const open = expanded.has(node.path);
        const kids = open && node.children ? renderTree(node.children, current, expanded) : "";
        return `<li>
          <button type="button" class="justify-start" data-dir="${escapeAttr(node.path)}">${escapeHtml(node.name)}</button>
          ${open ? `<ul>${kids}</ul>` : ""}
        </li>`;
      }
      const active = current === node.path ? "active" : "";
      const del =
        node.path === "main.tex"
          ? ""
          : `<button type="button" class="btn btn-ghost btn-xs" data-del="${escapeAttr(node.path)}" aria-label="delete">×</button>`;
      return `<li>
        <div class="flex items-center justify-between ${active}">
          <button type="button" class="flex-1 text-left" data-file="${escapeAttr(node.path)}">${escapeHtml(node.name)}</button>
          ${del}
        </div>
      </li>`;
    })
    .join("");
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (ch) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]!,
  );
}

function escapeAttr(s: string): string {
  return escapeHtml(s);
}

function toPdfBlobUrl(pdfData: string): string {
  const base64 = pdfData.startsWith("data:")
    ? pdfData.replace(/^data:application\/pdf;base64,/, "")
    : pdfData;
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return URL.createObjectURL(new Blob([bytes], { type: "application/pdf" }));
}

function mountEditor(host: HTMLElement, ytext: Y.Text, awareness: Awareness, path: string): EditorView {
  const undoManager = new Y.UndoManager(ytext);
  return new EditorView({
    parent: host,
    state: EditorState.create({
      doc: ytext.toString(),
      extensions: [
        basicSetup,
        latexLanguage,
        autocompletion({ override: [latexCompletions] }),
        EditorView.lineWrapping,
        yCollab(ytext, awareness, { undoManager }),
      ],
    }),
  });
}

function boot() {
  const root = document.getElementById("editor-root");
  if (!root) return;
  const projectId = root.dataset.projectId ?? "";
  const email = root.dataset.email ?? "";
  const compileBase = root.dataset.compile ?? "";
  const uploadURL = root.dataset.upload ?? "";
  const downloadPrefix = root.dataset.downloadPrefix ?? "";
  const wsPath = root.dataset.ws ?? "";
  let msgs: Record<string, string> = {};
  try {
    msgs = JSON.parse(root.dataset.i18n || "{}") as Record<string, string>;
  } catch {
    msgs = {};
  }

  const workspace = document.getElementById("editor-workspace")!;
  const statusEl = document.getElementById("editor-status")!;
  const toggleBtn = document.getElementById("editor-toggle-sidebar") as HTMLButtonElement;
  const compileBtn = document.getElementById("editor-compile") as HTMLButtonElement;
  const commitBtn = document.getElementById("editor-commit") as HTMLButtonElement;
  const chatToggle = document.getElementById("editor-chat-toggle") as HTMLButtonElement;

  toggleBtn.innerHTML = icon(ICO.menu);
  chatToggle.innerHTML = icon(ICO.chat);

  let sidebarOpen = true;
  let chatOpen = localStorage.getItem("superfolha-chat-open") === "1";
  let tab: Tab = "code";
  let currentPath: string | null = null;
  let compiling = false;
  let logs = "";
  let pdfUrl: string | null = null;
  let view: EditorView | null = null;
  const expanded = new Set<string>();

  const collab = new ProjectCollab(projectId, email, wsPath);
  collab.connect();

  workspace.innerHTML = `
    <aside id="sf-sidebar" class="editor-sidebar bg-base-200 border-r border-base-300 flex flex-col w-[min(18rem,85vw)] md:w-64 shrink-0 min-h-0"></aside>
    <div class="flex-1 flex flex-col min-w-0 min-h-0 bg-base-100">
      <div id="sf-path" class="px-3 py-1.5 text-sm border-b border-base-300 truncate text-base-content/80 shrink-0 hidden"></div>
      <div class="relative flex-1 min-h-0">
        <div id="sf-code" class="absolute inset-0 min-h-0"></div>
        <div id="sf-pdf" class="absolute inset-0 min-h-0 hidden"></div>
        <div id="sf-logs" class="absolute inset-0 min-h-0 hidden p-3">
          <pre id="sf-logs-pre" class="flex-1 h-full overflow-auto text-xs font-mono whitespace-pre-wrap bg-base-200 rounded-box p-3"></pre>
        </div>
      </div>
    </div>
    <aside id="sf-chat" class="hidden w-full md:w-96 max-w-full border-l border-base-300 bg-base-100 flex flex-col"></aside>
  `;

  const sidebar = document.getElementById("sf-sidebar")!;
  const codeHost = document.getElementById("sf-code")!;
  const pdfHost = document.getElementById("sf-pdf")!;
  const logsPre = document.getElementById("sf-logs-pre")!;
  const pathEl = document.getElementById("sf-path")!;
  const chatEl = document.getElementById("sf-chat")!;

  function showTab(next: Tab) {
    tab = next;
    document.getElementById("sf-code")!.classList.toggle("hidden", next !== "code");
    document.getElementById("sf-pdf")!.classList.toggle("hidden", next !== "pdf");
    document.getElementById("sf-logs")!.classList.toggle("hidden", next !== "logs");
    paintSidebar();
  }

  function paintStatus() {
    statusEl.className = `badge badge-soft ${statusClass(collab.status)} whitespace-nowrap`;
    statusEl.textContent = t(msgs, statusKey(collab.status));
  }

  function paintSidebar() {
    const files = collab.files.map((f) => ({ path: f.path, isDirty: false }));
    const tree = buildFileTree(files);
    const peers = collab.peers();
    sidebar.innerHTML = `
      <nav class="p-2 border-b border-base-300 shrink-0" aria-label="${escapeAttr(t(msgs, "editor.view"))}">
        <p class="px-2 pb-1 text-xs font-medium uppercase tracking-wide text-base-content/60">${escapeHtml(t(msgs, "editor.view"))}</p>
        <ul class="menu menu-md bg-transparent rounded-box w-full p-0 gap-0.5">
          <li><button type="button" data-tab="code" class="${tab === "code" ? "active" : ""}">${escapeHtml(t(msgs, "editor.code"))}</button></li>
          <li><button type="button" data-tab="pdf" class="${tab === "pdf" ? "active" : ""}">${escapeHtml(t(msgs, "editor.pdf"))}</button></li>
          <li><button type="button" data-tab="logs" class="${tab === "logs" ? "active" : ""}">${escapeHtml(t(msgs, "editor.logs"))}</button></li>
        </ul>
      </nav>
      ${
        peers.length
          ? `<div class="px-3 py-2 border-b border-base-300 flex flex-wrap gap-1">${peers
              .map(
                (p) =>
                  `<span class="badge badge-sm" style="border-color:${p.color};color:${p.color}">${escapeHtml(p.name)}</span>`,
              )
              .join("")}</div>`
          : ""
      }
      <div class="file-tree flex-1 min-h-0 flex flex-col p-3 overflow-hidden">
        <div class="flex justify-between items-center mb-2">
          <h3 class="font-bold">${escapeHtml(t(msgs, "editor.files"))}</h3>
          <div class="flex gap-1">
            <button type="button" id="sf-upload" class="btn btn-xs btn-primary">${escapeHtml(t(msgs, "editor.load"))}</button>
            <button type="button" id="sf-new" class="btn btn-xs btn-primary">+ ${escapeHtml(t(msgs, "editor.new_file"))}</button>
          </div>
        </div>
        <input id="sf-file" type="file" class="hidden" />
        <ul class="menu bg-base-100 rounded-box w-full flex-1 overflow-y-auto">${renderTree(tree, currentPath, expanded)}</ul>
      </div>
    `;
  }

  function paintChat() {
    chatEl.classList.toggle("hidden", !chatOpen);
    chatEl.classList.toggle("flex", chatOpen);
    if (!chatOpen) return;
    const items = collab.chat
      .map(
        (m: ChatMessage) =>
          `<div class="py-1"><span class="font-semibold text-primary mr-1.5">${escapeHtml(m.from)}</span>${escapeHtml(m.text)}</div>`,
      )
      .join("");
    chatEl.innerHTML = `
      <div class="flex items-center justify-between gap-2 min-h-12 px-3 border-b border-base-300 shrink-0">
        <div>
          <div class="text-sm font-semibold">${escapeHtml(t(msgs, "editor.chat"))}</div>
          <div class="text-xs text-base-content/50">${escapeHtml(t(msgs, "editor.chat_session_hint"))}</div>
        </div>
        <button type="button" id="sf-chat-close" class="btn btn-ghost btn-square btn-sm">${icon(ICO.x)}</button>
      </div>
      <div id="sf-chat-log" class="flex-1 min-h-0 overflow-y-auto px-4 py-3 text-sm">${
        collab.chat.length ? items : `<p class="text-base-content/60">${escapeHtml(t(msgs, "editor.chat_empty"))}</p>`
      }</div>
      <form id="sf-chat-form" class="flex flex-col gap-2 p-3 border-t border-base-300 shrink-0">
        <input id="sf-chat-input" type="text" class="input input-bordered input-sm w-full" placeholder="${escapeAttr(t(msgs, "editor.chat_placeholder"))}" autocomplete="off"/>
        <button class="btn btn-primary btn-sm" type="submit">${escapeHtml(t(msgs, "editor.chat_send"))}</button>
      </form>
    `;
    const log = document.getElementById("sf-chat-log");
    if (log) log.scrollTop = log.scrollHeight;
  }

  function paintCode() {
    pathEl.classList.toggle("hidden", !currentPath);
    pathEl.textContent = currentPath ?? "";
    view?.destroy();
    view = null;
    codeHost.replaceChildren();
    if (!currentPath) {
      codeHost.innerHTML = `<div class="flex items-center justify-center h-full text-base-content/70 p-4">${escapeHtml(t(msgs, "editor.select_file"))}</div>`;
      return;
    }
    const file = collab.files.find((f) => f.path === currentPath);
    const ytext = collab.getYText(currentPath);
    const body = ytext.toString();
    if (isBinaryContent(body, currentPath) || (file && file.size > 0 && body === "")) {
      const url = downloadPrefix + encodeURIComponent(currentPath);
      codeHost.innerHTML = `<div class="flex flex-col items-center justify-center h-full p-4">
        <p class="text-lg font-semibold mb-2">${escapeHtml(currentPath)}</p>
        <p class="text-sm text-base-content/70 mb-4">${escapeHtml(t(msgs, "editor.binary_file"))}</p>
        <a class="btn btn-primary" href="${url}">${escapeHtml(t(msgs, "editor.download"))}</a>
      </div>`;
      return;
    }
    if (!collab.initialSynced) {
      codeHost.innerHTML = `<div class="flex items-center justify-center h-full text-base-content/70 p-4">${escapeHtml(t(msgs, "editor.status_syncing"))}</div>`;
      return;
    }
    const host = document.createElement("div");
    host.className = "h-full min-h-0";
    codeHost.appendChild(host);
    view = mountEditor(host, ytext, collab.awareness, currentPath);
  }

  function paintLogs() {
    logsPre.textContent = logs || t(msgs, "editor.logs_empty");
  }

  function paintAll() {
    if (!currentPath && collab.files.length) {
      currentPath = collab.files[0]!.path;
    }
    paintStatus();
    paintSidebar();
    paintCode();
    paintLogs();
    paintChat();
    compileBtn.textContent = compiling ? t(msgs, "editor.compiling") : t(msgs, "editor.compile");
    compileBtn.classList.toggle("btn-disabled", compiling);
    sidebar.classList.toggle("hidden", !sidebarOpen);
    toggleBtn.innerHTML = icon(sidebarOpen ? ICO.x : ICO.menu);
    toggleBtn.setAttribute(
      "aria-label",
      t(msgs, sidebarOpen ? "editor.close_sidebar" : "editor.open_sidebar"),
    );
    chatToggle.classList.toggle("btn-active", chatOpen);
  }

  collab.subscribe(() => paintAll());

  toggleBtn.addEventListener("click", () => {
    sidebarOpen = !sidebarOpen;
    paintAll();
  });
  commitBtn.addEventListener("click", () => collab.commitNow());
  chatToggle.addEventListener("click", () => {
    chatOpen = !chatOpen;
    localStorage.setItem("superfolha-chat-open", chatOpen ? "1" : "0");
    paintAll();
  });
  compileBtn.addEventListener("click", async () => {
    if (!currentPath) {
      logs = t(msgs, "editor.no_file_compile");
      showTab("logs");
      paintLogs();
      return;
    }
    compiling = true;
    logs = "Compiling…\n";
    paintAll();
    try {
      const res = await fetch(
        (() => {
          const u = new URL(compileBase, window.location.origin);
          u.searchParams.set("file", currentPath);
          return u.pathname + u.search;
        })(),
        { credentials: "same-origin" },
      );
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        logs = (data as { message?: string }).message || t(msgs, "errors.COMPILE_FAILED");
        showTab("logs");
        return;
      }
      logs = (data as { logs?: string }).logs || "Compilation completed";
      const pdf = (data as { pdf?: string }).pdf;
      if (pdf) {
        if (pdfUrl) URL.revokeObjectURL(pdfUrl);
        pdfUrl = toPdfBlobUrl(pdf);
        pdfHost.innerHTML = `<iframe class="h-full w-full" title="PDF" src="${pdfUrl}"></iframe>`;
        showTab("pdf");
      } else if (!(data as { success?: boolean }).success) {
        showTab("logs");
      }
    } catch (e) {
      logs = `Error: ${e instanceof Error ? e.message : String(e)}`;
      showTab("logs");
    } finally {
      compiling = false;
      paintAll();
    }
  });

  sidebar.addEventListener("click", (ev) => {
    const el = ev.target as HTMLElement;
    const tabBtn = el.closest("[data-tab]") as HTMLElement | null;
    if (tabBtn?.dataset.tab) {
      showTab(tabBtn.dataset.tab as Tab);
      return;
    }
    const fileBtn = el.closest("[data-file]") as HTMLElement | null;
    if (fileBtn?.dataset.file) {
      currentPath = fileBtn.dataset.file;
      showTab("code");
      paintAll();
      return;
    }
    const dirBtn = el.closest("[data-dir]") as HTMLElement | null;
    if (dirBtn?.dataset.dir) {
      const p = dirBtn.dataset.dir;
      if (expanded.has(p)) expanded.delete(p);
      else expanded.add(p);
      paintSidebar();
      return;
    }
    const delBtn = el.closest("[data-del]") as HTMLElement | null;
    if (delBtn?.dataset.del) {
      collab.deleteFile(delBtn.dataset.del);
      if (currentPath === delBtn.dataset.del) currentPath = null;
      return;
    }
    if (el.id === "sf-new" || el.closest("#sf-new")) {
      const name = prompt(t(msgs, "editor.enter_file_name"));
      if (name) {
        collab.createFile(name, "");
        currentPath = name;
        showTab("code");
      }
      return;
    }
    if (el.id === "sf-upload" || el.closest("#sf-upload")) {
      (document.getElementById("sf-file") as HTMLInputElement | null)?.click();
    }
  });

  sidebar.addEventListener("change", async (ev) => {
    const input = ev.target as HTMLInputElement;
    if (input.id !== "sf-file" || !input.files?.[0]) return;
    const file = input.files[0];
    const form = new FormData();
    form.append("file", file);
    try {
      const res = await fetch(uploadURL, {
        method: "POST",
        body: form,
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(t(msgs, "editor.upload_failed", { Message: String(res.status) }));
      const result = (await res.json()) as { path?: string };
      if (result.path) {
        currentPath = result.path;
        showTab("code");
      }
    } catch (e) {
      alert(t(msgs, "editor.upload_failed", { Message: e instanceof Error ? e.message : String(e) }));
    } finally {
      input.value = "";
    }
  });

  chatEl.addEventListener("submit", (ev) => {
    ev.preventDefault();
    const input = document.getElementById("sf-chat-input") as HTMLInputElement | null;
    const text = input?.value.trim() ?? "";
    if (!text) return;
    collab.sendChat(text);
    if (input) input.value = "";
  });
  chatEl.addEventListener("click", (ev) => {
    if ((ev.target as HTMLElement).closest("#sf-chat-close")) {
      chatOpen = false;
      localStorage.setItem("superfolha-chat-open", "0");
      paintAll();
    }
  });

  paintAll();
}

boot();
