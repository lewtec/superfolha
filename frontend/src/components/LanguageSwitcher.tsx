import { useTranslation } from "react-i18next";
import { langCodes, supportedLngs, type SupportedLng } from "../i18n";

export default function LanguageSwitcher() {
  const { i18n, t } = useTranslation("common");
  const current = (i18n.resolvedLanguage || i18n.language || "en").slice(
    0,
    2,
  ) as SupportedLng;

  return (
    <label className="flex items-center gap-1">
      <span className="sr-only">{t("language")}</span>
      <select
        className="select select-bordered select-sm min-h-[var(--touch-min)] max-w-[4.5rem]"
        value={supportedLngs.includes(current) ? current : "en"}
        onChange={(e) => {
          void i18n.changeLanguage(e.target.value);
        }}
        aria-label={t("language")}
      >
        {supportedLngs.map((lng) => (
          <option key={lng} value={lng}>
            {langCodes[lng]}
          </option>
        ))}
      </select>
    </label>
  );
}
