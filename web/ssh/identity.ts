import * as ed from "@noble/ed25519";
import { sha256, sha512 } from "@noble/hashes/sha2.js";

try {
  ed.hashes.sha512 = sha512;
  ed.hashes.sha512Async = (m: Uint8Array) => Promise.resolve(sha512(m));
} catch (err) {
  console.error("superfolha identity hashes", err);
}

const DB = "superfolha-login";
const STORE = "keys";
const ROSTER = "roster";
const LEGACY = "identity";
const ACTIVE = "superfolha.activeIdentity";

export type IdentityRec = { id: string; seed: Uint8Array };

function hex(buf: Uint8Array): string {
  let s = "";
  for (let i = 0; i < buf.length; i++) s += buf[i]!.toString(16).padStart(2, "0");
  return s;
}

export function fingerprint(pub: Uint8Array): string {
  return "ed25519:" + hex(sha256(pub).subarray(0, 8));
}

export function idFromSeed(seed: Uint8Array): string {
  return fingerprint(ed.getPublicKey(seed));
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB, 3);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE);
      }
      if (!db.objectStoreNames.contains(ROSTER)) {
        db.createObjectStore(ROSTER, { keyPath: "id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function migrateLegacy(db: IDBDatabase): Promise<void> {
  const seed = await new Promise<Uint8Array | null>((resolve, reject) => {
    if (!db.objectStoreNames.contains(STORE)) {
      resolve(null);
      return;
    }
    const req = db.transaction(STORE, "readonly").objectStore(STORE).get(LEGACY);
    req.onsuccess = () => {
      const v = req.result;
      resolve(v instanceof Uint8Array && v.length === 32 ? v : null);
    };
    req.onerror = () => reject(req.error);
  });
  if (!seed) return;
  const rec: IdentityRec = { id: idFromSeed(seed), seed };
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction([ROSTER, STORE], "readwrite");
    tx.objectStore(ROSTER).put(rec);
    tx.objectStore(STORE).delete(LEGACY);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
  if (!sessionStorage.getItem(ACTIVE)) {
    sessionStorage.setItem(ACTIVE, rec.id);
  }
}

export async function listIdentities(): Promise<IdentityRec[]> {
  const db = await openDB();
  await migrateLegacy(db);
  return new Promise((resolve, reject) => {
    const req = db.transaction(ROSTER, "readonly").objectStore(ROSTER).getAll();
    req.onsuccess = () => {
      const rows = (req.result as IdentityRec[]) || [];
      resolve(rows.filter((r) => r?.id && r.seed instanceof Uint8Array && r.seed.length === 32));
    };
    req.onerror = () => reject(req.error);
  });
}

export function getActiveId(): string | null {
  return sessionStorage.getItem(ACTIVE);
}

export function setActiveId(id: string): void {
  sessionStorage.setItem(ACTIVE, id);
}

export function clearActiveId(): void {
  sessionStorage.removeItem(ACTIVE);
}

/** Prefer the server cookie identity when it matches a roster row. */
export function syncActiveFromLogin(login: string): void {
  if (login) sessionStorage.setItem(ACTIVE, login);
}

export async function getLoginSeed(): Promise<Uint8Array | null> {
  const all = await listIdentities();
  if (all.length === 0) return null;
  const active = getActiveId();
  const rec = all.find((r) => r.id === active) ?? all[0]!;
  if (rec.id !== active) setActiveId(rec.id);
  return rec.seed;
}

export async function addIdentity(seed: Uint8Array): Promise<string> {
  if (seed.length !== 32) throw new Error("bad seed");
  const rec: IdentityRec = { id: idFromSeed(seed), seed };
  const db = await openDB();
  await migrateLegacy(db);
  await new Promise<void>((resolve, reject) => {
    const req = db.transaction(ROSTER, "readwrite").objectStore(ROSTER).put(rec);
    req.onsuccess = () => resolve();
    req.onerror = () => reject(req.error);
  });
  setActiveId(rec.id);
  return rec.id;
}

export async function putLoginSeed(seed: Uint8Array): Promise<void> {
  await addIdentity(seed);
}

export async function getOrCreateLoginSeed(): Promise<Uint8Array> {
  const existing = await getLoginSeed();
  if (existing) return existing;
  const { secretKey } = ed.keygen();
  await addIdentity(secretKey);
  return secretKey;
}

export function b64url(buf: Uint8Array): string {
  let bin = "";
  for (let i = 0; i < buf.length; i++) bin += String.fromCharCode(buf[i]!);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

export async function verifyLogin(
  seed: Uint8Array,
  challengeURL: string,
  verifyURL: string,
  next = "",
): Promise<string> {
  const chRes = await fetch(challengeURL, { credentials: "same-origin" });
  if (!chRes.ok) throw new Error("challenge " + chRes.status);
  const chJSON = (await chRes.json()) as { challenge?: string };
  const challenge = chJSON.challenge;
  if (!challenge) throw new Error("empty challenge");
  const msg = new TextEncoder().encode(challenge);
  const sig = ed.sign(msg, seed);
  const pub = ed.getPublicKey(seed);
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
  return out.next || "/sessions";
}
