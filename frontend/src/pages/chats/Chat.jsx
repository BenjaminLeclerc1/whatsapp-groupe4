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
    deleteMessage,
    updateChat, // Ajouté via AppContext
    deleteChat, // Ajouté via AppContext
  } = useApp();

  const [newMessage, setNewMessage] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [newChatData, setNewChatData] = useState({
    participants: "",
    type: "private",
    name: "",
  });

  // États pour les menus contextuels
  const [openMenuId, setOpenMenuId] = useState(null); // Menu des messages
  const [openChatActionId, setOpenChatActionId] = useState(null); // Menu des chats (Sidebar)
  const [showHeaderMenu, setShowHeaderMenu] = useState(false); // Menu Header

  const messagesEndRef = useRef(null);

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
      await createChat({
        participants: newChatData.participants,
        type: newChatData.type,
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

  // Fonctions de gestion
  const onRename = (e, chat) => {
    e.stopPropagation();
    const newName = prompt("Nouveau nom du groupe :", chat.name);
    if (newName && newName.trim() !== "") {
      updateChat(chat.id, newName);
    }
  };

  const onDelete = (e, chat) => {
    e.stopPropagation();
    if (
      window.confirm(`Supprimer la discussion "${chat.name || "Privée"}" ?`)
    ) {
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
              <img
                src={`https://ui-avatars.com/api/?name=User&background=075E54&color=fff`}
                alt="me"
              />
            </div>
            <div className="header-actions">
              <button className="icon-btn" onClick={() => setShowModal(true)}>
                <span className="material-icons">+ Créer un groupe</span>
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
              <div className="loader-container">
                <div className="loader"></div>
              </div>
            ) : (
              [...chats]
                .sort(
                  (a, b) =>
                    new Date(b.updated_at || b.created_at) -
                    new Date(a.updated_at || a.created_at),
                )
                .map((chat) => (
                  <div
                    key={chat.id}
                    className={`chat-card ${selectedChat?.id === chat.id ? "active" : ""}`}
                    onClick={() => setSelectedChat(chat)}
                    style={{ position: "relative" }}
                  >
                    <div className="card-avatar">
                      <span className="material-icons">
                        {chat.type === "group" ? "groups" : "person"}
                      </span>
                    </div>
                    <div className="card-info">
                      <div className="card-row">
                        <span className="chat-name">
                          {chat.name || "Discussion"}
                        </span>
                        {/* Petit bouton menu sur chaque carte de chat */}
                        <button
                          className="msg-options-trigger"
                          onClick={(e) => {
                            e.stopPropagation();
                            setOpenChatActionId(
                              openChatActionId === chat.id ? null : chat.id,
                            );
                          }}
                        >
                          <span className="material-icons">expand_more</span>
                        </button>
                      </div>

                      {/* Menu contextuel du Chat dans la Sidebar */}
                      {openChatActionId === chat.id && (
                        <div
                          className="msg-popup-menu"
                          style={{
                            display: "block",
                            right: "10px",
                            top: "30px",
                          }}
                        >
                          {chat.type === "group" && (
                            <button
                              className="menu-item"
                              onClick={(e) => onRename(e, chat)}
                            >
                              Renommer
                            </button>
                          )}
                          <button
                            className="menu-item delete"
                            onClick={(e) => onDelete(e, chat)}
                          >
                            Supprimer
                          </button>
                        </div>
                      )}

                      <div className="card-row">
                        <span className="last-msg">
                          {chat.participants?.length} membres
                        </span>
                      </div>
                    </div>
                  </div>
                ))
            )}
          </div>
        </aside>

        {/* MAIN CHAT AREA */}
        <main className="chat-main">
          {selectedChat ? (
            <div className="active-chat-window">
              <header className="chat-header">
                <div className="header-avatar">
                  {selectedChat.type === "group" ? "👥" : "👤"}
                </div>
                <div className="header-contact-info">
                  <h3>{selectedChat.name || "Discussion"}</h3>
                  <p>{selectedChat.participants?.length} participants</p>
                </div>
                <div
                  className="header-actions"
                  style={{ position: "relative" }}
                >
                  <button
                    className="icon-btn"
                    onClick={(e) => {
                      e.stopPropagation();
                      setShowHeaderMenu(!showHeaderMenu);
                    }}
                  >
                    <span className="material-icons">more_vert</span>
                  </button>

                  {showHeaderMenu && (
                    <div
                      className="msg-popup-menu"
                      style={{ display: "block", top: "45px", right: "10px" }}
                    >
                      {selectedChat.type === "group" && (
                        <button
                          className="menu-item"
                          onClick={(e) => onRename(e, selectedChat)}
                        >
                          Renommer le groupe
                        </button>
                      )}
                      <button
                        className="menu-item delete"
                        onClick={(e) => onDelete(e, selectedChat)}
                      >
                        Supprimer le chat
                      </button>
                    </div>
                  )}
                </div>
              </header>

              <div className="chat-body">
                {messages.map((msg) => {
                  const isSentByMe =
                    String(msg.sender_id) === String(currentUserId);
                  return (
                    <div
                      key={msg.id}
                      className={`message-row ${isSentByMe ? "message-out" : "message-in"}`}
                    >
                      <div className="message-bubble">
                        <div className="message-text">
                          <span className="content">{msg.content}</span>
                          <div className="msg-menu-container">
                            <button
                              className="msg-options-trigger"
                              onClick={(e) => {
                                e.stopPropagation();
                                setOpenMenuId(
                                  openMenuId === msg.id ? null : msg.id,
                                );
                              }}
                            >
                              <span className="material-icons">
                                expand_more
                              </span>
                            </button>
                            {openMenuId === msg.id && (
                              <div className="msg-popup-menu">
                                <button className="menu-item">Répondre</button>
                                {isSentByMe && (
                                  <button
                                    className="menu-item delete"
                                    onClick={() => deleteMessage(msg.id)}
                                  >
                                    Supprimer
                                  </button>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                        <div className="message-footer">
                          <span className="message-time">
                            {new Date(msg.created_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                            })}
                          </span>
                          {isSentByMe && (
                            <span className="material-icons check-mark">
                              done_all
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
                <span className="material-icons">attach_file</span>
                <form onSubmit={handleSend} className="input-form">
                  <input
                    type="text"
                    placeholder="Taper un message"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                  />
                  <button type="submit" className="send-btn">
                    <span className="material-icons">
                      {newMessage.trim() ? "send" : "mic"}
                    </span>
                  </button>
                </form>
              </footer>
            </div>
          ) : (
            <div className="whatsapp-welcome">
              <div className="welcome-content">
                <div className="welcome-img"></div>
                <h2>Group4 App</h2>
                <p>Sélectionnez une discussion pour commencer.</p>
              </div>
            </div>
          )}
        </main>

        {/* MODAL CREATION */}
        {showModal && (
          <div className="modal-backdrop">
            <div className="modal-box">
              <header className="modal-header">
                <h2>
                  {newChatData.type === "groupe"
                    ? "Nouveau Groupe"
                    : "Nouveau Message"}
                </h2>
                <button
                  onClick={() => setShowModal(false)}
                  className="close-modal-btn"
                >
                  <span className="material-icons">close</span>
                </button>
              </header>
              <form onSubmit={handleStartChat} className="modal-form">
                <div className="type-selector">
                  <button
                    type="button"
                    className={newChatData.type === "private" ? "active" : ""}
                    onClick={() =>
                      setNewChatData({ ...newChatData, type: "private" })
                    }
                  >
                    Privé
                  </button>
                  <button
                    type="button"
                    className={newChatData.type === "groupe" ? "active" : ""}
                    onClick={() =>
                      setNewChatData({ ...newChatData, type: "groupe" })
                    }
                  >
                    Groupe
                  </button>
                </div>
                {newChatData.type === "groupe" && (
                  <div className="form-group">
                    <label>Nom du groupe</label>
                    <input
                      type="text"
                      placeholder="Ex: Famille..."
                      value={newChatData.name}
                      onChange={(e) =>
                        setNewChatData({ ...newChatData, name: e.target.value })
                      }
                      required
                    />
                  </div>
                )}
                <div className="form-group">
                  <label>ID des participants (UUID)</label>
                  <textarea
                    placeholder="uuid1, uuid2..."
                    value={newChatData.participants}
                    onChange={(e) =>
                      setNewChatData({
                        ...newChatData,
                        participants: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div className="modal-footer">
                  <button
                    type="submit"
                    className="btn-whatsapp"
                    disabled={isCreating}
                  >
                    {isCreating ? "..." : "LANCER"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Chats;
