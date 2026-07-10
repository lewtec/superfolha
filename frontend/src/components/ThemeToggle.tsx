import { Moon, Sun } from "feather-icons-react";
import { useTranslation } from "react-i18next";
import { useTheme } from "../hooks/useTheme";

export default function ThemeToggle() {
  const { t } = useTranslation("common");
  const { isDark, toggle } = useTheme();

  return (
    <button
      type="button"
      onClick={toggle}
      className="btn btn-ghost btn-square"
      aria-label={isDark ? t("theme_light") : t("theme_dark")}
      title={isDark ? t("theme_light") : t("theme_dark")}
    >
      {isDark ? <Sun size={20} /> : <Moon size={20} />}
    </button>
  );
}
