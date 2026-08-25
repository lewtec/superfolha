import { encodeIdentityKey, parseIdentitySeed } from "../login/openssh";
import { getOrCreateLoginSeed, putLoginSeed } from "../ssh/identity";
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

function cloneStatusEls(): HTMLElement[] {
  return Array.from(document.querySelectorAll<HTMLElement>(".sf-clone-status"));
}

function setCloneStatus(text: string, isError: boolean): void {
  for (const el of cloneStatusEls()) {
    el.textContent = text;
    el.classList.toggle("text-error", isError);
  }
}

function startClone(btn: HTMLElement): void {
  const wsPath = btn.getAttribute("data-ws") ?? "";
  const editor = btn.getAttribute("data-editor") ?? "";
  const remote = btn.getAttribute("data-remote") || input("sf-remote")?.value.trim() || "";
  const branch = btn.getAttribute("data-branch") || input("sf-branch")?.value.trim() || "main";
  const statusEl = cloneStatusEls()[0];
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
      void refresh().then(() => {
        if (form instanceof HTMLFormElement) form.submit();
      });
    });
    void refresh();
  }
  document.querySelectorAll<HTMLElement>("[data-ws][data-editor]").forEach((btn) => {
    if (btn.dataset.bound === "1") return;
    btn.dataset.bound = "1";
    btn.addEventListener("click", (ev) => {
      ev.preventDefault();
      startClone(btn);
    });
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", bind);
} else {
  bind();
}
