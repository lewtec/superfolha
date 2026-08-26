import * as ed from "@noble/ed25519";
import { parseIdentitySeed } from "../login/openssh";
import {
  addIdentity,
  clearActiveId,
  getActiveId,
  listIdentities,
  setActiveId,
  syncActiveFromLogin,
  verifyLogin,
  type IdentityRec,
} from "../ssh/identity";

function $(id: string): HTMLElement | null {
  return document.getElementById(id);
}

function initials(id: string): string {
  const hex = id.replace(/^ed25519:/, "");
  return (hex.slice(0, 2) || "?").toUpperCase();
}

function bind(): void {
  const root = $("sf-ident");
  if (!root || root.dataset.bound === "1") return;
  root.dataset.bound = "1";
  const login = root.getAttribute("data-login") ?? "";
  if (login) syncActiveFromLogin(login);
  const challengeURL = root.getAttribute("data-challenge") ?? "";
  const verifyURL = root.getAttribute("data-verify") ?? "";
  const fileEl = $("sf-ident-file") as HTMLInputElement | null;
  const fileNameEl = $("sf-ident-file-name");
  const confirmBtn = $("sf-ident-load-ok") as HTMLButtonElement | null;
  const menu = $("sf-ident-menu") as (HTMLElement & { showPopover: () => void; hidePopover: () => void }) | null;
  let pendingFile: File | null = null;
  const btn = $("sf-ident-btn");
  const label = $("sf-ident-label");
  const ini = $("sf-ident-initials");

  function menuOpen(): boolean {
    return !!menu && typeof menu.matches === "function" && menu.matches(":popover-open");
  }

  function setExpanded(on: boolean): void {
    btn?.setAttribute("aria-expanded", on ? "true" : "false");
  }

  function openMenu(): void {
    if (!menu || typeof menu.showPopover !== "function") return;
    if (!menuOpen()) menu.showPopover();
    setExpanded(true);
  }

  function closeMenu(): void {
    if (!menu || typeof menu.hidePopover !== "function") return;
    if (menuOpen()) menu.hidePopover();
    setExpanded(false);
  }

  function toggleMenu(): void {
    if (menuOpen()) closeMenu();
    else openMenu();
  }

  async function paint(): Promise<void> {
    const all = await listIdentities();
    const active = getActiveId() || login;
    if (label) label.textContent = active || "";
    if (ini) ini.textContent = initials(active || "?");
    if (!menu) return;
    const addL = root!.getAttribute("data-add-label") || "Add identity";
    const newL = root!.getAttribute("data-new-label") || "New identity";
    const leaveL = root!.getAttribute("data-leave-label") || "Leave";
    const inviteL = root!.getAttribute("data-invite-label") || "Invite";
    const inviteAction = root!.getAttribute("data-invite-action") || "";
    const commitL = root!.getAttribute("data-commit-label") || "";
    const themeL = root!.getAttribute("data-theme-label") || "Theme";
    const langL = root!.getAttribute("data-lang-label") || "Language";
    const lang = root!.getAttribute("data-lang") || "en";
    const langNext = root!.getAttribute("data-lang-next") || "/";
    const langAction = root!.getAttribute("data-lang-action") || "/lang";
    const logout = root!.getAttribute("data-logout") || "/logout";
    const items = all
      .map((r: IdentityRec) => {
        const on = r.id === active ? "menu-active" : "";
        return `<li><button type="button" class="${on} font-mono text-xs break-all" data-sf="switch" data-id="${r.id}">${r.id}</button></li>`;
      })
      .join("");
    const opt = (code: string) =>
      `<option value="${code}"${lang === code ? " selected" : ""}>${code.toUpperCase()}</option>`;
    const extras =
      (commitL ? `<li><button type="button" data-sf="commit">${commitL}</button></li>` : "") +
      (inviteAction ? `<li><button type="button" data-sf="invite">${inviteL}</button></li>` : "") +
      `<li><button type="button" data-sf="theme">${themeL}</button></li>` +
      `<li>
        <label class="px-3 py-1 text-xs opacity-60">${langL}</label>
        <form method="post" action="${langAction}">
          <input type="hidden" name="next" value="${langNext}"/>
          <select name="lang" class="select select-sm w-full" onchange="this.form.submit()">${opt("en")}${opt("pt")}${opt("es")}</select>
        </form>
      </li>`;
    menu.innerHTML =
      `<li class="menu-title">${root!.getAttribute("data-title") || "Identities"}</li>` +
      `<li class="menu-title"><span class="font-mono text-xs break-all normal-case">${active || ""}</span></li>` +
      items +
      `<li><button type="button" data-sf="add">${addL}</button></li>` +
      `<li><button type="button" data-sf="mint">${newL}</button></li>` +
      extras +
      `<li><form method="post" action="${logout}"><button type="submit" data-sf="leave">${leaveL}</button></form></li>`;
    menu.querySelector("[data-sf=leave]")?.closest("form")?.addEventListener("submit", () => {
      clearActiveId();
    });
  }

  btn?.addEventListener("click", (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    toggleMenu();
  });
  document.addEventListener("click", (ev) => {
    if (!menuOpen()) return;
    const t = ev.target;
    if (t instanceof Node && (menu?.contains(t) || btn?.contains(t))) return;
    closeMenu();
  });
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "Escape") closeMenu();
  });
  menu?.addEventListener("click", (ev) => {
    const t = ev.target;
    if (!(t instanceof HTMLElement)) return;
    const act = t.closest("[data-sf]") as HTMLElement | null;
    if (!act || !menu.contains(act)) return;
    switch (act.dataset.sf) {
      case "switch":
        ev.preventDefault();
        if (act.dataset.id) void switchTo(act.dataset.id);
        return;
      case "add":
        ev.preventDefault();
        closeMenu();
        pendingFile = null;
        if (fileEl) fileEl.value = "";
        if (fileNameEl) fileNameEl.textContent = root.getAttribute("data-no-file") || "No file chosen";
        confirmBtn?.setAttribute("disabled", "true");
        loadStatus("", false);
        ($("sf-ident-load") as HTMLDialogElement | null)?.showModal();
        return;
      case "mint":
        ev.preventDefault();
        void mintNew();
        return;
      case "commit":
        ev.preventDefault();
        closeMenu();
        $("editor-commit")?.click();
        return;
      case "invite":
        ev.preventDefault();
        closeMenu();
        void copyInvite();
        return;
      case "theme":
        ev.preventDefault();
        (window as unknown as { __sfTheme?: { toggle: () => void } }).__sfTheme?.toggle();
        return;
      default:
        return;
    }
  });

  fileEl?.addEventListener("change", () => {
    const file = fileEl.files?.[0] ?? null;
    pendingFile = file;
    if (fileNameEl) {
      fileNameEl.textContent = file ? file.name : root.getAttribute("data-no-file") || "No file chosen";
    }
    if (file) confirmBtn?.removeAttribute("disabled");
    else confirmBtn?.setAttribute("disabled", "true");
    loadStatus("", false);
  });
  confirmBtn?.addEventListener("click", (ev) => {
    ev.preventDefault();
    if (!pendingFile) return;
    void addFromFile(pendingFile);
  });

  async function switchTo(id: string): Promise<void> {
    const all = await listIdentities();
    const rec = all.find((r) => r.id === id);
    if (!rec) return;
    setActiveId(id);
    const next = await verifyLogin(rec.seed, challengeURL, verifyURL, location.pathname);
    location.assign(next);
  }

  function loadStatus(text: string, isError: boolean): void {
    const el = $("sf-ident-load-status");
    if (!el) return;
    el.textContent = text;
    el.classList.toggle("text-error", isError);
  }

  async function addFromFile(file: File): Promise<void> {
    const pass = (document.getElementById("sf-ident-pass") as HTMLInputElement | null)?.value ?? "";
    loadStatus(root.getAttribute("data-working") || "Signing in…", false);
    try {
      const seed = parseIdentitySeed(await file.text(), pass);
      await addIdentity(seed);
      const next = await verifyLogin(seed, challengeURL, verifyURL, location.pathname);
      location.assign(next);
    } catch (err) {
      console.error("superfolha identity load", err);
      loadStatus(root.getAttribute("data-bad-key") || "That file is not an Ed25519 OpenSSH key.", true);
    }
  }

  function toast(text: string): void {
    const wrap = $("sf-ident-toast");
    const msg = $("sf-ident-toast-msg");
    if (!wrap || !msg) return;
    msg.textContent = text;
    wrap.classList.remove("hidden");
    window.setTimeout(() => wrap.classList.add("hidden"), 4000);
  }

  function copyText(text: string): void {
    if (window.isSecureContext && navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(text);
      return;
    }
    const el = document.createElement("textarea");
    el.value = text;
    el.setAttribute("readonly", "");
    el.style.position = "fixed";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    el.select();
    document.execCommand("copy");
    el.remove();
  }

  async function copyInvite(): Promise<void> {
    const action = root.getAttribute("data-invite-action") || "";
    if (!action) return;
    try {
      const res = await fetch(action, {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!res.ok) throw new Error("invite " + res.status);
      const out = (await res.json()) as { url?: string };
      if (!out.url) throw new Error("empty invite");
      copyText(new URL(out.url, location.origin).href);
      toast(root.getAttribute("data-invite-copied") || "Invite link copied.");
    } catch (err) {
      console.error("superfolha invite", err);
    }
  }

  async function mintNew(): Promise<void> {
    const { secretKey } = ed.keygen();
    await addIdentity(secretKey);
    const next = await verifyLogin(secretKey, challengeURL, verifyURL, location.pathname);
    location.assign(next);
  }

  void paint().catch((err) => console.error("superfolha identities", err));
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", bind);
} else {
  bind();
}
