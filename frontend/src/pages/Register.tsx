import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useRegisterMutation } from "../hooks/useRegisterMutation";
import { translateError, translateGraphQLErrors } from "../i18n/translateError";

// Must match backend MinPasswordLength in internal/auth/auth.go.
const MIN_PASSWORD_LENGTH = 8;

export default function Register() {
  const { t } = useTranslation(["auth", "common", "errors"]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const { register, isInFlight } = useRegisterMutation({
    onCompleted: (response, errors) => {
      if (errors) {
        setError(translateGraphQLErrors(t, errors));
        return;
      }
      // Session is the HttpOnly authToken cookie set by the server.
      if (response.register?.user) {
        navigate("/projects");
      } else {
        setError(t("auth:registration_failed"));
      }
    },
    onError: (err) => {
      setError(translateError(t, err));
      console.error(err);
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (password !== confirmPassword) {
      setError(t("auth:passwords_do_not_match"));
      return;
    }

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(t("auth:password_too_short"));
      return;
    }

    register(email.trim(), password);
  };

  return (
    <div className="page-fill min-h-dvh flex items-center justify-center bg-base-200">
      <div className="card w-96 bg-base-100 shadow-xl">
        <div className="card-body">
          <h2 className="card-title text-2xl font-bold text-center">
            Superfolha
          </h2>
          <p className="text-center text-base-content/70 mb-4">
            {t("auth:register_title")}
          </p>

          {error && (
            <div className="alert alert-error">
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-control">
              <label className="label">
                <span className="label-text">{t("auth:email_label")}</span>
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
                <span className="label-text">{t("auth:password_label")}</span>
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
                  {t("auth:confirm_password_label")}
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
                {isInFlight
                  ? t("auth:creating_account")
                  : t("auth:register_button")}
              </button>
            </div>
          </form>

          <div className="divider">{t("common:or")}</div>

          <p className="text-center">
            {t("auth:already_have_account")}{" "}
            <Link to="/login" className="link link-primary">
              {t("auth:login_button")}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
