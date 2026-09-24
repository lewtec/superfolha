import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha2.js";
import { encodeStdB64 } from "./b64";

export { b64url, b64urlDecode, decodeStdB64, encodeStdB64 } from "./b64";

try {
  ed.hashes.sha512 = sha512;
  ed.hashes.sha512Async = (m: Uint8Array) => Promise.resolve(sha512(m));
} catch (err) {
  console.error("superfolha ssh hashes", err);
}

const DB = "superfolha-ssh";
const STORE = "keys";

export function storageKey(remote: string, branch: string): string {
  return `${remote.trim().replace(/\.git$/, "")}\n${branch.trim() || "main"}`;
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB, 1);
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

export function idbGet(key: string): Promise<Uint8Array | null> {
  return openDB().then(
    (db) =>
      new Promise((resolve, reject) => {
        const req = db.transaction(STORE, "readonly").objectStore(STORE).get(key);
        req.onsuccess = () => {
          const v = req.result;
          resolve(v instanceof Uint8Array ? v : null);
        };
        req.onerror = () => reject(req.error);
      }),
  );
}

export function idbPut(key: string, value: Uint8Array): Promise<void> {
  return openDB().then(
    (db) =>
      new Promise((resolve, reject) => {
        const req = db.transaction(STORE, "readwrite").objectStore(STORE).put(value, key);
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
      }),
  );
}

function sshString(data: Uint8Array): Uint8Array {
  const out = new Uint8Array(4 + data.length);
  new DataView(out.buffer).setUint32(0, data.length);
  out.set(data, 4);
  return out;
}

function concat(...parts: Uint8Array[]): Uint8Array {
  let n = 0;
  for (const p of parts) n += p.length;
  const out = new Uint8Array(n);
  let off = 0;
  for (const p of parts) {
    out.set(p, off);
    off += p.length;
  }
  return out;
}

export function authorized(pub: Uint8Array): string {
  const algo = new TextEncoder().encode("ssh-ed25519");
  const payload = concat(sshString(algo), sshString(pub));
  return `ssh-ed25519 ${encodeStdB64(payload)} superfolha`;
}

export async function seedFor(remote: string, branch: string): Promise<Uint8Array> {
  const key = storageKey(remote, branch);
  const existing = await idbGet(key);
  if (existing && existing.length === 32) {
    return existing;
  }
  const { secretKey } = ed.keygen();
  await idbPut(key, secretKey);
  return secretKey;
}

export function publicLine(seed: Uint8Array): string {
  return authorized(ed.getPublicKey(seed));
}

export function signSSH(seed: Uint8Array, data: Uint8Array): Uint8Array {
  return ed.sign(data, seed);
}

