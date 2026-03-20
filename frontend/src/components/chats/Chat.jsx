// import React, { useState, useEffect, useCallback } from 'react';
// import axios from 'axios';
// import '../styles/chat.css';

// const Chats = () => {
//   const [chats, setChats] = useState([]);
//   const [selectedChat, setSelectedChat] = useState(null);
//   const [showModal, setShowModal] = useState(false);
//   const [newChatData, setNewChatData] = useState({ participants: "", type: "private", name: "" });
//   const [loading, setLoading] = useState(true);
//   const [isCreating, setIsCreating] = useState(false); // New state for button loading

//   const apiUrl = process.env.REACT_APP_API_URL_CHAT || 'http://localhost:8080/api/v1';

//   const getAuthHeader = () => {
//     const token = localStorage.getItem('token');
//     return token ? { Authorization: `Bearer ${token}` } : {};
//   };

//   // 1. GET: Fetch chats
//   const fetchChats = useCallback(async () => {
//     try {
//       setLoading(true);
//       const res = await axios.get(`${apiUrl}/chats`, {
//         headers: getAuthHeader()
//       });
//       setChats(res.data || []);
//     } catch (err) {
//       console.error("Erreur lors du chargement:", err);
//     } finally {
//       setLoading(false);
//     }
//   }, [apiUrl]);

//   useEffect(() => { fetchChats(); }, [fetchChats]);

//   // 2. POST: Create chat (Optimized for your Postman structure)
//   const handleCreateChat = async (e) => {
//     e.preventDefault();
//     setIsCreating(true);

//     try {
//       // Deduplicate participants and remove empty spaces
//       const participantsArray = [
//         ...new Set(newChatData.participants.split(',').map(id => id.trim()))
//       ].filter(id => id !== "");

//       const payload = {
//         participants: participantsArray,
//         type: newChatData.type,
//         name: newChatData.type === 'group' ? newChatData.name : "Private Chat"
//       };

//       const response = await axios.post(`${apiUrl}/chats`, payload, {
//         headers: getAuthHeader()
//       });

//       // Close modal and reset form
//       setShowModal(false);
//       setNewChatData({ participants: "", type: "private", name: "" });
      
//       // Refresh list and select the new chat automatically
//       await fetchChats();
//       if (response.data && response.data.id) {
//         setSelectedChat(response.data);
//       }

//     } catch (err) {
//       console.error("Erreur creation chat:", err);
//       alert("Erreur: " + (err.response?.data?.error || "Impossible de créer le chat"));
//     } finally {
//       setIsCreating(false);
//     }
//   };

//   return (
//     <div className="whatsapp-main">
//       <div className="sidebar">
//         <div className="sidebar-header">
//           <div className="user-avatar">Me</div>
//           <div className="header-icons">
//             <span onClick={() => setShowModal(true)} style={{cursor:'pointer', fontSize: '20px'}}>➕</span>
//           </div>
//         </div>

//         <div className="chat-list">
//           {loading ? (
//             <p className="loading-msg">Chargement...</p>
//           ) : (
//             chats.map((chat) => (
//               <div 
//                 key={chat.id} 
//                 className={`chat-item ${selectedChat?.id === chat.id ? 'active' : ''}`}
//                 onClick={() => setSelectedChat(chat)}
//               >
//                 <div className="chat-avatar">{chat.type === 'group' ? '👥' : '👤'}</div>
//                 <div className="chat-info">
//                   <span className="chat-name">{chat.name || "Discussion"}</span>
//                   <p className="chat-preview">{chat.participants?.length || 0} participants</p>
//                 </div>
//               </div>
//             ))
//           )}
//         </div>
//       </div>

//       <div className="chat-window">
//         {selectedChat ? (
//           <div className="window-content">
//             <div className="window-header"><h3>{selectedChat.name}</h3></div>
//             <div className="message-area">
//                <p className="system-msg">Discussion sécurisée (ID: {selectedChat.id})</p>
//             </div>
//             <div className="input-area">
//               <input type="text" placeholder="Taper un message" />
//               <button className="send-btn">➤</button>
//             </div>
//           </div>
//         ) : (
//           <div className="empty-state"><h2>Sélectionnez une discussion</h2></div>
//         )}
//       </div>

//       {showModal && (
//         <div className="modal-overlay">
//           <div className="modal-content">
//             <h3>Nouvelle Discussion</h3>
//             <form onSubmit={handleCreateChat}>
//               <select value={newChatData.type} onChange={e => setNewChatData({...newChatData, type: e.target.value})}>
//                 <option value="private">Privé (1-on-1)</option>
//                 <option value="group">Groupe</option>
//               </select>

