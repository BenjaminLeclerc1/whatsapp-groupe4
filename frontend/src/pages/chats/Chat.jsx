

import React, { useState, useRef, useEffect, useCallback } from "react";
import "../../components/styles/message.css";
import { useApp } from "../../context/AppContext";

const Chats = () => {
  const {
    chats,
    selectedChat,
    setSelectedChat,
    messages,
    loading,
    currentUserId,
    sendMessage,
    createChat,
    updateChat,
    deleteChat,
    deleteMessage,
    getHistory,
    notifications,
    markNotificationsAsRead,
    users,
    searchUsers,
  } = useApp();

  const [newMessage, setNewMessage] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [newChatData, setNewChatData] = useState({
    participants: [],
    type: "private",
    name: "",
  });
  const [contactSearch, setContactSearch] = useState("");
  const contactDebounceRef = useRef(null);

  const handleContactSearch = useCallback((value) => {
    setContactSearch(value);
    if (contactDebounceRef.current) clearTimeout(contactDebounceRef.current);
    contactDebounceRef.current = setTimeout(() => {
      searchUsers(value);
    }, 300);
  }, [searchUsers]);

  const [openMenuId, setOpenMenuId] = useState(null);
  const [openChatActionId, setOpenChatActionId] = useState(null);
  const [showHeaderMenu, setShowHeaderMenu] = useState(false);

  const messagesEndRef = useRef(null);

  // Badge de notification
  const NotificationBadge = ({ chatId }) => {
    const count = notifications.filter((n) => n.chat_id === chatId).length;
    if (count === 0) return null;
    return <div className="notification-badge">{count}</div>;
  };

  const handleLoadHistory = async () => {
    if (selectedChat) {
      await getHistory(selectedChat.id);
      setShowHeaderMenu(false);
    }
  };

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    const handleClickOutside = () => {
      setOpenMenuId(null);
      setOpenChatActionId(null);
      setShowHeaderMenu(false);
    };
    window.addEventListener("click", handleClickOutside);
    return () => window.removeEventListener("click", handleClickOutside);
  }, []);

  const handleSend = async (e) => {
    e.preventDefault();
    if (!newMessage.trim()) return;
    const success = await sendMessage(newMessage);
    if (success) setNewMessage("");
  };

  const toggleParticipant = (userId) => {
    setNewChatData((prev) => {
      const already = prev.participants.includes(userId);
      return {
        ...prev,
        participants: already
          ? prev.participants.filter((id) => id !== userId)
          : [...prev.participants, userId],
      };
    });
  };

  const availableContacts = users.filter((u) => u.id !== currentUserId);

  const handleStartChat = async (e) => {
    e.preventDefault();
    if (newChatData.participants.length === 0) return;
    setIsCreating(true);
    try {
      await createChat({
        participants: newChatData.participants,
        type: newChatData.type === "groupe" ? "group" : "private",
        name: newChatData.name,
      });
      setShowModal(false);
      setNewChatData({ participants: [], type: "private", name: "" });
      setContactSearch("");
      searchUsers("");
    } catch (err) {
      alert("Erreur: " + err.message);
    } finally {
      setIsCreating(false);
    }
  };

