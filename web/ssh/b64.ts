function bytesToBinary(buf: Uint8Array): string {
  let bin = "";
  for (let i = 0; i < buf.length; i++) bin += String.fromCharCode(buf[i]!);
  return bin;
}

function binaryToBytes(bin: string): Uint8Array {
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

export function encodeStdB64(buf: Uint8Array): string {
  return btoa(bytesToBinary(buf));
}

export function decodeStdB64(s: string): Uint8Array {
  return binaryToBytes(atob(s));
}

export function b64url(buf: Uint8Array): string {
  return encodeStdB64(buf).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

export function b64urlDecode(s: string): Uint8Array {
  const pad = "=".repeat((4 - (s.length % 4)) % 4);
  return decodeStdB64(s.replace(/-/g, "+").replace(/_/g, "/") + pad);
}
