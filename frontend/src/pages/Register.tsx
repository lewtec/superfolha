import { reportError } from "../utils/errorReporting";
import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useRegisterMutation } from "../hooks/useRegisterMutation";

export default function Register() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const { register, isInFlight } = useRegisterMutation({
    onCompleted: (response, errors) => {
      if (errors) {
        setError(errors[0].message);
        return;
      }
      if (response.register?.token) {
        localStorage.setItem("token", response.register.token);
        navigate("/projects");
      } else {
        setError(t("registration_failed"));
      }
    },
    onError: (err) => {
      setError(t("registration_error"));
      reportError(err);
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (password !== confirmPassword) {
      setError(t("passwords_do_not_match"));
      return;
    }

    if (password.length < 6) {
      setError(t("password_too_short"));
      return;
    }

    register(email, password);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-base-200">
      <div className="card w-96 bg-base-100 shadow-xl">
        <div className="card-body">
          <h2 className="card-title text-2xl font-bold text-center">
            Superfolha
          </h2>
          <p className="text-center text-base-content/70 mb-4">
            {t("register_title")}
          </p>

          {error && (
            <div className="alert alert-error">
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-control">
              <label className="label">
                <span className="label-text">{t("email_label")}</span>
              </label>
              <input
                type="email"
                placeholder="email@example.com"
                className="input input-bordered"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>

            <div className="form-control mt-4">
              <label className="label">
                <span className="label-text">{t("password_label")}</span>
              </label>
              <input
                type="password"
                placeholder="••••••••"
                className="input input-bordered"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            <div className="form-control mt-4">
              <label className="label">
                <span className="label-text">
                  {t("confirm_password_label")}
                </span>
              </label>
              <input
                type="password"
                placeholder="••••••••"
                className="input input-bordered"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </div>

            <div className="form-control mt-6">
              <button
                type="submit"
                className={`btn btn-primary ${isInFlight ? "loading" : ""}`}
                disabled={isInFlight}
              >
                {isInFlight ? t("creating_account") : t("register_button")}
              </button>
            </div>
          </form>

          <div className="divider">OR</div>

          <p className="text-center">
            {t("already_have_account")}{" "}
            <Link to="/login" className="link link-primary">
              {t("login_button")}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
