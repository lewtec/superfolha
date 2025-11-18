// src/components/Header.tsx
import React from "react";
import { Link } from "react-router-dom";

const Header: React.FC = () => {
  return (
    <header className="bg-base-100 shadow-md">
      <div className="container mx-auto px-4 py-4 flex justify-between items-center">
        <Link to="/" className="text-2xl font-bold">
          Superfolha
        </Link>
        <nav>
          <Link to="/login" className="btn btn-ghost">
            Entrar
          </Link>
          <Link to="/register" className="btn btn-primary ml-4">
            Cadastre-se
          </Link>
        </nav>
      </div>
    </header>
  );
};

export default Header;
