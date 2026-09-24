import { expect, test } from "bun:test";
import { b64url, b64urlDecode, decodeStdB64, encodeStdB64 } from "./b64";

const raw = new Uint8Array([0x14, 0xfb, 0x9c, 0x03, 0xd9, 0x7e]);

test("std base64 matches the previous byte loop", () => {
  expect(encodeStdB64(raw)).toBe("FPucA9l+");
  expect(decodeStdB64("FPucA9l+")).toEqual(raw);
});

test("base64url strips padding and swaps alphabet", () => {
  expect(b64url(raw)).toBe("FPucA9l-");
  expect(b64urlDecode("FPucA9l-")).toEqual(raw);
});
