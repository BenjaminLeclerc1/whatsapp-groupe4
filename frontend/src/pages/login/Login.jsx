import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "../../components/styles/login.css";
import logo from "../../components/assets/logo.png";
import api from "../../api/axios";

function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  // --- REGEX PATTERNS ---
  // Standard email format
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  // Min 8 chars, 1 uppercase, 1 lowercase, 1 number
  const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/;

  const handleLogin = async (e) => {
    e.preventDefault();
    setError("");

    // ... (Your regex validation remains the same) ...

    try {
      const response = await api.post("/auth/login", {
        email: email,
        password: password,
      });

      console.log("Response from server:", response.data); // Log this to be 100% sure

      // 🛑 THE FIX:
      // Change from response.data.user.id (which is undefined)
      // To response.data.user_id (which matches your Postman output)
      // Save to localStorage
      localStorage.setItem("token", response.data.token);
      localStorage.setItem("user_id", response.data.user_id);
      console.log("Login successful!");
      // 🚀 Using window.location.href is safer than navigate("/")
      // when you need App.js to refresh and detect the new token
      window.location.href = "/chats";
    } catch (err) {
      console.error("Login Error:", err);
      // This will now only show if the server actually rejects the request (401)
      // or if the server is down (500/Network Error)
      setError(
        err.response?.data?.error ||
          "Connexion échouée. Vérifiez vos identifiants.",
      );
    }
  };

  return (
    <div className="container22">
      <div className="left">
        <h1 className="title">
          Rejoignez <span>nous</span>
        </h1>
        <div className="images-group">
          <img src={logo} className="logo" alt="logo" />
        </div>
      </div>

      <div className="right">
        <div className="login-card">
          <h2>Se connecter à Groupe 4</h2>

          {/* Error Message Display */}
          {error && (
            <p
              className="error-message"
              style={{
                color: "#ff4d4d",
                fontSize: "14px",
                marginBottom: "10px",
              }}
            >
              {error}
            </p>
          )}

          <input
            type="email"
            placeholder="Addresse email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={{
              borderColor: email && !emailRegex.test(email) ? "red" : "",
            }}
          />

          <input
            type="password"
            placeholder="Mot de passe"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{
              borderColor:
                password && !passwordRegex.test(password) ? "red" : "",
            }}
          />

          <button className="login-btn" onClick={handleLogin}>
            Se connecter
          </button>

          <p className="forgot">Mot de passe oublié ?</p>

          <button className="create-btn" onClick={() => navigate("/register")}>
            Créer un nouveau compte
          </button>
        </div>
      </div>
    </div>
  );
}

export default Login;
