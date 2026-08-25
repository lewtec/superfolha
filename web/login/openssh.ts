import { ctr } from "@noble/ciphers/aes.js";
import * as ed from "@noble/ed25519";
import bpf from "bcrypt-pbkdf";

const MAGIC = "openssh-key-v1\0";
const BEGIN = "-----BEGIN OPENSSH PRIVATE KEY-----";
const END = "-----END OPENSSH PRIVATE KEY-----";

class R {
  i = 0;
  constructor(readonly b: Uint8Array) {}
  u32(): number {
    if (this.i + 4 > this.b.length) throw new Error("truncated");
    const v =
      ((this.b[this.i]! << 24) |
        (this.b[this.i + 1]! << 16) |
        (this.b[this.i + 2]! << 8) |
        this.b[this.i + 3]!) >>>
      0;
    this.i += 4;
    return v;
  }
  take(n: number): Uint8Array {
    if (this.i + n > this.b.length) throw new Error("truncated");
    const s = this.b.subarray(this.i, this.i + n);
    this.i += n;
    return s;
  }
  str(): string {
    return new TextDecoder().decode(this.buf());
  }
  buf(): Uint8Array {
    return this.take(this.u32());
  }
}

class W {
  readonly parts: Uint8Array[] = [];
  u32(n: number): void {
    const b = new Uint8Array(4);
    new DataView(b.buffer).setUint32(0, n);
    this.parts.push(b);
  }
  raw(b: Uint8Array): void {
    this.parts.push(b);
  }
  str(s: string): void {
    this.buf(new TextEncoder().encode(s));
  }
  buf(b: Uint8Array): void {
    this.u32(b.length);
    this.raw(b);
  }
  finish(): Uint8Array {
    let n = 0;
    for (const p of this.parts) n += p.length;
    const out = new Uint8Array(n);
    let off = 0;
    for (const p of this.parts) {
      out.set(p, off);
      off += p.length;
    }
    return out;
  }
}

function fromB64(s: string): Uint8Array {
  const bin = atob(s);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function toB64(buf: Uint8Array): string {
  let bin = "";
  for (let i = 0; i < buf.length; i++) bin += String.fromCharCode(buf[i]!);
  return btoa(bin);
}

function sshPub(pub: Uint8Array): Uint8Array {
  const w = new W();
  w.str("ssh-ed25519");
  w.buf(pub);
  return w.finish();
}

function bcryptPbkdf(pass: Uint8Array, salt: Uint8Array, outLen: number, rounds: number): Uint8Array {
  const key = new Uint8Array(outLen);
  const rc = bpf.pbkdf(pass, pass.length, salt, salt.length, key, key.length, rounds);
  if (rc !== 0) throw new Error("kdf");
  return key;
}

function decryptPriv(priv: Uint8Array, cipher: string, kdf: string, kdfopts: Uint8Array, pass: string): Uint8Array {
  if (cipher === "none" && kdf === "none") return priv;
  if (kdf !== "bcrypt") throw new Error("unsupported kdf");
  if (cipher !== "aes256-ctr") throw new Error("unsupported cipher");
  const kr = new R(kdfopts);
  const salt = kr.buf();
  const rounds = kr.u32();
  const material = bcryptPbkdf(new TextEncoder().encode(pass), salt, 32 + 16, rounds);
  return ctr(material.subarray(0, 32), material.subarray(32, 48)).decrypt(priv);
}

function seedFromPrivBlock(priv: Uint8Array): Uint8Array {
  const p = new R(priv);
  const c1 = p.u32();
  const c2 = p.u32();
  if (c1 !== c2) throw new Error("bad passphrase");
  if (p.str() !== "ssh-ed25519") throw new Error("not ed25519");
  p.buf();
  const secret = p.buf();
  if (secret.length < 32) throw new Error("short secret");
  return secret.subarray(0, 32);
}

/** Extract a 32-byte Ed25519 seed from an OpenSSH private key, raw seed, or JSON {seed}. */
export function parseIdentitySeed(text: string, passphrase: string): Uint8Array {
  const t = text.trim();
  if (t.startsWith("{")) {
    const obj = JSON.parse(t) as { seed?: string };
    if (!obj.seed) throw new Error("missing seed");
    const pad = "=".repeat((4 - (obj.seed.length % 4)) % 4);
    const raw = fromB64(obj.seed.replace(/-/g, "+").replace(/_/g, "/") + pad);
    if (raw.length !== 32) throw new Error("bad seed");
    return raw;
  }
  if (t.includes(BEGIN)) {
    const b64 = t.replace(/-----[^-]+-----/g, "").replace(/\s+/g, "");
    const r = new R(fromB64(b64));
    const magic = new TextDecoder().decode(r.take(MAGIC.length));
    if (magic !== MAGIC) throw new Error("bad magic");
    const cipher = r.str();
    const kdf = r.str();
    const kdfopts = r.buf();
    if (r.u32() !== 1) throw new Error("one key only");
    r.buf();
    const enc = r.buf();
    const priv = decryptPriv(enc, cipher, kdf, kdfopts, passphrase);
    return seedFromPrivBlock(priv);
  }
  throw new Error("not an OpenSSH private key");
}

/** Write an unencrypted OpenSSH id_ed25519 PEM from a 32-byte seed. */
export function encodeIdentityKey(seed: Uint8Array, comment = "superfolha"): string {
  if (seed.length !== 32) throw new Error("bad seed");
  const pub = ed.getPublicKey(seed);
  const secret = new Uint8Array(64);
  secret.set(seed, 0);
  secret.set(pub, 32);
  const check = cryptoGetRandomU32();
  const inner = new W();
  inner.u32(check);
  inner.u32(check);
  inner.str("ssh-ed25519");
  inner.buf(pub);
  inner.buf(secret);
  inner.str(comment);
  let block = inner.finish();
  const padn = (8 - (block.length % 8)) % 8;
  if (padn) {
    const pad = new Uint8Array(padn);
    for (let i = 0; i < padn; i++) pad[i] = i + 1;
    const joined = new Uint8Array(block.length + padn);
    joined.set(block, 0);
    joined.set(pad, block.length);
    block = joined;
  }
  const outer = new W();
  outer.raw(new TextEncoder().encode(MAGIC));
  outer.str("none");
  outer.str("none");
  outer.buf(new Uint8Array(0));
  outer.u32(1);
  outer.buf(sshPub(pub));
  outer.buf(block);
  const b64 = toB64(outer.finish());
  const lines = [BEGIN];
  for (let i = 0; i < b64.length; i += 70) lines.push(b64.slice(i, i + 70));
  lines.push(END);
  return lines.join("\n") + "\n";
}

function cryptoGetRandomU32(): number {
  const b = new Uint8Array(4);
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(b);
  } else {
    for (let i = 0; i < 4; i++) b[i] = (Math.random() * 256) | 0;
  }
  return ((b[0]! << 24) | (b[1]! << 16) | (b[2]! << 8) | b[3]!) >>> 0;
}
