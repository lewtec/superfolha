import React, { useState, useEffect } from "react";
import { Sun, Moon } from "feather-icons-react"; // Import Feather Icons
import { useNavigate } from "react-router-dom"; // new import

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const [theme, setTheme] = useState(localStorage.getItem("theme") || "light");
  const navigate = useNavigate(); // new

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("theme", theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === "light" ? "dark" : "light"));
  };

  const logout = () => {
    localStorage.removeItem("token");
    navigate("/login");
  };

  return (
    <div className="flex min-h-screen flex-col">
      <div className="navbar flex justify-between bg-base-100 shadow-md">
        <div className="navbar-start">
          <a href="/" className="btn text-xl normal-case btn-ghost">
            superfolha
          </a>
        </div>
        <div className="navbar-end">
          {localStorage.getItem("token") && (
            <button className="btn btn-ghost" onClick={logout}>
              Logout
            </button>
          )}
          <button onClick={toggleTheme} className="btn btn-circle btn-ghost">
            {theme === "light" ? <Moon size={20} /> : <Sun size={20} />}
          </button>
        </div>
      </div>
      <main className="flex-1">{children}</main>
    </div>
  );
}
