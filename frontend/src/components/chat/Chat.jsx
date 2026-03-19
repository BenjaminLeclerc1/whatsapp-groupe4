import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import '../styles/chat.css';

const Chats = () => {
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [newChatData, setNewChatData] = useState({ participants: "", type: "private", name: "" });
  const [loading, setLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false); // New state for button loading

  const apiUrl = process.env.REACT_APP_API_URL_CHAT || 'http://localhost:8080/api/v1';

  const getAuthHeader = () => {
    const token = localStorage.getItem('token');
    return token ? { Authorization: `Bearer ${token}` } : {};
  };

  // 1. GET: Fetch chats
  const fetchChats = useCallback(async () => {
    try {
      setLoading(true);
      const res = await axios.get(`${apiUrl}/chats`, {
        headers: getAuthHeader()
      });
      setChats(res.data || []);
    } catch (err) {
      console.error("Erreur lors du chargement:", err);
    } finally {
      setLoading(false);
    }
  }, [apiUrl]);

  useEffect(() => { fetchChats(); }, [fetchChats]);

  // 2. POST: Create chat (Optimized for your Postman structure)
  const handleCreateChat = async (e) => {
    e.preventDefault();
    setIsCreating(true);

    try {
      // Deduplicate participants and remove empty spaces
      const participantsArray = [
        ...new Set(newChatData.participants.split(',').map(id => id.trim()))
      ].filter(id => id !== "");

      const payload = {
        participants: participantsArray,
        type: newChatData.type,
        name: newChatData.type === 'group' ? newChatData.name : "Private Chat"
      };

      const response = await axios.post(`${apiUrl}/chats`, payload, {
        headers: getAuthHeader()
      });

      // Close modal and reset form
      setShowModal(false);
      setNewChatData({ participants: "", type: "private", name: "" });
      
      // Refresh list and select the new chat automatically
      await fetchChats();
      if (response.data && response.data.id) {
        setSelectedChat(response.data);
      }

    } catch (err) {
      console.error("Erreur creation chat:", err);
      alert("Erreur: " + (err.response?.data?.error || "Impossible de créer le chat"));
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="whatsapp-main">
      <div className="sidebar">
        <div className="sidebar-header">
          <div className="user-avatar">Me</div>
          <div className="header-icons">
            <span onClick={() => setShowModal(true)} style={{cursor:'pointer', fontSize: '20px'}}>➕</span>
          </div>
        </div>

        <div className="chat-list">
          {loading ? (
            <p className="loading-msg">Chargement...</p>
          ) : (
            chats.map((chat) => (
              <div 
                key={chat.id} 
                className={`chat-item ${selectedChat?.id === chat.id ? 'active' : ''}`}
                onClick={() => setSelectedChat(chat)}
              >
                <div className="chat-avatar">{chat.type === 'group' ? '👥' : '👤'}</div>
                <div className="chat-info">
                  <span className="chat-name">{chat.name || "Discussion"}</span>
                  <p className="chat-preview">{chat.participants?.length || 0} participants</p>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="chat-window">
        {selectedChat ? (
          <div className="window-content">
            <div className="window-header"><h3>{selectedChat.name}</h3></div>
            <div className="message-area">
               <p className="system-msg">Discussion sécurisée (ID: {selectedChat.id})</p>
            </div>
            <div className="input-area">
              <input type="text" placeholder="Taper un message" />
              <button className="send-btn">➤</button>
            </div>
          </div>
        ) : (
          <div className="empty-state"><h2>Sélectionnez une discussion</h2></div>
        )}
      </div>

      {showModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <h3>Nouvelle Discussion</h3>
            <form onSubmit={handleCreateChat}>
              <select value={newChatData.type} onChange={e => setNewChatData({...newChatData, type: e.target.value})}>
                <option value="private">Privé (1-on-1)</option>
                <option value="group">Groupe</option>
              </select>

              <input 
                placeholder="UUIDs des participants (séparés par une virgule)" 
                value={newChatData.participants}
                onChange={e => setNewChatData({...newChatData, participants: e.target.value})}
                required
              />

              {newChatData.type === 'group' && (
                <input 
                  placeholder="Nom du groupe" 
                  value={newChatData.name}
                  onChange={e => setNewChatData({...newChatData, name: e.target.value})}
                  required
                />
              )}

              <div className="modal-buttons">
                <button type="button" onClick={() => setShowModal(false)}>Annuler</button>
                <button type="submit" disabled={isCreating}>
                  {isCreating ? "Création..." : "Créer"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Chats;


// import React, { useState, useEffect, useCallback } from 'react';
// import axios from 'axios';
// import '../styles/chat.css';

// const Chats = () => {
//   const [chats, setChats] = useState([]);
//   const [selectedChat, setSelectedChat] = useState(null);
//   const [showModal, setShowModal] = useState(false);
//   const [newChatData, setNewChatData] = useState({ participants: "", type: "private", name: "" });
//   const [loading, setLoading] = useState(true);

//   // Use the Gateway Port (8080)
//   const apiUrl = process.env.REACT_APP_API_URL_CHAT || 'http://localhost:8080/api/v1';

//   // Helper to get token (fresher than a static variable)
//   const getAuthHeader = () => {
//     const token = localStorage.getItem('token');
//     return token ? { Authorization: `Bearer ${token}` } : {};
//   };

//   // 1. GET: Fetch chats 
//   // Added useCallback to prevent unnecessary re-renders
//   const fetchChats = useCallback(async () => {
//     try {
//       setLoading(true);
//       // 🔥 FIX: No trailing slash at the end of /chats to prevent 307 Redirect
//       const res = await axios.get(`${apiUrl}/chats`, {
//         headers: getAuthHeader()
//       });
//       setChats(res.data || []);
//     } catch (err) {
//       console.error("Erreur lors du chargement des discussions", err);
//       if (err.response?.status === 401) alert("Session expirée, reconnectez-vous.");
//     } finally {
//       setLoading(false);
//     }
//   }, [apiUrl]);

//   useEffect(() => { 
//     fetchChats(); 
//   }, [fetchChats]);

//   // 2. POST: Create chat
//   const handleCreateChat = async (e) => {
//     e.preventDefault();
//     try {
//       const payload = {
//         // Filter out empty strings if the user puts extra commas
//         participants: newChatData.participants.split(',').map(id => id.trim()).filter(id => id !== ""),
//         type: newChatData.type,
//         name: newChatData.name
//       };

//       // 🔥 FIX: Ensure the POST matches the backend route exactly
//       await axios.post(`${apiUrl}/chats`, payload, {
//         headers: getAuthHeader()
//       });
      
//       setShowModal(false);
//       setNewChatData({ participants: "", type: "private", name: "" }); // Reset form
//       fetchChats(); 
//     } catch (err) {
//       console.error("Erreur creation chat:", err);
//       alert("Erreur: " + (err.response?.data?.error || "Impossible de créer le chat"));
//     }
//   };

//   return (
//     <div className="whatsapp-main">
//       {/* SIDEBAR */}
//       <div className="sidebar">
//         <div className="sidebar-header">
//           <div className="user-avatar">Me</div>
//           <div className="header-icons">
//             <span onClick={() => setShowModal(true)} style={{cursor:'pointer', fontSize: '20px'}} title="Nouveau Chat">➕</span>
//             <span style={{cursor:'pointer', marginLeft: '15px'}}>⋮</span>
//           </div>
//         </div>

//         <div className="chat-list">
//           {loading ? (
//             <p className="loading-msg">Chargement...</p>
//           ) : chats.length === 0 ? (
//             <p className="empty-msg">Aucune discussion</p>
//           ) : (
//             chats.map((chat) => (
//               <div 
//                 key={chat.id} 
//                 className={`chat-item ${selectedChat?.id === chat.id ? 'active' : ''}`}
//                 onClick={() => setSelectedChat(chat)}
//               >
//                 <div className="chat-avatar">{chat.type === 'group' ? '👥' : '👤'}</div>
//                 <div className="chat-info">
//                   <div className="chat-top-row">
//                     <span className="chat-name">{chat.name || "Discussion"}</span>
//                     {chat.created_at && (
//                       <span className="chat-time">
//                         {new Date(chat.created_at).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
//                       </span>
//                     )}
//                   </div>
//                   <p className="chat-preview">
//                     {chat.participants?.length || 0} participant(s) • {chat.type}
//                   </p>
//                 </div>
//               </div>
//             ))
//           )}
//         </div>
//       </div>

//       {/* CHAT WINDOW */}
//       <div className="chat-window">
//         {selectedChat ? (
//           <div className="window-content">
//             <div className="window-header">
//               <h3>{selectedChat.name || (selectedChat.type === 'private' ? "Chat Privé" : "Groupe")}</h3>
//             </div>
//             <div className="message-area">
//                <p className="system-msg">Début de la discussion sécurisée avec {selectedChat.name || "cet utilisateur"}</p>
//             </div>
//             <div className="input-area">
//               <input type="text" placeholder="Taper un message" />
//               <button className="send-btn">➤</button>
//             </div>
//           </div>
//         ) : (
//           <div className="empty-state">
//             <div className="intro-icon">💬</div>
//             <h2>Sélectionnez une discussion</h2>
//             <p>Envoyez et recevez des messages sans garder votre téléphone en ligne.</p>
//           </div>
//         )}
//       </div>

//       {/* NEW CHAT MODAL */}
//       {showModal && (
//         <div className="modal-overlay">
//           <div className="modal-content">
//             <h3>Nouvelle Discussion</h3>
//             <form onSubmit={handleCreateChat}>
//               <label>Type de discussion</label>
//               <select 
//                 value={newChatData.type}
//                 onChange={e => setNewChatData({...newChatData, type: e.target.value})}
//               >
//                 <option value="private">Privé (1-on-1)</option>
//                 <option value="group">Groupe</option>
//               </select>

//               <label>Participants (IDs séparés par des virgules)</label>
//               <input 
//                 placeholder="ex: uuid-123, uuid-456" 
//                 value={newChatData.participants}
//                 onChange={e => setNewChatData({...newChatData, participants: e.target.value})}
//                 required
//               />

//               {newChatData.type === 'group' && (
//                 <>
//                   <label>Nom du groupe</label>
//                   <input 
//                     placeholder="Nom du groupe" 
//                     value={newChatData.name}
//                     onChange={e => setNewChatData({...newChatData, name: e.target.value})}
//                     required
//                   />
//                 </>
//               )}

//               <div className="modal-buttons">
//                 <button type="button" className="cancel-btn" onClick={() => setShowModal(false)}>Annuler</button>
//                 <button type="submit" className="create-confirm">Créer</button>
//               </div>
//             </form>
//           </div>
//         </div>
//       )}
//     </div>
//   );
// };

// export default Chats;