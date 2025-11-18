// src/components/Footer.tsx
import React from "react";
import { useTranslation } from "react-i18next";

const Footer: React.FC = () => {
  const { t } = useTranslation();

  return (
    <footer className="bg-base-300 text-base-content">
      <div className="container mx-auto px-4 py-6 text-center">
        <p>{t("footer_text")}</p>
      </div>
    </footer>
  );
};

export default Footer;
