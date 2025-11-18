// src/components/Header.tsx
import React from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

const Header: React.FC = () => {
  const { t } = useTranslation();

  return (
    <header className="bg-base-100 shadow-md">
      <div className="container mx-auto px-4 py-4 flex justify-between items-center">
        <Link to="/" className="text-2xl font-bold">
          Superfolha
        </Link>
        <nav>
          <Link to="/login" className="btn btn-ghost">
            {t("login_button")}
          </Link>
          <Link to="/register" className="btn btn-primary ml-4">
            {t("register_button")}
          </Link>
        </nav>
      </div>
    </header>
  );
};

export default Header;
