/** Must match internal/crdt.TextKey on the server. */
export function textKey(relPath: string): string {
  return "text:" + relPath.replace(/\\/g, "/");
}
