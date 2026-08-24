import * as Y from "yjs";
import * as awarenessProtocol from "y-protocols/awareness";
import * as syncProtocol from "y-protocols/sync";
import * as encoding from "lib0/encoding";
import * as decoding from "lib0/decoding";
import { textKey } from "./textKey";

export type SyncStatus =
  | "connecting"
  | "syncing"
  | "synced"
  | "dirty"
  | "committing"
  | "committed"
  | "flush_error"
  | "commit_error"
  | "error"
  | "offline";

export type TreeFile = { path: string; size: number };

export type ChatMessage = { from: string; text: string; at: number };

export type CollabListener = () => void;
export type ChatListener = (message: ChatMessage) => void;

const COLORS = [
  "#0ea5e9",
  "#8b5cf6",
  "#f59e0b",
  "#10b981",
  "#ef4444",
  "#ec4899",
  "#14b8a6",
  "#f97316",
];

function sessionStorageKey(projectId: string): string {
  return `superfolha.session.${projectId}`;
}

function pickColor(seed: string): string {
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0;
  return COLORS[h % COLORS.length]!;
}

export class ProjectCollab {
  readonly ydoc = new Y.Doc();
  readonly awareness: awarenessProtocol.Awareness;
  readonly projectId: string;

  status: SyncStatus = "connecting";
  /** True after the first CRDT sync from the server (docs have real content). */
  initialSynced = false;
  files: TreeFile[] = [];
  chat: ChatMessage[] = [];
  clientId = "";
  sessionId = "";
  errorMessage = "";
  sshPublic = "";

  private ws: WebSocket | null = null;
  private listeners = new Set<CollabListener>();
  private chatListeners = new Set<ChatListener>();
  private destroyed = false;
  private email: string;
  private wsPath: string;
  /** After hello.ack we must send SyncStep1 so the server returns SyncStep2 with full state. */
  private fencePassed = false;

  constructor(projectId: string, email: string, wsPath: string) {
    this.projectId = projectId;
    this.email = email || "user";
    this.wsPath = wsPath;
    this.awareness = new awarenessProtocol.Awareness(this.ydoc);
    const color = pickColor(this.email);
    this.awareness.setLocalStateField("user", {
      name: this.email.includes("@") ? this.email.split("@")[0] : this.email,
      email: this.email,
      color,
      colorLight: color + "33",
    });

    this.ydoc.on("update", this.onLocalDocUpdate);
    this.awareness.on("update", this.onLocalAwareness);
  }

