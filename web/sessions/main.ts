import {
  authorized,
  b64url,
  b64urlDecode,
  decodeStdB64,
  encodeStdB64,
  idbPut,
  publicLine,
  seedFor,
  signSSH,
  storageKey,
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
  const remoteEl = input("sf-remote");
  const branchEl = input("sf-branch");
  const pubHidden = input("sf-ssh-public");
  const pubEl = textarea("sf-ssh-pub");
  if (!remoteEl || !branchEl || !pubHidden || !pubEl) return;
  const remote = remoteEl.value.trim();
  const branch = branchEl.value.trim() || "main";
  if (!remote) {
    pubHidden.value = "";
    pubEl.value = "";
    return;
  }
  const seed = await seedFor(remote, branch);
  const line = publicLine(seed);
  pubHidden.value = line;
  pubEl.value = line;
}

function store(): void {
  const remote = input("sf-remote")?.value.trim() ?? "";
  const branch = input("sf-branch")?.value.trim() || "main";
  const pub = textarea("sf-ssh-pub")?.value ?? "";
  void seedFor(remote, branch).then((seed) => {
    const blob = new Blob(
      [JSON.stringify({ v: 1, remote, branch, seed: b64url(seed), public: pub }, null, 2)],
      { type: "application/json" },
    );
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "superfolha-ssh.json";
    a.click();
    URL.revokeObjectURL(a.href);
  });
}

async function loadFile(file: File): Promise<void> {
  const obj = JSON.parse(await file.text()) as {
    remote?: string;
    branch?: string;
    seed?: string;
  };
  if (!obj.seed) throw new Error("missing seed");
  const seed = b64urlDecode(obj.seed);
  if (seed.length !== 32) throw new Error("bad seed");
  const remoteEl = input("sf-remote");
  const branchEl = input("sf-branch");
  if (remoteEl && obj.remote) remoteEl.value = obj.remote;
  if (branchEl && obj.branch) branchEl.value = obj.branch;
  const remote = remoteEl?.value.trim() ?? "";
  const branch = branchEl?.value.trim() || "main";
  if (!remote) throw new Error("missing remote");
  await idbPut(storageKey(remote, branch), seed);
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
  void seedFor(remote, branch)
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
    branchEl?.addEventListener("input", () => {
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
