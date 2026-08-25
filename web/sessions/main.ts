import { encodeIdentityKey, parseIdentitySeed } from "../login/openssh";
import { getOrCreateLoginSeed, putLoginSeed } from "../ssh/identity";
import { forgetRecent, loadRecents, recentKey, rememberRecent } from "./recents";
import {
  authorized,
  decodeStdB64,
  encodeStdB64,
  publicLine,
  signSSH,
} from "../ssh/sessionKey";
import * as ed from "@noble/ed25519";

function $(id: string): HTMLElement | null {
  return document.getElementById(id);
}

function input(id: string): HTMLInputElement | null {
  const el = $(id);
  return el instanceof HTMLInputElement ? el : null;
}

function textarea(id: string): HTMLTextAreaElement | null {
  const el = $(id);
  return el instanceof HTMLTextAreaElement ? el : null;
}

async function refresh(): Promise<void> {
  const pubHidden = input("sf-ssh-public");
  const pubEl = textarea("sf-ssh-pub");
  if (!pubHidden || !pubEl) return;
  const seed = await getOrCreateLoginSeed();
  const line = publicLine(seed);
  pubHidden.value = line;
  pubEl.value = line;
}

function store(): void {
  void getOrCreateLoginSeed().then((seed) => {
    const pem = encodeIdentityKey(seed);
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([pem], { type: "application/x-pem-file" }));
    a.download = "id_ed25519";
    a.click();
    URL.revokeObjectURL(a.href);
  });
}

async function loadFile(file: File): Promise<void> {
  const pass = input("sf-ssh-pass")?.value ?? "";
  const seed = parseIdentitySeed(await file.text(), pass);
  await putLoginSeed(seed);
  await refresh();
}

function cloneToast(): HTMLElement | null {
  return $("sf-clone-toast");
}

function setCloneStatus(text: string, isError: boolean): void {
  const toast = cloneToast();
  const alertEl = $("sf-clone-alert");
  const spin = $("sf-clone-spin");
  const msg = $("sf-clone-msg");
  if (!toast || !msg) return;
  const shown = localizeClone(toast, text);
  msg.textContent = shown;
  toast.classList.toggle("hidden", !shown);
  alertEl?.classList.toggle("alert-error", isError);
  spin?.classList.toggle("hidden", isError || !shown);
}

function localizeClone(el: HTMLElement, text: string): string {
  if (text === "sessions.key_unauthorized") return el.getAttribute("data-unauthorized") || text;
  if (text === "sessions.key_read_only") return el.getAttribute("data-readonly") || text;
  if (text === "sessions.cloning") return el.getAttribute("data-working") || text;
  return text;
}

function startClone(btn: HTMLElement): void {
  const wsPath = btn.getAttribute("data-ws") ?? "";
  const editor = btn.getAttribute("data-editor") ?? "";
  const remote = btn.getAttribute("data-remote") || input("sf-remote")?.value.trim() || "";
  const branch = btn.getAttribute("data-branch") || input("sf-branch")?.value.trim() || "main";
  const statusEl = cloneToast();
  const working = statusEl?.getAttribute("data-working") || "Cloning…";
  const failed = statusEl?.getAttribute("data-failed") || "Clone failed.";
  const timedOut = statusEl?.getAttribute("data-timeout") || "Clone timed out.";
  if (!wsPath || !remote) {
    setCloneStatus(failed, true);
    return;
  }
  btn.setAttribute("disabled", "true");
  btn.classList.add("btn-disabled");
  setCloneStatus(working, false);
  let settled = false;
  const finish = (ok: boolean, text: string) => {
    if (settled) return;
    settled = true;
    btn.removeAttribute("disabled");
    btn.classList.remove("btn-disabled");
    setCloneStatus(text, !ok);
  };
  void getOrCreateLoginSeed()
    .then((seed) => {
      const pub = authorized(ed.getPublicKey(seed));
      const proto = location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${proto}//${location.host}${wsPath}`);
      const timer = window.setTimeout(() => {
        ws.close();
        finish(false, timedOut);
      }, 60_000);
      ws.onmessage = (ev) => {
        if (typeof ev.data !== "string") return;
        let msg: Record<string, string>;
        try {
          msg = JSON.parse(ev.data) as Record<string, string>;
        } catch {
          return;
        }
        if (msg.type === "ssh.sign" && msg.id && msg.data) {
          try {
            const sig = signSSH(seed, decodeStdB64(msg.data));
            ws.send(JSON.stringify({ type: "ssh.sign.ok", id: msg.id, signature: encodeStdB64(sig) }));
          } catch (err) {
            ws.send(JSON.stringify({
              type: "ssh.sign.err",
              id: msg.id,
              message: err instanceof Error ? err.message : "sign failed",
            }));
          }
          return;
        }
        if (msg.type === "clone.status" && msg.message) {
          setCloneStatus(msg.message === "sessions.cloning" ? working : msg.message, false);
          return;
        }
        if (msg.type === "clone.ok") {
          window.clearTimeout(timer);
          settled = true;
          ws.close();
          location.assign(editor);
          return;
        }
        if (msg.type === "clone.err") {
          window.clearTimeout(timer);
          ws.close();
          const pubEl = textarea("sf-ssh-pub");
          if (pubEl && msg.ssh_public) pubEl.value = msg.ssh_public;
          finish(false, msg.message || failed);
          const modal = document.getElementById("ssh-deploy-modal");
          if (modal instanceof HTMLDialogElement) modal.showModal();
        }
      };
      ws.onopen = () => {
        ws.send(JSON.stringify({ type: "clone", public: pub }));
      };
      ws.onerror = () => {
        window.clearTimeout(timer);
        finish(false, failed);
      };
      ws.onclose = () => {
        window.clearTimeout(timer);
        if (!settled) finish(false, failed);
      };
    })
    .catch((err) => {
      finish(false, err instanceof Error ? err.message : failed);
    });
}

