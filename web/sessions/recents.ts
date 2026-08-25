import { getActiveId } from "../ssh/identity";

export type Recent = { remote: string; branch: string; at: number };

const PREFIX = "superfolha.recents.";
const CAP = 12;

export function canonRemote(remote: string): string {
  return remote.trim().replace(/\.git$/, "");
}

export function recentKey(remote: string, branch: string): string {
  return `${canonRemote(remote)}\n${(branch || "main").trim() || "main"}`;
}

function storeKey(): string {
  return PREFIX + (getActiveId() || "anon");
}

export function loadRecents(): Recent[] {
  try {
    const raw = localStorage.getItem(storeKey());
    if (!raw) return [];
    const rows = JSON.parse(raw) as Recent[];
    if (!Array.isArray(rows)) return [];
    return rows.filter((r) => r && typeof r.remote === "string" && r.remote);
  } catch {
    return [];
  }
}

export function rememberRecent(remote: string, branch: string): void {
  const rec: Recent = {
    remote: canonRemote(remote),
    branch: (branch || "main").trim() || "main",
    at: Date.now(),
  };
  if (!rec.remote) return;
  const rest = loadRecents().filter((r) => recentKey(r.remote, r.branch) !== recentKey(rec.remote, rec.branch));
  localStorage.setItem(storeKey(), JSON.stringify([rec, ...rest].slice(0, CAP)));
}

export function forgetRecent(remote: string, branch: string): void {
  const key = recentKey(remote, branch);
  const next = loadRecents().filter((r) => recentKey(r.remote, r.branch) !== key);
  localStorage.setItem(storeKey(), JSON.stringify(next));
}
