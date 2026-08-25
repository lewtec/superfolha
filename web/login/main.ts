import { encodeIdentityKey, parseIdentitySeed } from "./openssh";
import {
  addIdentity,
  getOrCreateLoginSeed,
  verifyLogin,
} from "../ssh/identity";

function $(id: string): HTMLElement | null {
  return document.getElementById(id);
}

function setStatus(text: string, isError: boolean): void {
  const el = $("sf-login-status");
  if (!el) return;
  el.textContent = text;
  el.classList.toggle("text-error", isError);
}

async function signIn(): Promise<void> {
  const root = $("sf-login");
  if (!root) return;
  const challengeURL = root.getAttribute("data-challenge") ?? "";
  const verifyURL = root.getAttribute("data-verify") ?? "";
  const next = root.getAttribute("data-next") ?? "";
  setStatus(root.getAttribute("data-working") || "Signing in…", false);
  try {
    const secretKey = await getOrCreateLoginSeed();
    const dest = await verifyLogin(secretKey, challengeURL, verifyURL, next);
    window.location.assign(dest);
  } catch (err) {
    console.error("superfolha login", err);
    setStatus(root.getAttribute("data-failed") || "Sign-in failed.", true);
  }
}

async function loadKey(file: File): Promise<void> {
  const root = $("sf-login");
  const pass = (document.getElementById("sf-login-pass") as HTMLInputElement | null)?.value ?? "";
  try {
    const seed = parseIdentitySeed(await file.text(), pass);
    await addIdentity(seed);
    setStatus(root?.getAttribute("data-loaded") || "Key loaded. Sign in.", false);
  } catch (err) {
    console.error("superfolha login load", err);
    setStatus(root?.getAttribute("data-bad-key") || "That file is not an Ed25519 OpenSSH key.", true);
  }
}

async function storeKey(): Promise<void> {
  const root = $("sf-login");
  try {
    const seed = await getOrCreateLoginSeed();
    const pem = encodeIdentityKey(seed);
    const blob = new Blob([pem], { type: "application/x-pem-file" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "id_ed25519";
    a.click();
    URL.revokeObjectURL(a.href);
    setStatus(root?.getAttribute("data-stored") || "Saved id_ed25519.", false);
  } catch (err) {
    console.error("superfolha login store", err);
    setStatus(root?.getAttribute("data-failed") || "Sign-in failed.", true);
  }
}

(globalThis as unknown as { __sfLogin?: () => void }).__sfLogin = () => {
  void signIn();
};

function bind(): void {
  const btn = $("sf-login-btn");
  const loadBtn = $("sf-login-load");
  const storeBtn = $("sf-login-store");
  const fileEl = document.getElementById("sf-login-file") as HTMLInputElement | null;
  if (!btn || btn.dataset.bound === "1") return;
  btn.dataset.bound = "1";
  btn.addEventListener("click", (ev) => {
    ev.preventDefault();
    void signIn();
  });
  loadBtn?.addEventListener("click", (ev) => {
    ev.preventDefault();
    fileEl?.click();
  });
  storeBtn?.addEventListener("click", (ev) => {
    ev.preventDefault();
    void storeKey();
  });
  fileEl?.addEventListener("change", () => {
    const file = fileEl.files?.[0];
    fileEl.value = "";
    if (!file) return;
    void loadKey(file);
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", bind);
} else {
  bind();
}
