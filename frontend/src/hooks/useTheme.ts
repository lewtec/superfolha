import { useCallback, useEffect, useState } from "react";
import {
  getStoredPreference,
  preferenceToTheme,
  applyTheme,
  toggleLightDark,
  isDarkTheme,
  type ThemePreference,
} from "../theme";

export function useTheme() {
  const [preference, setPreferenceState] = useState<ThemePreference>(() =>
    typeof window === "undefined" ? "system" : getStoredPreference(),
  );
  const [themeName, setThemeName] = useState(() =>
    typeof window === "undefined"
      ? "superfolha"
      : preferenceToTheme(getStoredPreference()),
  );

  const refresh = useCallback(() => {
    const pref = getStoredPreference();
    const name = preferenceToTheme(pref);
    setPreferenceState(pref);
    setThemeName(name);
    applyTheme(name);
  }, []);

  useEffect(() => {
    // State is initialized from storage; only subscribe to system theme changes.
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      if (getStoredPreference() === "system") {
        refresh();
      }
    };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [refresh]);

  const toggle = useCallback(() => {
    const next = toggleLightDark(isDarkTheme(themeName));
    setPreferenceState(next);
    setThemeName(preferenceToTheme(next));
  }, [themeName]);

  return {
    preference,
    themeName,
    isDark: isDarkTheme(themeName),
    toggle,
  };
}