  subscribe(fn: CollabListener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  /** Live `chat.message` only — not the initial `chat.history` snapshot. */
  subscribeChat(fn: ChatListener): () => void {
    this.chatListeners.add(fn);
    return () => this.chatListeners.delete(fn);
  }

  private notify() {
    for (const fn of this.listeners) fn();
  }

  private setStatus(s: SyncStatus) {
    this.status = s;
    this.notify();
  }

  /** Drop a failed mutation so the badge can recover on a valid file. */
  clearError() {
    if (this.status !== "error") return;
    this.errorMessage = "";
    this.setStatus(this.initialSynced ? "synced" : "syncing");
  }

  getYText(path: string): Y.Text {
    return this.ydoc.getText(textKey(path));
  }

  connect() {
    if (this.destroyed) return;
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${proto}//${window.location.host}${this.wsPath}`;
    this.setStatus("connecting");
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    this.ws = ws;

    ws.onopen = () => {
      // wait for hello
    };

    ws.onmessage = (ev) => {
      if (typeof ev.data === "string") {
        this.handleJSON(ev.data);
        return;
      }
      const u8 = new Uint8Array(ev.data as ArrayBuffer);
      this.handleSyncMessage(u8);
    };

    ws.onclose = () => {
      if (this.destroyed) return;
      this.setStatus("offline");
      // Reconnect with backoff
      setTimeout(() => {
        if (!this.destroyed) this.connect();
      }, 1500);
    };

    ws.onerror = () => {
      this.errorMessage = "WebSocket error";
      this.setStatus("error");
    };
  }

  destroy() {
    this.destroyed = true;
    this.ydoc.off("update", this.onLocalDocUpdate);
    this.awareness.off("update", this.onLocalAwareness);
    try {
      awarenessProtocol.removeAwarenessStates(
        this.awareness,
        [this.awareness.clientID],
        "local",
      );
    } catch {
      /* ignore */
    }
    this.ws?.close();
    this.ws = null;
    this.ydoc.destroy();
  }

  private sendBinary = (data: Uint8Array) => {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  };

  private sendJSON = (obj: unknown) => {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(obj));
    }
  };

  private onLocalDocUpdate = (update: Uint8Array, origin: unknown) => {
    if (
      origin === "remote" ||
      !this.ws ||
      this.ws.readyState !== WebSocket.OPEN
    ) {
      return;
    }
    const encoder = encoding.createEncoder();
    syncProtocol.writeUpdate(encoder, update);
    this.sendBinary(encoding.toUint8Array(encoder));
    // optimistic local dirty until server sync.status
    if (this.status === "synced" || this.status === "committed") {
      this.setStatus("dirty");
    }
  };

  private onLocalAwareness = ({
    added,
    updated,
    removed,
  }: {
    added: number[];
    updated: number[];
    removed: number[];
  }) => {
    const changed = added.concat(updated, removed);
    if (changed.length === 0) return;
    const update = awarenessProtocol.encodeAwarenessUpdate(
      this.awareness,
      changed,
    );
    let bin = "";
    for (let i = 0; i < update.length; i++)
      bin += String.fromCharCode(update[i]!);
    this.sendJSON({ type: "awareness", update: btoa(bin) });
  };

  /** Ask the server for everything we are missing (empty client → full doc). */
  private sendSyncStep1() {
    const encoder = encoding.createEncoder();
    syncProtocol.writeSyncStep1(encoder, this.ydoc);
    this.sendBinary(encoding.toUint8Array(encoder));
  }

  private handleSyncMessage(u8: Uint8Array) {
    const encoder = encoding.createEncoder();
    const decoder = decoding.createDecoder(u8);
    // origin "remote" so we do not echo applied server state as local updates
    syncProtocol.readSyncMessage(decoder, encoder, this.ydoc, "remote");
    if (encoding.length(encoder) > 1) {
      this.sendBinary(encoding.toUint8Array(encoder));
    }
    // First server payload after fence means we can mount editors with real text.
    if (this.fencePassed && !this.initialSynced) {
      this.initialSynced = true;
      this.setStatus("synced");
      this.notify();
    }
  }

  private handleJSON(raw: string) {
    let msg: Record<string, unknown>;
    try {
      msg = JSON.parse(raw) as Record<string, unknown>;
    } catch {
      return;
    }
    const type = msg.type as string;
    switch (type) {
      case "hello": {
        const sid = String(msg.session_id ?? "");
        const cid = String(msg.client_id ?? "");
        this.clientId = cid;
        this.sessionId = sid;
        const key = sessionStorageKey(this.projectId);
        const prev = sessionStorage.getItem(key);
        if (prev && prev !== sid) {
          sessionStorage.setItem(key, sid);
          window.location.reload();
          return;
        }
        sessionStorage.setItem(key, sid);
        this.sendJSON({ type: "hello.ack", session_id: sid });
        this.fencePassed = true;
        this.initialSynced = false;
        this.setStatus("syncing");
        // Critical: empty client must send SyncStep1; server replies SyncStep2
        // with the loaded project text. Server-only SyncStep1 cannot fill us.
        this.sendSyncStep1();
        // Re-announce awareness after fence.
        this.onLocalAwareness({
          added: [this.awareness.clientID],
          updated: [],
          removed: [],
        });
        break;
      }
      case "tree.snapshot": {
        const files = (msg.files as TreeFile[]) || [];
        this.files = files.map((f) => ({
          path: f.path,
          size: f.size ?? 0,
        }));
        this.notify();
        break;
      }
      case "tree.event": {
        const op = msg.op as string;
        const path = String(msg.path ?? "");
        if (op === "delete") {
          this.files = this.files.filter((f) => f.path !== path);
        } else if (op === "create" && path) {
          if (!this.files.some((f) => f.path === path)) {
            this.files = [...this.files, { path, size: 0 }];
          }
        }
        this.notify();
        break;
      }
      case "sync.status": {
        const st = String(msg.status ?? "");
        if (
          st === "synced" ||
          st === "dirty" ||
          st === "committing" ||
          st === "committed" ||
          st === "flush_error" ||
          st === "commit_error"
        ) {
          this.setStatus(st as SyncStatus);
        }
        break;
      }
      case "commit.done": {
        this.setStatus("committed");
        this.notify();
        break;
      }
      case "chat.history": {
        this.chat = (msg.messages as ChatMessage[]) || [];
        this.notify();
        break;
      }
      case "chat.message": {
        const incoming: ChatMessage = {
          from: String(msg.from ?? ""),
          text: String(msg.text ?? ""),
          at: Number(msg.at ?? Date.now()),
        };
        this.chat = [...this.chat, incoming];
        this.notify();
        for (const fn of this.chatListeners) fn(incoming);
        break;
      }
      case "awareness": {
        const b64 = String(msg.update ?? "");
        if (!b64) break;
        try {
          const bin = atob(b64);
          const u8 = new Uint8Array(bin.length);
          for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i);
          awarenessProtocol.applyAwarenessUpdate(this.awareness, u8, "remote");
          this.notify();
        } catch {
          /* ignore */
        }
        break;
      }
      case "push.error": {
        this.sshPublic = String(msg.ssh_public ?? "");
        this.errorMessage = String(msg.message ?? "push failed");
        this.setStatus("commit_error");
        break;
      }
      case "error": {
        this.errorMessage = String(msg.message ?? "error");
        this.setStatus("error");
        break;
      }
      case "pong":
        break;
      default:
        break;
    }
  }

  createFile(path: string, content = "") {
    this.sendJSON({ type: "file.create", path, content });
  }

  deleteFile(path: string) {
    this.sendJSON({ type: "file.delete", path });
  }

  commitNow(message?: string) {
    this.sendJSON({ type: "commit.now", message: message || "Manual commit" });
  }

  sendChat(text: string) {
    this.sendJSON({ type: "chat.send", text });
  }

  /** Peers from awareness (excluding self). */
  peers(): { name: string; color: string; clientId: number }[] {
    const out: { name: string; color: string; clientId: number }[] = [];
    this.awareness.getStates().forEach((state, clientId) => {
      if (clientId === this.awareness.clientID) return;
      const user = state.user as { name?: string; color?: string } | undefined;
      out.push({
        clientId,
        name: user?.name || `user-${clientId}`,
        color: user?.color || "#888",
      });
    });
    return out;
  }
}