function liveFor(remote: string, branch: string): { editor: string; ready: boolean } | null {
  const want = recentKey(remote, branch);
  for (const el of document.querySelectorAll<HTMLElement>("[data-sf-live]")) {
    if (recentKey(el.dataset.remote || "", el.dataset.branch || "") !== want) continue;
    return { editor: el.dataset.editor || "", ready: el.dataset.ready === "1" };
  }
  return null;
}

function paintRecents(): void {
  const wrap = $("sf-recents-wrap");
  const list = $("sf-recents");
  if (!wrap || !list) return;
  const resumeL = list.getAttribute("data-label-resume") || "Resume";
  const openL = list.getAttribute("data-label-open") || "Open";
  const forgetL = list.getAttribute("data-label-forget") || "Remove";
  const rows = loadRecents();
  wrap.classList.toggle("hidden", rows.length === 0);
  list.innerHTML = rows
    .map((r) => {
      const live = liveFor(r.remote, r.branch);
      const action = live?.ready ? openL : resumeL;
      const remote = esc(r.remote);
      const branch = esc(r.branch);
      return `<li class="list-row">
        <div class="list-col-grow min-w-0">
          <div class="font-mono text-sm break-all">${remote}</div>
          <div class="text-xs text-base-content/60">${branch}</div>
        </div>
        <button type="button" class="btn btn-sm btn-primary" data-resume="${remote}" data-branch="${branch}">${esc(action)}</button>
        <button type="button" class="btn btn-sm btn-ghost" data-forget="${remote}" data-branch="${branch}" aria-label="${esc(forgetL)}">×</button>
      </li>`;
    })
    .join("");
}

function esc(s: string): string {
  return s.replace(/[&<>"']/g, (ch) => {
    switch (ch) {
      case "&":
        return "&amp;";
      case "<":
        return "&lt;";
      case ">":
        return "&gt;";
      case '"':
        return "&quot;";
      default:
        return "&#39;";
    }
  });
}

function resume(remote: string, branch: string): void {
  rememberRecent(remote, branch);
  const live = liveFor(remote, branch);
  if (live?.ready && live.editor) {
    location.assign(live.editor);
    return;
  }
  const remoteEl = input("sf-remote");
  const branchEl = input("sf-branch");
  const form = $("sf-session-form");
  if (!remoteEl || !form || !(form instanceof HTMLFormElement)) {
    setCloneStatus("Cannot resume.", true);
    return;
  }
  remoteEl.value = remote;
  if (branchEl) branchEl.value = branch;
  const working = cloneToast()?.getAttribute("data-working") || "Cloning…";
  setCloneStatus(working, false);
  void refresh()
    .then(() => form.submit())
    .catch((err) => {
      console.error("superfolha resume", err);
      setCloneStatus(err instanceof Error ? err.message : "Cannot resume.", true);
    });
}

function bind(): void {
  const form = $("sf-session-form");
  const remoteEl = input("sf-remote");
  const branchEl = input("sf-branch");
  const storeBtn = $("sf-ssh-store");
  const loadBtn = $("sf-ssh-load");
  const fileEl = input("sf-ssh-file");
  if (form && form.dataset.bound !== "1") {
    form.dataset.bound = "1";
    remoteEl?.addEventListener("input", () => {
      void refresh();
    });
    storeBtn?.addEventListener("click", (ev) => {
      ev.preventDefault();
      store();
    });
    loadBtn?.addEventListener("click", (ev) => {
      ev.preventDefault();
      fileEl?.click();
    });
    fileEl?.addEventListener("change", () => {
      const file = fileEl.files?.[0];
      fileEl.value = "";
      if (!file) return;
      void loadFile(file).catch((err) => {
        console.error("superfolha ssh load", err);
      });
    });
    form.addEventListener("submit", (ev) => {
      ev.preventDefault();
      rememberRecent(remoteEl?.value ?? "", branchEl?.value ?? "");
      void refresh().then(() => {
        if (form instanceof HTMLFormElement) form.submit();
      });
    });
    void refresh();
  }
  const recents = $("sf-recents");
  if (recents && recents.dataset.bound !== "1") {
    recents.dataset.bound = "1";
    recents.addEventListener("click", (ev) => {
      const t = ev.target;
      if (!(t instanceof HTMLElement)) return;
      const forget = t.closest("button[data-forget]") as HTMLElement | null;
      if (forget?.dataset.forget) {
        ev.preventDefault();
        forgetRecent(forget.dataset.forget, forget.dataset.branch || "main");
        paintRecents();
        return;
      }
      const go = t.closest("button[data-resume]") as HTMLElement | null;
      if (go?.dataset.resume) {
        ev.preventDefault();
        resume(go.dataset.resume, go.dataset.branch || "main");
      }
    });
    paintRecents();
  }
  document.querySelectorAll<HTMLElement>("[data-sf-live]").forEach((card) => {
    if (card.dataset.remembered === "1") return;
    card.dataset.remembered = "1";
    card.querySelector("a[href]")?.addEventListener("click", () => {
      rememberRecent(card.dataset.remote || "", card.dataset.branch || "");
    });
  });
  document.querySelectorAll<HTMLElement>("[data-ws][data-editor]").forEach((btn) => {
    if (btn.dataset.bound === "1") return;
    btn.dataset.bound = "1";
    btn.addEventListener("click", (ev) => {
      ev.preventDefault();
      startClone(btn);
    });
  });
  const auto = $("sf-ssh-retry");
  if (auto) startClone(auto);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", bind);
} else {
  bind();
}
