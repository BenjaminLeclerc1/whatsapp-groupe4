

import React, { useState, useEffect, useCallback, useMemo, useRef } from "react";
import axios from "axios";
import "../styles/chat.css";

const Chats = () => {
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [loading, setLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);

  // --- MESSAGES STATES ---
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState("");
  const messagesEndRef = useRef(null);
  
  // Get current user info from storage
  const currentUserId = localStorage.getItem("user_id");
  console.log("Current User ID:", currentUserId);

  const [newChatData, setNewChatData] = useState({
    participants: "",
    type: "private",
    name: "",
  });

  const apiUrl = "http://localhost:8080/api/v1";
  // const apiUrl1 = "http://localhost:8082/api/v1";

  // --- FIX 1: Robust Auth Headers ---
  // const authHeaders = useMemo(() => {
  //   const token = localStorage.getItem("token");
  //   // Only return headers if token is a real string (not null or "undefined")
  //   if (!token || token === "undefined") return { headers: {} };
    
  //   return {
  //     headers: {
  //       Authorization: `Bearer ${token}`,
  //       "Content-Type": "application/json",
  //     },
  //   };
  // }, []);

const authHeaders = useMemo(() => {
  const token = localStorage.getItem("token");
  const userId = localStorage.getItem("user_id");

  const headersObject = {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      "X-User-ID": userId
    },
  };

  console.log("Auth Headers:", headersObject);

  return headersObject;
}, []);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  // 1. Fetch Chats
  const fetchChats = useCallback(async () => {
    if (!authHeaders.headers.Authorization) return;

    try {
      const res = await axios.get(`${apiUrl}/chats`, authHeaders);
      setChats(res.data || []);
    } catch (err) {
      console.error("Fetch Error:", err.response?.data?.error || err.message);
    } finally {
      setLoading(false);
    }
  }, [apiUrl, authHeaders]);

  // 2. Fetch Messages for Selected Chat
  const fetchMessages = useCallback(async (chatId) => {
    // if (!chatId || !authHeaders.headers.Authorization) return;
    if (!chatId) return;
    try {
      // FIX 2: Check if your backend expects /messages/:chatId or /messages/chat/:chatId
      // const res = await axios.get(`${apiUrl}/messages/${chatId}`, authHeaders);
      // Change this line in fetchMessages (React)
const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
      const data = res.data.messages || res.data || [];
      setMessages(Array.isArray(data) ? data : []);
      setTimeout(scrollToBottom, 100);
    } catch (err) {
      console.error("Fetch Messages Error:", err.message);
    }
  }, [apiUrl, authHeaders]);

  useEffect(() => {
    fetchChats();
    const interval = setInterval(fetchChats, 10000);
    return () => clearInterval(interval);
  }, [fetchChats]);

  useEffect(() => {
    if (selectedChat) {
      fetchMessages(selectedChat.id);
    } else {
      setMessages([]);
    }
  }, [selectedChat, fetchMessages]);

  // 3. Send Message
  const handleSendMessage = async (e) => {
    e.preventDefault();
    if (!newMessage.trim() || !selectedChat) return;

    const payload = {
      chat_id: selectedChat.id,
      content: newMessage,
    };

    try {
      // Note: check if endpoint should be /messages or /messages/ (trailing slash)
      const res = await axios.post(`${apiUrl}/messages`, payload, authHeaders);
      
      // FIX 3: Ensure the local UI update has the currentUserId to apply "sent" CSS
      const sentMsg = res.data.message || res.data;
      
      setMessages((prev) => [...prev, {
        ...sentMsg,
        sender_id: currentUserId // Force current user ID if backend response is nested
      }]);
      
      setNewMessage("");
      setTimeout(scrollToBottom, 50);
    } catch (err) {
      console.error("Send Error:", err.response?.data?.error || err.message);
    }


  };

  // 4. Create Chat (Logic preserved)
  const handleCreateChat = async (e) => {
    e.preventDefault();
    setIsCreating(true);

    try {
      const inputIds = newChatData.participants.split(",").map((id) => id.trim());
      const participantsArray = [...new Set([...inputIds, currentUserId])].filter((id) => id && id.length > 5);

      const payload = {
        participants: participantsArray,
        type: newChatData.type,
        name: newChatData.type === "group" ? newChatData.name : "Private Chat",
      };

      const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);
      setShowModal(false);
      setNewChatData({ participants: "", type: "private", name: "" });
      await fetchChats(); 
      if (response.data) setSelectedChat(response.data);
    } catch (err) {
      alert("Erreur: " + (err.response?.data?.error || "Vérifiez les UUIDs"));
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="whatsapp-layout">
      {/* SIDEBAR */}
      <aside className="sidebar">
        <header className="sidebar-header">
          <div className="profile-img">
            <img src={`https://ui-avatars.com/api/?name=Me&background=random`} alt="user" />
          </div>
          <div className="header-actions">
            <button className="icon-btn" onClick={() => setShowModal(true)}>
              <span className="material-icons">add_comment</span>
            </button>
          </div>
        </header>

        <div className="search-bar">
          <div className="search-inner">
            <span className="material-icons">search</span>
            <input type="text" placeholder="Rechercher..." />
          </div>
        </div>

        <div className="chat-list">
          {loading && chats.length === 0 ? (
            <div className="loader-container"><div className="loader"></div></div>
          ) : (
            chats.map((chat) => (
              <div
                key={chat.id}
                className={`chat-card ${selectedChat?.id === chat.id ? "active" : ""}`}
                onClick={() => setSelectedChat(chat)}
              >
                <div className="card-avatar">{chat.type === "group" ? "👥" : "👤"}</div>
                <div className="card-content">
                  <div className="card-top">
                    <span className="chat-title">{chat.name || "Discussion"}</span>
                    <span className="chat-time">
                      {chat.created_at ? new Date(chat.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : ""}
                    </span>
                  </div>
                  <div className="card-bottom">
                    <span className="last-msg">{chat.participants?.length} membres</span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </aside>

      {/* MAIN CHAT */}
      <main className="chat-container">
        {selectedChat ? (
          <div className="active-chat">
            <header className="chat-header">
              <div className="header-info">
                <h3>{selectedChat.name}</h3>
                <p>{selectedChat.type} • {selectedChat.participants?.length} participants</p>
              </div>
            </header>

            <div className="messages-viewport">
              <div className="msg-bubble system">
                Discussion créée le {new Date(selectedChat.created_at).toLocaleDateString()}
              </div>

              {messages.map((msg) => (
                <div 
                  key={msg.id} 
                  className={`msg-wrapper ${String(msg.sender_id) === String(currentUserId) ? "sent" : "received"}`}
                >
                  <div className="msg-bubble">
                    <div className="msg-text">{msg.content}</div>
                    <div className="msg-meta">
                      {new Date(msg.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                    </div>
                  </div>
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            <form className="chat-footer" onSubmit={handleSendMessage}>
              <button type="button" className="icon-btn">📎</button>
              <input 
                type="text" 
                placeholder="Taper un message..." 
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
              />
              <button type="submit" className="send-btn">
                <span className="material-icons">send</span>
              </button>
            </form>
          </div>
        ) : (
          <div className="welcome-screen">
            <h2>WhatsApp Groupe 4</h2>
            <p>Sélectionnez une discussion pour commencer.</p>
          </div>
        )}
      </main>

      {/* MODAL (Preserved) */}
      {showModal && (
        <div className="modal-backdrop">
          <div className="modal-box">
            <div className="modal-header">
              <h2>Démarrer une conversation</h2>
              <button onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateChat}>
              <div className="form-group">
                <label>Type</label>
                <select
                  value={newChatData.type}
                  onChange={(e) => setNewChatData({ ...newChatData, type: e.target.value })}
                >
                  <option value="private">🔒 Privée</option>
                  <option value="group">👥 Groupe</option>
                </select>
              </div>
              {newChatData.type === "group" && (
                <div className="form-group">
                  <label>Nom du Groupe</label>
                  <input
                    type="text"
                    value={newChatData.name}
                    onChange={(e) => setNewChatData({ ...newChatData, name: e.target.value })}
                    required
                  />
                </div>
              )}
              <div className="form-group">
                <label>Participants (UUIDs)</label>
                <textarea
                  value={newChatData.participants}
                  onChange={(e) => setNewChatData({ ...newChatData, participants: e.target.value })}
                  required
                />
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)}>Annuler</button>
                <button type="submit" className="btn-primary" disabled={isCreating}>Lancer</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Chats;




// import React, { useContext, useState } from "react";
// import { AppContext } from "../../context/AppContext";
// import { AppContext } from "../context/AppContext";

// const Chats = () => {

//   const {
//     chats,
//     messages,
//     selectedChat,
//     setSelectedChat,
//     sendMessage,
//     createChat,
//     loading
//   } = useContext(AppContext);

//   const [newMessage, setNewMessage] = useState("");

//   const handleSend = (e) => {
//     e.preventDefault();

//     if (!newMessage.trim() || !selectedChat) return;

//     sendMessage(selectedChat.id, newMessage);

//     setNewMessage("");
//   };

//   return (
//     <div>

//       {chats.map(chat => (
//         <div key={chat.id} onClick={() => setSelectedChat(chat)}>
//           {chat.name}
//         </div>
//       ))}

//       {messages.map(msg => (
//         <div key={msg.id}>{msg.content}</div>
//       ))}

//       <form onSubmit={handleSend}>
//         <input
//           value={newMessage}
//           onChange={(e)=>setNewMessage(e.target.value)}
//         />
//       </form>

//     </div>
//   );
// };

// export default Chats;