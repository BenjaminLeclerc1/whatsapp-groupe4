import React, { useState, useEffect, useRef, useCallback } from "react";
import { useApp } from "../../context/AppContext";
import "../../components/styles/contacts.css";
import { useNavigate } from "react-router-dom";

const Contacts = () => {
  const { users, loading, createChat, getUserById, searchUsers } = useApp();
  const [searchTerm, setSearchTerm] = useState("");
  const debounceRef = useRef(null);

  const handleSearchChange = useCallback((value) => {
    setSearchTerm(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      searchUsers(value);
    }, 300);
  }, [searchUsers]);

  useEffect(() => {
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, []);
  
  const [selectedUser, setSelectedUser] = useState(null);
  const navigate = useNavigate();

  // const handleStartChat = async (userId) => {
  //   try {
  //     await createChat({ participants: userId, type: "private" });
  //   } catch (err) {
  //     alert("Erreur lors de la création du chat");
  //   }
  // };

  // ... inside your Contacts component ...

const handleStartChat = async (userId) => {
    try {
      const newChat = await createChat({ participants: [userId], type: "private" });
      
      if (newChat) {
        setSelectedUser(null);
        // 3. Redirect to your chat route
        // Replace "/chat" or "/" with your actual chat page path
        navigate("/chats"); 
      }
    } catch (err) {
      console.error(err);
      alert("Erreur lors de l'ouverture de la discussion");
    }
  };

  // Nouvelle fonction pour voir les détails
  const handleSeeMore = async (userId) => {
    const data = await getUserById(userId);
    if (data) {
      setSelectedUser(data);
    }
  };

  return (
    <div className="whatsapp-app">
      <div className="contacts-container">
        <aside className="contacts-sidebar">
          <header className="sidebar-header">
            <div className="header-title-row">
              <span className="material-icons">group</span>
              <h2>Tous les utilisateurs</h2>
            </div>
          </header>

          <div className="search-container">
            <div className="search-input-wrapper">
              <span className="material-icons">search</span>
              <input 
                type="text" 
                placeholder="Rechercher par nom ou email..." 
                value={searchTerm}
                onChange={(e) => handleSearchChange(e.target.value)}
              />
            </div>
          </div>

          <div className="chat-list">
            {loading ? (
              <div className="loader-container"><div className="loader"></div></div>
            ) : users.map((user) => (
              <div key={user.id} className="chat-card contact-item">
                <div className="card-avatar" onClick={() => handleStartChat(user.id)}>
                  <img src={`https://ui-avatars.com/api/?name=${user.username}&background=00a884&color=fff`} alt="photo de profil" />
                </div>
                
                <div className="card-info" onClick={() => handleStartChat(user.id)}>
                  <div className="card-row"><span className="chat-name">{user.username}</span></div>
                  <div className="card-row"><span className="user-email">{user.email}</span></div>
                </div>

                {/* --- REPLACED more_vert WITH ... --- */}
                <button className="see-more-btn" onClick={() => handleSeeMore(user.id)}>
                  Voir plus
                </button>
              </div>
            ))}
          </div>
        </aside>

        {/* Section Détails (S'affiche à droite quand on clique sur Voir plus) */}
        {selectedUser && (
          <div className="user-details-panel">
            <header className="details-header">
              <button onClick={() => setSelectedUser(null)} className="close-btn">✕</button>
              <h3>Détails du profil</h3>
            </header>
            
            <div className="details-content">
              <div className="details-avatar">
                 <img src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${selectedUser.username}`} alt="avatar" />
              </div>
              <div className="details-info-group">
                <label>NOM D'UTILISATEUR</label>
                <p>{selectedUser.username}</p>
              </div>
              <div className="details-info-group">
                <label>IDENTIFIANT UNIQUE</label>
                <p className="uuid-text">{selectedUser.id}</p>
              </div>
              <button className="action-btn" onClick={() => handleStartChat(selectedUser.id)}>
                Envoyer un message
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Contacts;