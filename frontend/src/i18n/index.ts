import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enEditor from "./locales/en/editor.json";
import enProjects from "./locales/en/projects.json";
import enLanding from "./locales/en/landing.json";
import enErrors from "./locales/en/errors.json";

import ptCommon from "./locales/pt/common.json";
import ptAuth from "./locales/pt/auth.json";
import ptEditor from "./locales/pt/editor.json";
import ptProjects from "./locales/pt/projects.json";
import ptLanding from "./locales/pt/landing.json";
import ptErrors from "./locales/pt/errors.json";

import esCommon from "./locales/es/common.json";
import esAuth from "./locales/es/auth.json";
import esEditor from "./locales/es/editor.json";
import esProjects from "./locales/es/projects.json";
import esLanding from "./locales/es/landing.json";
import esErrors from "./locales/es/errors.json";

export const supportedLngs = ["en", "pt", "es"] as const;
export type SupportedLng = (typeof supportedLngs)[number];

const resources = {
  en: {
    common: enCommon,
    auth: enAuth,
    editor: enEditor,
    projects: enProjects,
    landing: enLanding,
    errors: enErrors,
  },
  pt: {
    common: ptCommon,
    auth: ptAuth,
    editor: ptEditor,
    projects: ptProjects,
    landing: ptLanding,
    errors: ptErrors,
  },
  es: {
    common: esCommon,
    auth: esAuth,
    editor: esEditor,
    projects: esProjects,
    landing: esLanding,
    errors: esErrors,
  },
};

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs: [...supportedLngs],
    defaultNS: "common",
    ns: ["common", "auth", "editor", "projects", "landing", "errors"],
    interpolation: { escapeValue: false },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "lng",
    },
    debug: false,
  });

export default i18n;
