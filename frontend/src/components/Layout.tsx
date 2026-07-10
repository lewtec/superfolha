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

  const logout = async () => {
    try {
      await fetch("/api/logout", {
        method: "POST",
        credentials: "same-origin",
      });
    } catch (error) {
      console.error("Logout request failed:", error);
      // Still leave the app even if the network call fails.
    }
    navigate("/login");
  };

  return (
    <div className="min-h-screen flex flex-col">
      <div className="navbar bg-base-100 shadow-md flex justify-between">
        <div className="navbar-start">
          <a href="/" className="btn btn-ghost normal-case text-xl">
            superfolha
          </a>
        </div>
        <div className="navbar-end">
          <button className="btn btn-ghost" onClick={logout}>
            Logout
          </button>
          <button onClick={toggleTheme} className="btn btn-ghost btn-circle">
            {theme === "light" ? <Moon size={20} /> : <Sun size={20} />}
          </button>
        </div>
      </div>
      <main className="flex-1">{children}</main>
    </div>
  );
}
