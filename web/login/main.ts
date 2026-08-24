import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha2.js";

try {
  ed.hashes.sha512 = sha512;
  ed.hashes.sha512Async = (m: Uint8Array) => Promise.resolve(sha512(m));
} catch (err) {
  console.error("superfolha login hashes", err);
}

const DB = "superfolha-login";
const STORE = "keys";
const KEY = "identity";

function $(id: string): HTMLElement | null {
  return document.getElementById(id);
}

function setStatus(text: string, isError: boolean): void {
  const el = $("sf-login-status");
  if (!el) return;
  el.textContent = text;
  el.classList.toggle("text-error", isError);
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB, 2);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE);
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

function idbGet(db: IDBDatabase): Promise<Uint8Array | null> {
  return new Promise((resolve, reject) => {
    const req = db.transaction(STORE, "readonly").objectStore(STORE).get(KEY);
    req.onsuccess = () => {
      const v = req.result;
      if (v instanceof Uint8Array) {
        resolve(v);
        return;
      }
      resolve(null);
    };
    req.onerror = () => reject(req.error);
  });
}

function idbPut(db: IDBDatabase, value: Uint8Array): Promise<void> {
  return new Promise((resolve, reject) => {
    const req = db.transaction(STORE, "readwrite").objectStore(STORE).put(value, KEY);
    req.onsuccess = () => resolve();
    req.onerror = () => reject(req.error);
  });
}

function b64url(buf: Uint8Array): string {
  let bin = "";
  for (let i = 0; i < buf.length; i++) bin += String.fromCharCode(buf[i]!);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

async function getSecretKey(): Promise<Uint8Array> {
  const db = await openDB();
  const existing = await idbGet(db);
  if (existing && existing.length === 32) {
    return existing;
  }
  const { secretKey } = ed.keygen();
  await idbPut(db, secretKey);
  return secretKey;
}

async function signIn(): Promise<void> {
  const root = $("sf-login");
  if (!root) return;
  const challengeURL = root.getAttribute("data-challenge") ?? "";
  const verifyURL = root.getAttribute("data-verify") ?? "";
  const next = root.getAttribute("data-next") ?? "";
  setStatus(root.getAttribute("data-working") || "Signing in…", false);
  try {
    const secretKey = await getSecretKey();
    const chRes = await fetch(challengeURL, { credentials: "same-origin" });
    if (!chRes.ok) throw new Error("challenge " + chRes.status);
    const chJSON = (await chRes.json()) as { challenge?: string };
    const challenge = chJSON.challenge;
    if (!challenge) throw new Error("empty challenge");
    const msg = new TextEncoder().encode(challenge);
    const sig = ed.sign(msg, secretKey);
    const pub = ed.getPublicKey(secretKey);
    const res = await fetch(verifyURL, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        challenge,
        public_key: b64url(pub),
        signature: b64url(sig),
        next,
      }),
    });
    if (!res.ok) throw new Error("verify " + res.status);
    const out = (await res.json()) as { next?: string };
    window.location.assign(out.next || "/sessions");
  } catch (err) {
    console.error("superfolha login", err);
    setStatus(root.getAttribute("data-failed") || "Sign-in failed.", true);
  }
}

(globalThis as unknown as { __sfLogin?: () => void }).__sfLogin = () => {
  void signIn();
};

function bind(): void {
  const btn = $("sf-login-btn");
  if (!btn || btn.dataset.bound === "1") return;
  btn.dataset.bound = "1";
  btn.addEventListener("click", (ev) => {
    ev.preventDefault();
    void signIn();
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", bind);
} else {
  bind();
}
