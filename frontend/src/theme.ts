/** daisyUI theme names for Superfolha slate/cobalt system */
export const THEME_LIGHT = "superfolha";
export const THEME_DARK = "superfolha-dark";

const STORAGE_KEY = "theme";

export type ThemePreference = "light" | "dark" | "system";

export function getStoredPreference(): ThemePreference {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (raw === "light" || raw === "dark" || raw === "system") {
    return raw;
  }
  // Legacy daisy "light"/"dark" or theme names
  if (raw === THEME_LIGHT || raw === "light") return "light";
  if (raw === THEME_DARK || raw === "dark") return "dark";
  return "system";
}

export function preferenceToTheme(pref: ThemePreference): string {
  if (pref === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? THEME_DARK
      : THEME_LIGHT;
  }
  return pref === "dark" ? THEME_DARK : THEME_LIGHT;
}

export function applyTheme(themeName: string) {
  document.documentElement.setAttribute("data-theme", themeName);
}

export function setPreference(pref: ThemePreference) {
  localStorage.setItem(STORAGE_KEY, pref);
  applyTheme(preferenceToTheme(pref));
}

/** Toggle between explicit light and dark (leaves system). */
export function toggleLightDark(currentResolvedDark: boolean): ThemePreference {
  const next: ThemePreference = currentResolvedDark ? "light" : "dark";
  setPreference(next);
  return next;
}

export function isDarkTheme(themeName: string): boolean {
  return themeName === THEME_DARK;
}

/** Apply theme before first paint when possible */
export function initThemeFromStorage() {
  const pref = getStoredPreference();
  applyTheme(preferenceToTheme(pref));
}
