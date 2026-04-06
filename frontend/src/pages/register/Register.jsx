import React, { useState } from "react";
import axios from "axios";
import { API_BASE_URL } from "../../config";
import { useNavigate } from "react-router-dom";
import "../../components/styles/login.css"; // Reuse your existing login CSS
import logo from "../../components/assets/logo.png";

function Register() {
  const [formData, setFormData] = useState({
    username: "",
    telephone: "",
    email: "",
    password: "",
  });
  const [error, setError] = useState("");
  const navigate = useNavigate();

  // --- REGEX PATTERNS (Matching your Login.js) ---
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/;

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setError("");

    // 1. Client-side Validation
    if (!formData.username.trim()) {
      setError("Le nom d'utilisateur est requis.");
      return;
    }
    if (!emailRegex.test(formData.email)) {
      setError("Veuillez entrer une adresse email valide.");
      return;
    }
    if (!passwordRegex.test(formData.password)) {
      setError(
        "Le mot de passe doit contenir au moins 8 caractères, une majuscule et un chiffre.",
      );
      return;
    }

    try {
      const response = await axios.post(
        `${API_BASE_URL}/auth/register`,
        formData,
      );

      // user-service renvoie l'utilisateur créé (sans JWT) : connexion via /login ensuite
      const id = response.data.id;
      if (id) {
        localStorage.setItem("user_id", id);
      }
      localStorage.setItem("user", JSON.stringify(response.data));

      console.log("Registration successful, redirecting to login...");

      window.location.href = "/login";
      // navigate("/chat");
    } catch (err) {
      setError(
        err.response?.data?.error ||
          "L'inscription a échoué. Veuillez réessayer.",
      );
    }
  };

  return (
    <div className="container22">
      <div className="left">
        <h1 className="title">
          Créez votre <span>compte</span>
        </h1>
        <div className="images-group">
          <img src={logo} className="logo" alt="logo" />
        </div>
      </div>

      <div className="right">
        <div className="login-card">
          <h2>S'inscrire à Groupe 4</h2>

          {/* Error Message */}
          {error && (
            <p
              style={{
                color: "#ff4d4d",
                fontSize: "14px",
                marginBottom: "10px",
                textAlign: "center",
              }}
            >
              {error}
            </p>
          )}

          <input
            type="text"
            name="username"
            placeholder="Nom d'utilisateur"
            value={formData.username}
            onChange={handleChange}
            required
          />

          <input
            type="email"
            name="email"
            placeholder="Addresse email"
            value={formData.email}
            onChange={handleChange}
            style={{
              borderColor:
                formData.email && !emailRegex.test(formData.email) ? "red" : "",
            }}
            required
          />

          <input
            type="text"
            name="telephone"
            placeholder="Téléphone"
            value={formData.telephone}
            onChange={handleChange}
            required
          />

          <input
            type="password"
            name="password"
            placeholder="Mot de passe (8+ caractères, 1 Maj, 1 Chiffre)"
            value={formData.password}
            onChange={handleChange}
            style={{
              borderColor:
                formData.password && !passwordRegex.test(formData.password)
                  ? "red"
                  : "",
            }}
            required
          />

          <button className="login-btn" onClick={handleRegister}>
            S'inscrire
          </button>

          <hr style={{ border: "0.5px solid #ddd", margin: "10px 0" }} />

          <button className="create-btn" onClick={() => navigate("/login")}>
            Vous avez déjà un compte ?
          </button>
        </div>
      </div>
    </div>
  );
}

export default Register;