//               <input 
//                 placeholder="UUIDs des participants (séparés par une virgule)" 
//                 value={newChatData.participants}
//                 onChange={e => setNewChatData({...newChatData, participants: e.target.value})}
//                 required
//               />

//               {newChatData.type === 'group' && (
//                 <input 
//                   placeholder="Nom du groupe" 
//                   value={newChatData.name}
//                   onChange={e => setNewChatData({...newChatData, name: e.target.value})}
//                   required
//                 />
//               )}

//               <div className="modal-buttons">
//                 <button type="button" onClick={() => setShowModal(false)}>Annuler</button>
//                 <button type="submit" disabled={isCreating}>
//                   {isCreating ? "Création..." : "Créer"}
//                 </button>
//               </div>
//             </form>
//           </div>
//         </div>
//       )}
//     </div>
//   );
// };

// export default Chats;


import React, { useState, useEffect, useCallback, useMemo } from 'react';
import axios from 'axios';
import '../styles/chat.css';

const Chats = () => {
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [newChatData, setNewChatData] = useState({ participants: "", type: "private", name: "" });
  const [loading, setLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);

  // Updated to match your Postman port: 8088
  const apiUrl = process.env.REACT_APP_API_URL_CHAT || 'http://localhost:8080/api/v1';

  // Memoized Auth Headers
  const authHeaders = useMemo(() => {
    const token = localStorage.getItem('token');
    return {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    };
  }, []);

  // 1. GET: Fetch chats
  const fetchChats = useCallback(async () => {
  const token = localStorage.getItem('token'); 
  
  // Safety check: if no token, don't even try the request
  if (!token) {
    console.error("No token found in localStorage!");
    return;
  }

  try {
    setLoading(true);
    const res = await axios.get(`${apiUrl}/chats`, {
      headers: {
        // MUST BE EXACTLY THIS STRING
        'Authorization': `Bearer ${token}` 
      }
    });
    setChats(res.data || []);
  } catch (err) {
    // This will tell us exactly WHY the Go backend said 401
    console.error("Backend Error Message:", err.response?.data?.error);
    
    if (err.response?.status === 401) {
      // If the token is truly dead, clear it
      // localStorage.removeItem('token');
    }
  } finally {
    setLoading(false);
  }
}, [apiUrl]);


  useEffect(() => {
    fetchChats();
  }, [fetchChats]);

  // 2. POST: Create chat
  const handleCreateChat = async (e) => {
    e.preventDefault();
    setIsCreating(true);

    try {
      // Clean input: Split string to Array and remove duplicates/whitespace
      const participantsArray = [
        ...new Set(newChatData.participants.split(',').map(id => id.trim()))
      ].filter(id => id.length > 0);

      // EXACT Payload as seen in your Postman
      const payload = {
        participants: participantsArray,
        type: newChatData.type,
        name: newChatData.name || "Groupe4"
      };

      const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);

      // UI Reset
      setShowModal(false);
      setNewChatData({ participants: "", type: "private", name: "" });
      
      // Refresh and auto-select new chat
      await fetchChats();
      if (response.data?.id) setSelectedChat(response.data);

    } catch (err) {
      alert("Erreur: " + (err.response?.data?.error || "Connexion refusée"));
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="whatsapp-main">
      {/* Sidebar Section */}
      <div className="sidebar">
        <div className="sidebar-header">
          <div className="user-avatar">Me</div>
          <div className="header-icons">
            <span onClick={() => setShowModal(true)} style={{ cursor: 'pointer' }}>➕</span>
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

      {/* Main Chat Window */}
      <div className="chat-window">
        {selectedChat ? (
          <div className="window-content">
            <div className="window-header"><h3>{selectedChat.name}</h3></div>
            <div className="message-area">
               <p className="system-msg">Discussion créée le {new Date(selectedChat.created_at).toLocaleDateString()}</p>
               <p className="system-msg">ID: {selectedChat.id}</p>
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

      {/* Create Chat Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <h3>Nouvelle Discussion</h3>
            <form onSubmit={handleCreateChat}>
              <select value={newChatData.type} onChange={e => setNewChatData({...newChatData, type: e.target.value})}>
                <option value="private">Privé</option>
                <option value="group">Groupe</option>
              </select>

              <input 
                placeholder="UUIDs des participants (id1, id2...)" 
                value={newChatData.participants}
                onChange={e => setNewChatData({...newChatData, participants: e.target.value})}
                required
              />

              <input 
                placeholder="Nom du groupe" 
                value={newChatData.name}
                onChange={e => setNewChatData({...newChatData, name: e.target.value})}
                required
              />

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