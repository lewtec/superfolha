// src/components/Header.tsx
import React from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

const Header: React.FC = () => {
  const { t } = useTranslation();

  return (
    <header className="bg-base-100 shadow-md">
      <div className="container mx-auto flex items-center justify-between p-4">
        <Link to="/" className="text-2xl font-bold">
          Superfolha
        </Link>
        <nav>
          <Link to="/login" className="btn btn-ghost">
            {t("login_button")}
          </Link>
          <Link to="/register" className="btn ml-4 btn-primary">
            {t("register_button")}
          </Link>
        </nav>
      </div>
    </header>
  );
};

export default Header;
