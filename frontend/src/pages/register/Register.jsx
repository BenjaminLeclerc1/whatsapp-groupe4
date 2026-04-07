import React, { useState, useMemo } from "react";
import axios from "axios";
import { API_BASE_URL } from "../../config";
import { useNavigate } from "react-router-dom";
import "../../components/styles/login.css";
import logo from "../../components/assets/logo.png";

function Register() {
  const [formData, setFormData] = useState({
    username: "",
    telephone: "",
    email: "",
    password: "",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/;

  const passwordStrength = useMemo(() => {
    const p = formData.password;
    if (!p) return 0;
    let score = 0;
    if (p.length >= 8) score++;
    if (/[A-Z]/.test(p) && /[a-z]/.test(p)) score++;
    if (/\d/.test(p)) score++;
    if (/[^a-zA-Z0-9]/.test(p)) score++;
    return score;
  }, [formData.password]);

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setError("");

    if (!formData.username.trim()) {
      setError("Le nom d'utilisateur est requis.");
      return;
    }
    if (!emailRegex.test(formData.email)) {
      setError("Veuillez entrer une adresse email valide.");
      return;
    }
    if (!passwordRegex.test(formData.password)) {
      setError("Le mot de passe doit contenir au moins 8 caractères, une majuscule et un chiffre.");
      return;
    }

    setLoading(true);
    try {
      const response = await axios.post(
        `${API_BASE_URL}/auth/register`,
        formData,
      );

      // auth-service renvoie { user: { id, ... }, token, refresh_token }
      const token = response.data.token;
      const userId = response.data.user?.id;
      if (token) localStorage.setItem("token", token);
      if (userId) localStorage.setItem("user_id", userId);
      localStorage.setItem("user", JSON.stringify(response.data.user || response.data));

      console.log("Registration successful!");

      window.location.href = token ? "/chats" : "/login";
      // navigate("/chat");
    } catch (err) {
      setError(err.response?.data?.error || "L'inscription a échoué. Veuillez réessayer.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-container">
        <div className="auth-branding">
          <img src={logo} className="brand-logo" alt="Groupe 4" />
          <h1>Rejoignez <span>Groupe 4</span></h1>
          <p>Créez votre compte en quelques secondes et commencez à discuter.</p>
        </div>

        <div className="auth-form-panel">
          <h2>Inscription</h2>
          <p className="auth-subtitle">Remplissez les informations ci-dessous</p>

          {error && (
            <div className="auth-error">
              ⚠️
              {error}
            </div>
          )}

          <form className="auth-form" onSubmit={handleRegister}>
            <div className="form-field">
              <label>Nom d'utilisateur</label>
              <input
                type="text"
                name="username"
                placeholder="Votre pseudo"
                value={formData.username}
                onChange={handleChange}
                autoComplete="username"
              />
            </div>

            <div className="form-field">
              <label>Adresse email</label>
              <input
                type="email"
                name="email"
                placeholder="vous@exemple.com"
                value={formData.email}
                onChange={handleChange}
                className={formData.email && !emailRegex.test(formData.email) ? "input-error" : ""}
                autoComplete="email"
              />
            </div>

            <div className="form-field">
              <label>Téléphone</label>
              <input
                type="tel"
                name="telephone"
                placeholder="+33 6 12 34 56 78"
                value={formData.telephone}
                onChange={handleChange}
                autoComplete="tel"
              />
            </div>

            <div className="form-field">
              <label>Mot de passe</label>
              <input
                type="password"
                name="password"
                placeholder="Min. 8 caractères, 1 majuscule, 1 chiffre"
                value={formData.password}
                onChange={handleChange}
                className={formData.password && !passwordRegex.test(formData.password) ? "input-error" : ""}
                autoComplete="new-password"
              />
              {formData.password && (
                <div className="password-strength">
                  {[1, 2, 3, 4].map((level) => (
                    <div
                      key={level}
                      className={`strength-bar ${
                        passwordStrength >= level
                          ? passwordStrength <= 1 ? "active" : passwordStrength <= 2 ? "medium" : "strong"
                          : ""
                      }`}
                    />
                  ))}
                </div>
              )}
            </div>

            <button className="auth-btn-primary" type="submit" disabled={loading}>
              {loading ? "Création du compte..." : "S'inscrire"}
            </button>

            <div className="auth-divider">ou</div>

            <button
              type="button"
              className="auth-btn-secondary"
              onClick={() => navigate("/login")}
            >
              Déjà un compte ? Se connecter
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Register;