const onRename = (e, chat) => {
  if (e) e.stopPropagation(); // Empêche le clic de fermer d'autres éléments
  const newName = prompt("Nouveau nom du groupe :", chat.name);
  if (newName && newName.trim() !== "") {
    updateChat(chat.id, { name: newName });
    setShowHeaderMenu(false); // Ferme le menu de l'en-tête après renommage
  }
};

  const onDeleteChat = (e, chat) => {
    e.stopPropagation();
    if (window.confirm(`Supprimer la discussion "${chat.name || "Privée"}" ?`)) {
      deleteChat(chat.id);
    }
  };

  return (
    <div className="whatsapp-app">
      <div className="whatsapp-layout">
        {/* SIDEBAR */}
        <aside className="sidebar">
          <header className="sidebar-header">
            <div className="user-avatar">
              <img src={`https://ui-avatars.com/api/?name=User&background=075E54&color=fff`} alt="moi" />
            </div>
            <div className="header-actions">
              <button className="icon-btn" onClick={() => setShowModal(true)} style={{backgroundColor: 'green', padding: '5px 9px', color:'white'}}>
                + Nouveau groupe
              </button>
            </div>
          </header>

          <div className="search-container">
            <div className="search-input-wrapper">
              <input type="text" placeholder="Rechercher une discussion" />
            </div>
          </div>

          <div className="chat-list">
            {loading && chats.length === 0 ? (
              <div className="loader-container"><div className="loader"></div></div>
            ) : (
              [...chats]
                .sort((a, b) => new Date(b.updated_at || b.created_at) - new Date(a.updated_at || a.created_at))
                .map((chat) => (
                  <div
                    key={chat.id}
                    className={`chat-card ${selectedChat?.id === chat.id ? "active" : ""}`}
                    onClick={() => {
                      setSelectedChat(chat);
                      markNotificationsAsRead(chat.id);
                    }}
                  >
                    <div className="card-avatar">
                      {chat.type === "group" ? "👥" : "👤"}
                    </div>
                    <div className="card-info">
                      <div className="card-row">
                        <span className="chat-name">{chat.name || "Discussion"}</span>
                        <NotificationBadge chatId={chat.id} />
                        <button
                          className="msg-options-trigger"
                          style={{border: 'none', backgroundColor:'white'}}
                          onClick={(e) => {
                            e.stopPropagation();
                            setOpenChatActionId(openChatActionId === chat.id ? null : chat.id);
                          }}
                        >
                          ⋯
                        </button>
                      </div>
                      {openChatActionId === chat.id && (
                        <div className="msg-popup-menu" style={{ display: "block", right: "10px", top: "30px" }}>
                          {chat.type === "group" && (
                            <button className="menu-item" onClick={(e) => onRename(e, chat)}>Renommer</button>
                          )}
                          <button className="menu-item delete" onClick={(e) => onDeleteChat(e, chat)}>Supprimer</button>
                        </div>
                      )}
                      <div className="card-row">
                        <span className="last-msg">{chat.participants?.length} membres</span>
                      </div>
                    </div>
                  </div>
                ))
            )}
          </div>
        </aside>

        {/* MAIN CHAT */}
        <main className="chat-main">
          {selectedChat ? (
            <div className="active-chat-window">
              <header className="chat-header">
                <div className="header-avatar" style={{fontSize: '24px'}}>{selectedChat.type === "group" ? "👥" : "👤"}</div>
                <div className="header-contact-info">
                  <h3>{selectedChat.name || "Discussion"}</h3>
                  <p>{selectedChat.participants?.length} participants</p>
                </div>
               {/* MAIN CHAT HEADER ACTIONS */}
<div className="header-actions">
  <button className="icon-btn" onClick={(e) => { e.stopPropagation(); setShowHeaderMenu(!showHeaderMenu); }}>
    <span style={{marginLeft: '20px', padding: '0px 10px 7px 10px', cursor: 'pointer'}}>⋯</span>
  </button>
  
  {showHeaderMenu && (
    <div className="msg-popup-menu" style={{ display: "block", top: "45px", right: "10px" }}>
      <button className="menu-item" onClick={handleLoadHistory}>
        Voir l'historique
      </button>

      {/* AJOUT ICI : Option renommer si c'est un groupe */}
      {selectedChat.type === "group" && (
        <button className="menu-item" onClick={(e) => onRename(e, selectedChat)}>
          Modifier le nom
        </button>
      )}

      <button className="menu-item delete" onClick={(e) => onDeleteChat(e, selectedChat)}>
        Supprimer la discussion
      </button>
    </div>
  )}
</div>
              </header>

              <div className="chat-body">
                {messages.map((msg) => {
                  const isSentByMe = String(msg.sender_id) === String(currentUserId);
                  return (
                    <div key={msg.id} className={`message-row ${isSentByMe ? "message-out" : "message-in"}`}>
                      <div className="message-bubble">
                        <div className="message-text">
                          <span className="content">{msg.content}</span>
                          
                          {/* MENU DE SUPPRESSION MESSAGE RÉINTÉGRÉ */}
                          <div className="msg-menu-container">
                            <button 
                                className="msg-options-trigger"
                                 style={{border: 'none', backgroundColor:'white'}}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    setOpenMenuId(openMenuId === msg.id ? null : msg.id);
                                }}
                            >
                              <span style={{border: 'none', backgroundColor:'#f0f2f5', cursor: 'pointer'}}>⋯</span>
                            </button>
                            {openMenuId === msg.id && (
                              <div className="msg-popup-menu">
                                {isSentByMe && (
                                  <button className="menu-item delete" onClick={() => deleteMessage(msg.id)}>
                                    Supprimer pour moi
                                  </button>
                                )}
                                <button className="menu-item">Répondre</button>
                              </div>
                            )}
                          </div>
                        </div>

                        <div className="message-footer">
                          <span className="message-time">
                            {new Date(msg.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                          </span>
                          {isSentByMe && (
                            <span 
                              style={{ fontSize: "14px", marginLeft: "4px", color: msg.is_read ? "#53bdeb" : "#919191" }}
                            >
                              {msg.is_read ? "✓✓" : "✓"}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
                <div ref={messagesEndRef} />
              </div>

              <footer className="chat-input-area">
                <form onSubmit={handleSend} className="input-form">
                  <input
                    type="text"
                    placeholder="Taper un message"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                  />
                  <button type="submit" className="send-btn" style={{border:'none', backgroundColor:'white', padding: '1px 20px', cursor:'pointer'}}>
                    ➤
                  </button>
                </form>
              </footer>
            </div>
          ) : (
            <div className="whatsapp-welcome">
              <div className="welcome-content">
                <h2>Groupe4App</h2>
                <p>Sélectionnez une discussion pour commencer.</p>
              </div>
            </div>
          )}
        </main>

        {/* MODAL CRÉATION */}
        {showModal && (
          <div className="modal-backdrop" onClick={() => { setShowModal(false); setContactSearch(""); searchUsers(""); }}>
            <div className="modal-box" onClick={(e) => e.stopPropagation()}>
              <header className="modal-header">
                <h2>{newChatData.type === "groupe" ? "Nouveau Groupe" : "Nouveau Message"}</h2>
                <button onClick={() => { setShowModal(false); setContactSearch(""); searchUsers(""); }} className="close-modal-btn">✕</button>
              </header>
              <form onSubmit={handleStartChat} className="modal-form">
                <div className="type-selector">
                  <button type="button" className={newChatData.type === "private" ? "active" : ""} onClick={() => setNewChatData({ ...newChatData, type: "private", participants: [] })}>Privé</button>
                  <button type="button" className={newChatData.type === "groupe" ? "active" : ""} onClick={() => setNewChatData({ ...newChatData, type: "groupe", participants: [] })}>Groupe</button>
                </div>
                {newChatData.type === "groupe" && (
                  <div className="form-group">
                    <label>Nom du groupe</label>
                    <input type="text" value={newChatData.name} onChange={(e) => setNewChatData({ ...newChatData, name: e.target.value })} required />
                  </div>
                )}

                {newChatData.participants.length > 0 && (
                  <div className="selected-chips">
                    {newChatData.participants.map((pid) => {
                      const u = users.find((x) => x.id === pid);
                      return (
                        <span key={pid} className="chip" onClick={() => toggleParticipant(pid)}>
                          {u?.username || pid.substring(0, 8)}
                          <span className="chip-remove">×</span>
                        </span>
                      );
                    })}
                  </div>
                )}

                <div className="form-group">
                  <label>Sélectionner des contacts</label>
                  <div className="modal-search-wrapper">
                    <input
                      type="text"
                      placeholder="Rechercher un contact..."
                      value={contactSearch}
                      onChange={(e) => handleContactSearch(e.target.value)}
                    />
                  </div>
                </div>

                <div className="modal-contact-list">
                  {availableContacts.length === 0 ? (
                    <p className="modal-empty">{contactSearch.trim() ? "Aucun contact trouvé" : "Tapez un nom pour rechercher"}</p>
                  ) : (
                    availableContacts.map((user) => {
                      const isSelected = newChatData.participants.includes(user.id);
                      return (
                        <div
                          key={user.id}
                          className={`modal-contact-item ${isSelected ? "selected" : ""}`}
                          onClick={() => {
                            if (newChatData.type === "private") {
                              setNewChatData({ ...newChatData, participants: isSelected ? [] : [user.id] });
                            } else {
                              toggleParticipant(user.id);
                            }
                          }}
                        >
                          <img
                            src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${user.username}`}
                            alt={user.username}
                            className="modal-contact-avatar"
                          />
                          <div className="modal-contact-info">
                            <span className="modal-contact-name">{user.username}</span>
                            {user.email && <span className="modal-contact-email">{user.email}</span>}
                          </div>
                          <div className={`modal-check ${isSelected ? "checked" : ""}`}>
                            {isSelected && "✓"}
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>

                <button
                  type="submit"
                  className="btn-whatsapp"
                  disabled={isCreating || newChatData.participants.length === 0}
                >
                  {isCreating ? "Création..." : `Créer${newChatData.participants.length > 0 ? ` (${newChatData.participants.length})` : ""}`}
                </button>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Chats;