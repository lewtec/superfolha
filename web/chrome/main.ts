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
  const menu = $("sf-ident-menu") as (HTMLElement & { showPopover: () => void; hidePopover: () => void }) | null;
  const btn = $("sf-ident-btn");
  const label = $("sf-ident-label");
  const ini = $("sf-ident-initials");
  const leaveForm = $("sf-ident-leave") as HTMLFormElement | null;
  leaveForm?.addEventListener("submit", () => {
    clearActiveId();
  });

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
    const addL = root!.getAttribute("data-add") || "Add identity";
    const newL = root!.getAttribute("data-new") || "New identity";
    const leaveL = root!.getAttribute("data-leave") || "Leave";
    const logout = root!.getAttribute("data-logout") || "/logout";
    const items = all
      .map((r: IdentityRec) => {
        const on = r.id === active ? "menu-active" : "";
        return `<li><button type="button" class="${on}" data-switch="${r.id}">${r.id}</button></li>`;
      })
      .join("");
    menu.innerHTML =
      `<li class="menu-title">${root!.getAttribute("data-title") || "Identities"}</li>` +
      items +
      `<li><button type="button" data-add="1">${addL}</button></li>` +
      `<li><button type="button" data-mint="1">${newL}</button></li>` +
      `<li><form method="post" action="${logout}"><button type="submit">${leaveL}</button></form></li>`;
    const leave = menu.querySelector("form");
    leave?.addEventListener("submit", () => {
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
  root.addEventListener("click", (ev) => {
    const t = ev.target;
    if (!(t instanceof HTMLElement)) return;
    const sw = t.closest("[data-switch]") as HTMLElement | null;
    if (sw?.dataset.switch) {
      ev.preventDefault();
      void switchTo(sw.dataset.switch);
      return;
    }
    if (t.closest("[data-add]")) {
      ev.preventDefault();
      closeMenu();
      const dlg = $("sf-ident-load") as HTMLDialogElement | null;
      dlg?.showModal();
      return;
    }
    if (t.closest("[data-mint]")) {
      ev.preventDefault();
      void mintNew();
    }
  });

  $("sf-ident-load-pick")?.addEventListener("click", (ev) => {
    ev.preventDefault();
    fileEl?.click();
  });
  fileEl?.addEventListener("change", () => {
    const file = fileEl.files?.[0];
    fileEl.value = "";
    if (!file) return;
    void addFromFile(file);
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
