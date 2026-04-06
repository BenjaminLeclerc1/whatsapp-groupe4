import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useApp } from "../../context/AppContext";
import "../../components/styles/profile.css";

function Profile() {
  const { getUserById, updateUser, deleteUser } = useApp();
  const navigate = useNavigate();

  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState("");
  const [formData, setFormData] = useState({ username: "", telephone: "", email: "" });

  const userId = localStorage.getItem("user_id");

  useEffect(() => {
    const loadUser = async () => {
      if (!userId) return;
      const data = await getUserById(userId);
      if (data) {
        setUser(data);
        setFormData({
          username: data.username || "",
          telephone: data.telephone || "",
          email: data.email || "",
        });
      }
      setLoading(false);
    };
    loadUser();
  }, [userId, getUserById]);

  const handleSave = async () => {
    if (!formData.username.trim()) return;
    setSaving(true);
    const ok = await updateUser(userId, formData);
    if (ok) {
      localStorage.setItem("username", formData.username);
      setUser((prev) => ({ ...prev, ...formData }));
      setEditing(false);
      setSuccess("Profil mis à jour avec succès");
      setTimeout(() => setSuccess(""), 3000);
    }
    setSaving(false);
  };

  const handleDelete = async () => {
    const ok = await deleteUser(userId);
    if (ok) {
      localStorage.clear();
      window.location.href = "/login";
    }
  };

  if (loading) {
    return (
      <div className="profile-page">
        <div className="profile-loader">
          <div className="loader-spinner" />
        </div>
      </div>
    );
  }

  return (
    <div className="profile-page">
      <div className="profile-card">
        <div className="profile-cover">
          <div className="cover-gradient" />
        </div>

        <div className="profile-body">
          <div className="profile-avatar-section">
            <div className="profile-avatar-ring">
              <img
                src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${user?.username || "default"}`}
                alt="avatar"
                className="profile-avatar-img"
              />
            </div>
            {!editing && (
              <button className="edit-profile-btn" onClick={() => setEditing(true)}>
                <span className="material-icons-round">edit</span>
                Modifier le profil
              </button>
            )}
          </div>

          {success && (
            <div className="profile-success">
              <span className="material-icons-round">check_circle</span>
              {success}
            </div>
          )}

          <div className="profile-info">
            {editing ? (
              <div className="profile-edit-form">
                <div className="profile-field">
                  <label>Nom d'utilisateur</label>
                  <input
                    type="text"
                    value={formData.username}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    placeholder="Votre pseudo"
                  />
                </div>

                <div className="profile-field">
                  <label>Adresse e-mail</label>
                  <input
                    type="email"
                    value={formData.email}
                    onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                    placeholder="exemple@email.com"
                  />
                </div>

                <div className="profile-field">
                  <label>Téléphone</label>
                  <input
                    type="tel"
                    value={formData.telephone}
                    onChange={(e) => setFormData({ ...formData, telephone: e.target.value })}
                    placeholder="+33 6 12 34 56 78"
                  />
                </div>

                <div className="profile-edit-actions">
                  <button className="save-btn" onClick={handleSave} disabled={saving}>
                    {saving ? "Enregistrement..." : "Enregistrer"}
                  </button>
                  <button className="cancel-btn" onClick={() => setEditing(false)}>
                    Annuler
                  </button>
                </div>
              </div>
            ) : (
              <div className="profile-details">
                <div className="detail-row">
                  <span className="material-icons-round detail-icon">person</span>
                  <div>
                    <p className="detail-label">Nom d'utilisateur</p>
                    <p className="detail-value">{user?.username}</p>
                  </div>
                </div>

                <div className="detail-row">
                  <span className="material-icons-round detail-icon">email</span>
                  <div>
                    <p className="detail-label">Adresse e-mail</p>
                    <p className="detail-value">{user?.email || "Non renseigné"}</p>
                  </div>
                </div>

                <div className="detail-row">
                  <span className="material-icons-round detail-icon">phone</span>
                  <div>
                    <p className="detail-label">Téléphone</p>
                    <p className="detail-value">{user?.telephone || "Non renseigné"}</p>
                  </div>
                </div>

              </div>
            )}
          </div>

          <div className="profile-footer">
            <button className="back-btn" onClick={() => navigate("/chats")}>
              <span className="material-icons-round">arrow_back</span>
              Retour aux discussions
            </button>
            <button className="delete-account-btn" onClick={handleDelete}>
              <span className="material-icons-round">delete_forever</span>
              Supprimer le compte
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Profile;
