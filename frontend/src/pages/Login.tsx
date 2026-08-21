import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { BrandMark } from "../components/Brand";
import { useLoginMutation } from "../hooks/useLoginMutation";
import { translateError, translateGraphQLErrors } from "../i18n/translateError";

export default function Login() {
  const { t } = useTranslation(["auth", "common", "errors"]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const { login, isInFlight } = useLoginMutation({
    onCompleted: (response, errors) => {
      if (errors) {
        setError(translateGraphQLErrors(t, errors));
        return;
      }
      // Session is the HttpOnly authToken cookie set by the server.
      if (response.login?.user) {
        navigate("/projects");
      } else {
        setError(t("auth:login_failed"));
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
    login(email.trim(), password);
  };

  return (
    <div className="page-fill min-h-dvh flex items-center justify-center bg-base-200">
      <div className="card w-96 bg-base-100 shadow-xl">
        <figure className="pt-8">
          <Link to="/" aria-label="Superfolha">
            <BrandMark className="h-16 w-16" />
          </Link>
        </figure>
        <div className="card-body">
          <h2 className="card-title text-2xl font-bold justify-center">
            Superfolha
          </h2>
          <p className="text-center text-base-content/70 mb-4">
            {t("auth:login_title")}
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

            <div className="form-control mt-6">
              <button
                type="submit"
                className={`btn btn-primary ${isInFlight ? "loading" : ""}`}
                disabled={isInFlight}
              >
                {isInFlight ? t("auth:signing_in") : t("auth:login_button")}
              </button>
            </div>
          </form>

          <div className="divider">{t("common:or")}</div>

          <p className="text-center">
            {t("auth:no_account")}{" "}
            <Link to="/register" className="link link-primary">
              {t("auth:register_button")}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
