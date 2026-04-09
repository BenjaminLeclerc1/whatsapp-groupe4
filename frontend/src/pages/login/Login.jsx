import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "../../components/styles/login.css";
import logo from "../../components/assets/logo.png";
import api from "../../api/axios";

function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/;

  const handleLogin = async (e) => {
    e.preventDefault();
    setError("");

    if (!emailRegex.test(email)) {
      setError("Veuillez entrer une adresse email valide.");
      return;
    }
    if (!passwordRegex.test(password)) {
      setError("Le mot de passe doit contenir au moins 8 caractères, une majuscule et un chiffre.");
      return;
    }

    setLoading(true);
    try {
      const response = await api.post("/auth/login", {
        email: email,
        password: password,
      });

      console.log("Response from server:", response.data); // Log this to be 100% sure

      localStorage.setItem("token", response.data.token);
      localStorage.setItem("user_id", response.data.user?.id || response.data.user_id || "");
      console.log("Login successful!");
      // 🚀 Using window.location.href is safer than navigate("/")
      // when you need App.js to refresh and detect the new token
      window.location.href = "/chats";
    } catch (err) {
      setError(err.response?.data?.error || "Connexion échouée. Vérifiez vos identifiants.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-container">
        <div className="auth-branding">
          <img src={logo} className="brand-logo" alt="Groupe 4" />
          <h1>Bienvenue sur <span>Groupe 4</span></h1>
          <p>Connectez-vous pour retrouver vos conversations et vos contacts.</p>
        </div>

        <div className="auth-form-panel">
          <h2>Connexion</h2>
          <p className="auth-subtitle">Entrez vos identifiants pour continuer</p>

          {error && (
            <div className="auth-error">
              ⚠️
              {error}
            </div>
          )}

          <form className="auth-form" onSubmit={handleLogin}>
            <div className="form-field">
              <label>Adresse email</label>
              <input
                type="email"
                placeholder="vous@exemple.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={email && !emailRegex.test(email) ? "input-error" : ""}
                autoComplete="email"
              />
            </div>

            <div className="form-field">
              <label>Mot de passe</label>
              <input
                type="password"
                placeholder="Votre mot de passe"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={password && !passwordRegex.test(password) ? "input-error" : ""}
                autoComplete="current-password"
              />
            </div>

            <p className="auth-forgot">Mot de passe oublié ?</p>

            <button className="auth-btn-primary" type="submit" disabled={loading}>
              {loading ? "Connexion..." : "Se connecter"}
            </button>

            <div className="auth-divider">ou</div>

            <button
              type="button"
              className="auth-btn-secondary"
              onClick={() => navigate("/register")}
            >
              Créer un nouveau compte
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Login;
