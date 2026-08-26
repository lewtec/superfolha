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
  const hex = id.replace(/^ed25519:/i, "");
  return (hex.slice(0, 2) || "?").toUpperCase();
}

function hexPart(id: string): string {
  return id.replace(/^ed25519:/i, "");
}

function icon(path: string, extra = ""): string {
  const cls = extra ? `size-4 shrink-0 ${extra}` : "size-4 shrink-0";
  return `<svg xmlns="http://www.w3.org/2000/svg" class="${cls}" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="${path}"/></svg>`;
}

const ICO = {
  plus: "M12 4.5v15m7.5-7.5h-15",
  userPlus:
    "M18 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0ZM3 19.235v-.11a6.375 6.375 0 0 1 12.75 0v.109A12.318 12.318 0 0 1 9.374 21c-2.331 0-4.512-.645-6.374-1.766Z",
  upload:
    "M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5",
  link: "M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244",
  sun: "M12 3v2.25m6.364.386-1.591 1.591M21 12h-2.25m-.386 6.364-1.591-1.591M12 18.75V21m-4.773-4.227-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z",
  moon: "M21.752 15.002A9.72 9.72 0 0 1 18 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 0 0 3 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 0 0 9.002-5.998Z",
  globe:
    "M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418",
  leave:
    "M8.25 9V5.25A2.25 2.25 0 0 1 10.5 3h6a2.25 2.25 0 0 1 2.25 2.25v13.5A2.25 2.25 0 0 1 16.5 21h-6a2.25 2.25 0 0 1-2.25-2.25V15M12 9l3 3m0 0-3 3m3-3H2.25",
  check: "M4.5 12.75l6 6 9-13.5",
};

const AVATAR_TONES = [
  "bg-neutral text-neutral-content",
  "bg-secondary text-secondary-content",
  "bg-accent text-accent-content",
  "bg-info text-info-content",
];

function avatarHTML(id: string, size: string): string {
  const hex = hexPart(id);
  let n = 0;
  for (let i = 0; i < hex.length; i++) n = (n + hex.charCodeAt(i)) % AVATAR_TONES.length;
  return `<div class="avatar avatar-placeholder"><div class="${size} rounded-full ${AVATAR_TONES[n]}"><span class="text-xs">${initials(id)}</span></div></div>`;
}

function actionItem(sf: string, path: string, label: string): string {
  return `<li><button type="button" data-sf="${sf}">${icon(path)}<span>${label}</span></button></li>`;
}

function themeDark(): boolean {
  return document.documentElement.getAttribute("data-theme") === "superfolha-dark";
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
    const themeLight = root!.getAttribute("data-theme-light") || "Light theme";
    const themeDarkL = root!.getAttribute("data-theme-dark") || "Dark theme";
    const signedIn = root!.getAttribute("data-signed-in") || "Signed in as";
    const langL = root!.getAttribute("data-lang-label") || "Language";
    const lang = root!.getAttribute("data-lang") || "en";
    const langNext = root!.getAttribute("data-lang-next") || "/";
    const langAction = root!.getAttribute("data-lang-action") || "/lang";
    const logout = root!.getAttribute("data-logout") || "/logout";
    const dark = themeDark();
    const items = all
      .map((r: IdentityRec) => {
        const on = r.id === active;
        return `<li><button type="button" class="${on ? "menu-active" : ""}" data-sf="switch" data-id="${r.id}" title="${r.id}">${avatarHTML(r.id, "w-8")}<span class="min-w-0 flex flex-col items-start gap-0"><span class="font-mono text-xs">${hexPart(r.id)}</span><span class="text-xs opacity-60">ed25519</span></span>${on ? icon(ICO.check) : ""}</button></li>`;
      })
      .join("");
    const langs: [string, string][] = [
      ["en", "English"],
      ["pt", "Português"],
      ["es", "Español"],
    ];
    const opts = langs
      .map(
        ([code, name]) =>
          `<option value="${code}"${lang === code ? " selected" : ""}>${name}</option>`,
      )
      .join("");
    const extras =
      (commitL ? actionItem("commit", ICO.upload, commitL) : "") +
      (inviteAction ? actionItem("invite", ICO.link, inviteL) : "") +
      `<li><button type="button" data-sf="theme"><span class="swap swap-rotate${dark ? " swap-active" : ""}">${icon(ICO.sun, "swap-on")}${icon(ICO.moon, "swap-off")}</span><span class="grow">${dark ? themeLight : themeDarkL}</span></button></li>` +
      `<li><form method="post" action="${langAction}"><input type="hidden" name="next" value="${langNext}"/>${icon(ICO.globe)}<span class="grow">${langL}</span><select name="lang" class="select select-sm w-28 shrink-0" aria-label="${langL}" onchange="this.form.submit()">${opts}</select></form></li>`;
    menu.innerHTML =
      `<li class="menu-title">${root!.getAttribute("data-title") || "Identities"}</li>` +
      (active
        ? `<li><div>${avatarHTML(active, "w-10")}<span class="min-w-0"><span class="block text-xs">${signedIn}</span><span class="block font-mono text-xs break-all">${active}</span></span></div></li>`
        : "") +
      items +
      actionItem("add", ICO.plus, addL) +
      actionItem("mint", ICO.userPlus, newL) +
      `<li aria-hidden="true"><hr class="border-base-300 mx-2 my-1"/></li>` +
      extras +
      `<li><form method="post" action="${logout}"><button type="submit" data-sf="leave">${icon(ICO.leave)}<span>${leaveL}</span></button></form></li>`;
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
        void paint();
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
