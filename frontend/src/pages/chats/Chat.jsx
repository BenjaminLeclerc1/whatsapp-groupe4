

import React, { useState, useRef, useEffect } from "react";
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
    deleteMessage, // <-- RÉINTÉGRÉ
    getHistory,
    notifications,
    markNotificationsAsRead,
  } = useApp();

  const [newMessage, setNewMessage] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [newChatData, setNewChatData] = useState({
    participants: "",
    type: "private",
    name: "",
  });

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

  const handleStartChat = async (e) => {
    e.preventDefault();
    setIsCreating(true);
    try {
      const participantsArray = newChatData.participants
        .split(",")
        .map((id) => id.trim())
        .filter((id) => id !== "");

      await createChat({
        participants: participantsArray,
        type: newChatData.type === "groupe" ? "group" : "private",
        name: newChatData.name,
      });
      setShowModal(false);
      setNewChatData({ participants: "", type: "private", name: "" });
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
              <img src={`https://ui-avatars.com/api/?name=User&background=075E54&color=fff`} alt="me" />
            </div>
            <div className="header-actions">
              <button className="icon-btn" onClick={() => setShowModal(true)} style={{backgroundColor: 'green', padding: '5px 9px', color:'white'}}>
                <span className="material-icons">+ Nouveau groupe</span>
              </button>
            </div>
          </header>

          <div className="search-container">
            <div className="search-input-wrapper">
              <span className="material-icons">search</span>
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
                      <span className="material-icons">{chat.type === "group" ? "👥" : "👤"}</span>
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
                          <span className="material-icons"  style={{border: 'none', cursor:'pointer'}}>...</span>
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
                <div className="header-avatar">{selectedChat.type === "group" ? "👥" : "👤"}</div>
                <div className="header-contact-info">
                  <h3>{selectedChat.name || "Discussion"}</h3>
                  <p>{selectedChat.participants?.length} participants</p>
                </div>
               {/* MAIN CHAT HEADER ACTIONS */}
<div className="header-actions">
  <button className="icon-btn" onClick={(e) => { e.stopPropagation(); setShowHeaderMenu(!showHeaderMenu); }}>
    <span className="material-icons" style={{marginLeft: '20px', padding: '0px 10px 7px 10px'}}>...</span>
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
        Supprimer le chat
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
                              <span className="material-icons"  style={{border: 'none', backgroundColor:'#f0f2f5'}}>...</span>
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
                              className="material-icons check-mark" 
                              style={{ fontSize: "16px", marginLeft: "4px", color: msg.is_read ? "#53bdeb" : "#919191" }}
                            >
                              {msg.is_read ? "done_all" : "done"} 
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
                <span className="material-icons">insert_emoticon</span>
                <form onSubmit={handleSend} className="input-form">
                  <input
                    type="text"
                    placeholder="Taper un message"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                  />
                  <button type="submit" className="send-btn" style={{border:'none', backgroundColor:'white', padding: '1px 20px', cursor:'pointer'}}>
                    <span className="material-icons">{newMessage.trim() ? "send" : "➤"}</span>
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
          <div className="modal-backdrop">
            <div className="modal-box">
              <header className="modal-header">
                <h2>{newChatData.type === "groupe" ? "Nouveau Groupe" : "Nouveau Message"}</h2>
                <button onClick={() => setShowModal(false)} className="close-modal-btn"  style={{backgroundColor: 'green', color: 'white', border: 'none', padding: '5px 9px', marginRight: '4px', cursor:'pointer'}}>
                   <span className="material-icons">Fermé</span>
                </button>
              </header>
              <form onSubmit={handleStartChat} className="modal-form">
                <div className="type-selector" style={{marginBottom: '11px'}}>
                  <button type="button" className={newChatData.type === "private" ? "active" : ""} onClick={() => setNewChatData({ ...newChatData, type: "private" })} style={{backgroundColor: 'green', color: 'white', border: 'none', padding: '5px 9px', marginRight: '4px', cursor:'pointer'}}>Privé</button>
                  <button type="button" className={newChatData.type === "groupe" ? "active" : ""} onClick={() => setNewChatData({ ...newChatData, type: "groupe" })} style={{backgroundColor: 'green', color: 'white', border: 'none', padding: '5px 9px', marginLeft: '4px', cursor:'pointer'}}>Groupe</button>
                </div>
                {newChatData.type === "groupe" && (
                  <div className="form-group">
                    <label>Nom du groupe</label>
                    <input type="text" value={newChatData.name} onChange={(e) => setNewChatData({ ...newChatData, name: e.target.value })} required style={{border: '1px solid gray'}} />
                  </div>
                )}
                <div className="form-group">
                  <label>ID des participants (séparés par virgule)</label>
                  <textarea value={newChatData.participants} onChange={(e) => setNewChatData({ ...newChatData, participants: e.target.value })} required style={{border: '1px solid gray'}}/>
                </div>
                <button type="submit" className="btn-whatsapp" disabled={isCreating}>LANCER</button>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Chats;