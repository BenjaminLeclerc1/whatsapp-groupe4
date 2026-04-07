import React, { useState, useEffect, useRef, useCallback } from "react";
import { useApp } from "../../context/AppContext";
import "../../components/styles/contacts.css";
import { useNavigate } from "react-router-dom";

const Contacts = () => {
  const { users, createChat, searchUsers, chats, currentUserId, getAuthUser } = useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [resolvedNames, setResolvedNames] = useState({});
  const debounceRef = useRef(null);
  const navigate = useNavigate();

  const handleSearchChange = useCallback((value) => {
    setSearchTerm(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      searchUsers(value);
    }, 300);
  }, [searchUsers]);

  const openModal = () => {
    setModalOpen(true);
    setSearchTerm("");
    searchUsers("");
  };

  const closeModal = () => {
    setModalOpen(false);
    setSearchTerm("");
    searchUsers("");
  };

  const handleStartChat = async (userId) => {
    try {
      const newChat = await createChat({ participants: [userId], type: "private" });
      if (newChat) {
        closeModal();
        navigate("/chats");
      }
    } catch (err) {
      console.error(err);
      alert("Erreur lors de l'ouverture de la discussion");
    }
  };

  const recentContacts = chats
    .filter((c) => c.type === "private")
    .map((c) => {
      const other = c.participants?.find((p) => p !== currentUserId);
      const resolved = resolvedNames[other];
      return {
        chatId: c.id,
        participantId: other,
        name: resolved?.username || c.name || "...",
        telephone: resolved?.telephone || "",
      };
    })
    .filter((c) => c.participantId);

  useEffect(() => {
    const idsToResolve = chats
      .filter((c) => c.type === "private")
      .map((c) => c.participants?.find((p) => p !== currentUserId))
      .filter((id) => id && !resolvedNames[id]);

    if (idsToResolve.length === 0) return;

    idsToResolve.forEach(async (id) => {
      const data = await getAuthUser(id);
      if (data) {
        setResolvedNames((prev) => ({ ...prev, [id]: data }));
      }
    });
  }, [chats, currentUserId, getAuthUser, resolvedNames]);

  return (
    <div className="whatsapp-app">
      <div className="contacts-container">
        <aside className="contacts-sidebar">
          <header className="sidebar-header">
            <div className="header-title-row">
              <h2>Contacts</h2>
            </div>
          </header>

          <div className="cta-container">
            <button className="add-contact-cta" onClick={openModal}>
              Ajouter un contact
            </button>
          </div>

          <div className="chat-list">
            {recentContacts.length === 0 ? (
              <div className="empty-contacts">
                <p>Aucun contact pour le moment</p>
                <p className="empty-hint">Ajoutez un contact pour commencer une discussion</p>
              </div>
            ) : (
              recentContacts.map((contact) => (
                <div
                  key={contact.chatId}
                  className="chat-card contact-item"
                  onClick={() => { navigate("/chats"); }}
                >
                  <div className="card-avatar">
                    <img
                      src={`https://ui-avatars.com/api/?name=${encodeURIComponent(contact.name)}&background=00a884&color=fff`}
                      alt="avatar"
                    />
                  </div>
                  <div className="card-info">
                    <div className="card-row">
                      <span className="chat-name">{contact.name}</span>
                    </div>
                    {contact.telephone && (
                      <div className="card-row">
                        <span className="user-email">{contact.telephone}</span>
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </aside>
      </div>

      {modalOpen && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <header className="modal-header">
              <h3>Ajouter un contact</h3>
              <button className="modal-close" onClick={closeModal}>✕</button>
            </header>

            <div className="modal-search">
              <input
                type="text"
                placeholder="Rechercher par nom, email ou téléphone..."
                value={searchTerm}
                onChange={(e) => handleSearchChange(e.target.value)}
                autoFocus
              />
            </div>

            <div className="modal-results">
              {!searchTerm.trim() ? (
                <div className="modal-empty">
                  <p>Tapez un nom, email ou numéro pour rechercher</p>
                </div>
              ) : users.length === 0 ? (
                <div className="modal-empty">
                  <p>Aucun résultat pour « {searchTerm} »</p>
                </div>
              ) : (
                users
                  .filter((u) => u.id !== currentUserId)
                  .map((user) => (
                    <div key={user.id} className="modal-user-row">
                      <div className="card-avatar">
                        <img
                          src={`https://ui-avatars.com/api/?name=${encodeURIComponent(user.username)}&background=00a884&color=fff`}
                          alt="avatar"
                        />
                      </div>
                      <div className="modal-user-info">
                        <span className="chat-name">{user.username}</span>
                        <span className="user-email">{user.email}</span>
                      </div>
                      <button
                        className="modal-add-btn"
                        onClick={() => handleStartChat(user.id)}
                      >
                        Discuter
                      </button>
                    </div>
                  ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Contacts;
