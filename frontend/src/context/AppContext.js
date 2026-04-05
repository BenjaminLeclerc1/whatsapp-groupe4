// import React, { createContext, useState, useEffect, useCallback, useMemo, useContext } from "react";
// import axios from "axios";

// const AppContext = createContext();

// export const AppProvider = ({ children }) => {
//   const [users, setUsers] = useState([]); // NEW: State for contacts
//   const [chats, setChats] = useState([]);
//   const [selectedChat, setSelectedChat] = useState(null);
//   const [messages, setMessages] = useState([]);
//   const [loading, setLoading] = useState(true);
//   const [notifications, setNotifications] = useState([]);

  
//   const currentUserId = localStorage.getItem("user_id");
//   const apiUrl = "http://localhost:8080/api/v1";
//   const userApiUrl = "http://localhost:8081/api/v1"; // NEW: User service URL
//   // const historyApiUrl = "http://localhost:8082/api/v1"; // NEW: History service URL

//   // Memoized Headers
//   const authHeaders = useMemo(() => {
//     const token = localStorage.getItem("token");
//     const userId = localStorage.getItem("user_id");
//     return {
//       headers: {
//         Authorization: `Bearer ${token}`,
//         "Content-Type": "application/json",
//         "X-User-ID": userId
//       },
//     };
//   }, []);


//   // --- USER ACTIONS ---

//   // 1. Get all users
// const fetchAllUsers = useCallback(async () => {
//   setLoading(true);
//   try {
//     // 1. Double check this URL in your browser!
//     const res = await axios.get(`http://localhost:8081/api/v1/users`, authHeaders);
    
//     console.log("DEBUG: Raw Response Data ->", res.data);

//     // 2. Flexible extraction: handle different backend formats
//     const data = res.data.users || res.data.data || res.data;

//     if (Array.isArray(data)) {
//       setUsers(data);
//     } else {
//       console.error("API returned success but no array found. Check console log above.");
//       setUsers([]);
//     }
//   } catch (err) {
//     console.error("Fetch Users Error:", err.response?.data || err.message);
//     setUsers([]); // Clear state on error
//   } finally {
//     setLoading(false);
//   }
// }, [authHeaders]);



//   // 2. Get User By ID
//   const getUserById = async (userId) => {
//     try {
//       const res = await axios.get(`${userApiUrl}/users/${userId}`, authHeaders);
//       return res.data;
//     } catch (err) {
//       console.error("Get User Error:", err.message);
//     }
//   };

//   // 3. Update User
//   const updateUser = async (userId, updateData) => {
//     try {
//       const res = await axios.put(`${userApiUrl}/users/${userId}`, updateData, authHeaders);
//       // Update local state so UI reflects change immediately
//       setUsers(prev => prev.map(u => u.id === userId ? { ...u, ...updateData } : u));
//       return true;
//     } catch (err) {
//       console.error("Update User Error:", err.message);
//       return false;
//     }
//   };

//   // 4. Delete User
//   const deleteUser = async (userId) => {
//     if (!window.confirm("Supprimer cet utilisateur définitivement ?")) return;
//     try {
//       await axios.delete(`${userApiUrl}/users/${userId}`, authHeaders);
//       setUsers(prev => prev.filter(u => u.id !== userId));
//       return true;
//     } catch (err) {
//       console.error("Delete User Error:", err.message);
//       return false;
//     }
//   };


// const createChat = async (newChatData) => {
//   let participantsArray = [];

//   // Vérifie si participants est une string ou déjà un tableau
//   if (typeof newChatData.participants === "string") {
//     participantsArray = newChatData.participants
//       .split(",")
//       .map((id) => id.trim())
//       .filter((id) => id.length > 10);
//   } else if (Array.isArray(newChatData.participants)) {
//     participantsArray = newChatData.participants;
//   }

//   // Correction du type pour Go
//   const backendType = newChatData.type === "groupe" ? "group" : newChatData.type;

//   const payload = {
//     participants: participantsArray,
//     type: backendType,
//     name: backendType === "group" ? (newChatData.name || "Nouveau Groupe") : "",
//   };

//   try {
//     const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);
//     const newChat = response.data.chat || response.data;

//     setChats((prev) => [newChat, ...prev]);
//     setSelectedChat(newChat);
//     return newChat;
//   } catch (err) {
//     const errorMsg = err.response?.data?.error || "Erreur lors de la création";
//     console.error("Détails Erreur Backend:", err.response?.data);
//     throw new Error(errorMsg);
//   }
// };


// const fetchChats = useCallback(async () => {
//     if (!authHeaders.headers.Authorization) return;
//     try {
//       const res = await axios.get(`${apiUrl}/chats`, authHeaders);
//       setChats(res.data || []);
//     } catch (err) {
//       console.error("Fetch Chats Error:", err.message);
//     } finally {
//       setLoading(false);
//     }
//   }, [authHeaders]);


//   // 1. Update Chat (Usually for renaming a group)
// const updateChat = async (chatId, updateData) => {
//   try {
//     // Go Backend expects: PUT /api/v1/chats/:id
//     const res = await axios.put(`${apiUrl}/chats/${chatId}`, updateData, authHeaders);
//     const updatedChat = res.data.chat || res.data;

//     setChats((prev) =>
//       prev.map((c) => (c.id === chatId ? { ...c, ...updatedChat } : c))
//     );
    
//     if (selectedChat?.id === chatId) {
//       setSelectedChat(prev => ({ ...prev, ...updatedChat }));
//     }
//     return true;
//   } catch (err) {
//     console.error("Update Chat Error:", err.response?.data || err.message);
//     return false;
//   }
// };

// // 2. Delete Chat
// const deleteChat = async (chatId) => {
//   if (!window.confirm("Supprimer cette discussion ? Cette action est irréversible.")) return;

//   try {
//     // Go Backend expects: DELETE /api/v1/chats/:id
//     await axios.delete(`${apiUrl}/chats/${chatId}`, authHeaders);

//     // Remove from local state
//     setChats((prev) => prev.filter((c) => c.id !== chatId));
    
//     // Clear selection if the deleted chat was open
//     if (selectedChat?.id === chatId) {
//       setSelectedChat(null);
//     }
//     return true;
//   } catch (err) {
//     console.error("Delete Chat Error:", err.response?.data || err.message);
//     alert("Erreur lors de la suppression du chat.");
//     return false;
//   }
// };


// // 5. Get Chat History from History Service
// const getHistory = useCallback(async (chatId) => {
//   if (!chatId) return [];
//   setLoading(true);
//   try {
//     // Appel vers le port 8082
//     const res = await axios.get(`${userApiUrl}/messages/${chatId}`, authHeaders);
    
//     // Extraction flexible des données
//     const data = res.data.messages || res.data.data || res.data;
    
//     const historyMessages = Array.isArray(data) ? data : [];
    
//     // On met à jour l'état local des messages avec l'historique reçu
//     setMessages(historyMessages);
    
//     return historyMessages;
//   } catch (err) {
//     console.error("Get History Error:", err.response?.data || err.message);
//     return [];
//   } finally {
//     setLoading(false);
//   }
// }, [authHeaders]);


//   useEffect(() => {
//   fetchAllUsers();
// }, [fetchAllUsers]);

//   // --- ACTIONS ---

  
// const sendMessage = async (content) => {
//   if (!content.trim() || !selectedChat) return;
  
//   const payload = { 
//     chat_id: selectedChat.id, 
//     content: content 
//   };

//   try {
//     const res = await axios.post(`${apiUrl}/messages`, payload, authHeaders);
    
//     // On extrait le message tel que renvoyé par le backend Go
//     const sentMsg = res.data.message || res.data;

//     // MISE À JOUR LOCALE IMMÉDIATE
//     setMessages((prev) => [...prev, { 
//       ...sentMsg, 
//       sender_id: currentUserId, // On force l'ID pour l'alignement UI
//       created_at: sentMsg.created_at || new Date().toISOString() 
//     }]);

//     // OPTIONNEL : On rafraîchit la liste des chats pour mettre à jour l'ordre dans la sidebar
//     fetchChats(); 
    
//     return true;
//   } catch (err) {
//     console.error("Send Error:", err.response?.data || err.message);
//     return false;
//   }
// };

// //  const fetchMessages = useCallback(async (chatId) => {
// //   if (!chatId) return;
// //   setLoading(true); // Optional: show a small spinner
// //   try {
// //     const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
// //     const data = res.data.messages || res.data || [];
// //     setMessages(Array.isArray(data) ? data : []);

// //     // FIX: Also refresh the chat list so the sidebar knows there are new messages
// //     // This ensures the "updated_at" timestamp moves the chat to the top
// //     await fetchChats(); 
    
// //   } catch (err) {
// //     console.error("Fetch Messages Error:", err.message);
// //   } finally {
// //     setLoading(false);
// //   }
// // }, [authHeaders, fetchChats]); // Make sure fetchChats is in the dependency array

// // const fetchMessages = useCallback(async (chatId, showSpinner = true) => {
// //   if (!chatId) return;
  
// //   // On n'active le loading que si demandé (ex: au premier clic)
// //   if (showSpinner) setLoading(true); 

// //   try {
// //     const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
// //     const data = res.data.messages || res.data || [];
    
// //     // On ne met à jour l'état que si les données ont réellement changé 
// //     // pour éviter des re-renders inutiles
// //     setMessages(prev => {
// //       if (JSON.stringify(prev) === JSON.stringify(data)) return prev;
// //       return Array.isArray(data) ? data : [];
// //     });

// //     await fetchChats(); 
    
// //   } catch (err) {
// //     console.error("Fetch Messages Error:", err.message);
// //   } finally {
// //     setLoading(false);
// //   }
// // }, [authHeaders, fetchChats]);


// const fetchMessages = useCallback(async (chatId, showSpinner = true) => {
//   if (!chatId) return;
//   if (showSpinner) setLoading(true); 

//   try {
//     const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
//     let data = res.data.messages || res.data || [];
    
//     // S'assurer que c'est un tableau
//     if (!Array.isArray(data)) data = [];

//     // --- LOGIQUE D'INVERSION ---
//     // Si ton premier message dans 'data' est le plus récent, on inverse le tableau
//     // pour que le plus récent soit à la FIN (donc en bas de l'écran).
//     const formattedData = [...data].reverse(); 

//     setMessages(prev => {
//       // On compare avec les données formatées
//       if (JSON.stringify(prev) === JSON.stringify(formattedData)) return prev;
//       return formattedData;
//     });

//     await fetchChats(); 
//   } catch (err) {
//     console.error("Fetch Error:", err.message);
//   } finally {
//     setLoading(false);
//   }
// }, [authHeaders, fetchChats]);

//   const deleteMessage = async (messageId) => {
//   // Optional: Add a confirmation dialog
//   if (!window.confirm("Supprimer ce message ?")) return;

//   try {
//     // Make sure your backend route matches: DELETE /messages/:id
//     await axios.delete(`${apiUrl}/messages/${messageId}`, authHeaders);

//     // Update the UI immediately by filtering out the message
//     setMessages((prevMessages) => 
//       prevMessages.filter((msg) => msg.id !== messageId)
//     );

//     return true;
//   } catch (err) {
//     console.error("Delete Message Error:", err.response?.data?.error || err.message);
//     alert("Impossible de supprimer ce message.");
//     return false;
//   }
// };


// // NOTIFICATIONS
// // Dans AppContext.jsx

// // 1. Récupérer les notifications
// const fetchNotifications = useCallback(async () => {
//   try {
//     // Note le port 8083 pour le service notification
//     const res = await axios.get(`http://localhost:8083/api/v1/notifications`, authHeaders);
//     setNotifications(res.data || []);
//   } catch (err) {
//     console.error("Error fetching notifications:", err.message);
//   }
// }, [authHeaders]);

// // 2. Marquer comme lu (pour vider le compteur)
// const markNotificationsAsRead = async (chatId) => {
//   try {
//     await axios.post(`http://localhost:8083/api/v1/notifications/read`, { chat_id: chatId }, authHeaders);
//     // On filtre localement pour mettre à jour l'UI instantanément
//     setNotifications(prev => prev.filter(n => n.chat_id !== chatId));
//   } catch (err) {
//     console.error("Error marking notifications as read:", err.message);
//   }
// };

// // N'oublie pas d'ajouter fetchNotifications dans un useEffect de chargement initial
// useEffect(() => {
//     if (token) {
//         fetchNotifications();
//         // Optionnel : Polling pour les notifs toutes les 10 secondes
//         const interval = setInterval(fetchNotifications, 10000);
//         return () => clearInterval(interval);
//     }
// }, [token, fetchNotifications]);





//   useEffect(() => {
//     fetchChats();
//     const interval = setInterval(fetchChats, 10000);
//     return () => clearInterval(interval);
//   }, [fetchChats]);




// useEffect(() => {
//   let interval;

//   if (selectedChat) {
//     // 1. Chargement immédiat avec spinner
//     fetchMessages(selectedChat.id, true);

//     // 2. Vérification toutes les 2 secondes en arrière-plan (sans spinner)
//     interval = setInterval(() => {
//       fetchMessages(selectedChat.id, false);
//     }, 2000); 
//   } else {
//     setMessages([]);
//   }

//   // NETTOYAGE : Indispensable pour arrêter de consommer du réseau 
//   // quand on change de chat ou qu'on quitte la page
//   return () => {
//     if (interval) clearInterval(interval);
//   };
// }, [selectedChat, fetchMessages]);

//   return (
//     <AppContext.Provider value={{
//       chats, selectedChat, setSelectedChat, 
//       messages, loading, currentUserId,
//       sendMessage, createChat, fetchChats,
//       deleteMessage, fetchAllUsers, getUserById,
//       updateUser, deleteUser, users, updateChat, deleteChat,
//       getHistory, notifications,
//     fetchNotifications,
//     markNotificationsAsRead,
     
//     }}>
//       {children}
//     </AppContext.Provider>
//   );
// };

// // Custom hook for easy access
// export const useApp = () => useContext(AppContext);

import React, { createContext, useState, useEffect, useCallback, useMemo, useContext } from "react";
import axios from "axios";

const AppContext = createContext();

export const AppProvider = ({ children }) => {
  const [users, setUsers] = useState([]);
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [notifications, setNotifications] = useState([]);

  const currentUserId = localStorage.getItem("user_id");
  const apiUrl = "http://localhost:8080/api/v1";
  const userApiUrl = "http://localhost:8081/api/v1";

  // Memoized Headers
  // const authHeaders = useMemo(() => {
  //   const token = localStorage.getItem("token");
  //   const userId = localStorage.getItem("user_id");
  //   return {
  //     headers: {
  //       Authorization: `Bearer ${token}`,
  //       "Content-Type": "application/json",
  //       "X-User-ID": userId,
  //     },
  //   };
  // }, []);

  // Remplace ton bloc authHeaders par celui-ci :
const authHeaders = useMemo(() => {
  const token = localStorage.getItem("token");
  const userId = localStorage.getItem("user_id");
  
  return {
    headers: {
      // On ajoute un check pour éviter d'envoyer "Bearer null"
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json",
      "X-User-ID": userId || "",
    },
  };
  // On ajoute 'chats' ou une autre variable qui change après login 
  // pour forcer React à rafraîchir les headers
}, [chats.length]);
  // --- USER ACTIONS ---

  const fetchAllUsers = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get(`${userApiUrl}/users`, authHeaders);
      const data = res.data.users || res.data.data || res.data;
      if (Array.isArray(data)) {
        setUsers(data);
      } else {
        setUsers([]);
      }
    } catch (err) {
      console.error("Fetch Users Error:", err.response?.data || err.message);
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }, [authHeaders, userApiUrl]);

  const getUserById = async (userId) => {
    try {
      const res = await axios.get(`${userApiUrl}/users/${userId}`, authHeaders);
      return res.data;
    } catch (err) {
      console.error("Get User Error:", err.message);
    }
  };

  const updateUser = async (userId, updateData) => {
    try {
      await axios.put(`${userApiUrl}/users/${userId}`, updateData, authHeaders);
      setUsers((prev) => prev.map((u) => (u.id === userId ? { ...u, ...updateData } : u)));
      return true;
    } catch (err) {
      console.error("Update User Error:", err.message);
      return false;
    }
  };

  const deleteUser = async (userId) => {
    if (!window.confirm("Supprimer cet utilisateur définitivement ?")) return;
    try {
      await axios.delete(`${userApiUrl}/users/${userId}`, authHeaders);
      setUsers((prev) => prev.filter((u) => u.id !== userId));
      return true;
    } catch (err) {
      console.error("Delete User Error:", err.message);
      return false;
    }
  };

  // --- CHAT ACTIONS ---

  const createChat = async (newChatData) => {
    let participantsArray = [];
    if (typeof newChatData.participants === "string") {
      participantsArray = newChatData.participants
        .split(",")
        .map((id) => id.trim())
        .filter((id) => id.length > 10);
    } else if (Array.isArray(newChatData.participants)) {
      participantsArray = newChatData.participants;
    }

    const backendType = newChatData.type === "groupe" ? "group" : newChatData.type;
    const payload = {
      participants: participantsArray,
      type: backendType,
      name: backendType === "group" ? (newChatData.name || "Nouveau Groupe") : "",
    };

    try {
      const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);
      const newChat = response.data.chat || response.data;
      setChats((prev) => [newChat, ...prev]);
      setSelectedChat(newChat);
      return newChat;
    } catch (err) {
      const errorMsg = err.response?.data?.error || "Erreur lors de la création";
      throw new Error(errorMsg);
    }
  };

  const fetchChats = useCallback(async () => {
    if (!authHeaders.headers.Authorization) return;
    try {
      const res = await axios.get(`${apiUrl}/chats`, authHeaders);
      setChats(res.data || []);
    } catch (err) {
      console.error("Fetch Chats Error:", err.message);
    } finally {
      setLoading(false);
    }
  }, [authHeaders, apiUrl]);

  const updateChat = async (chatId, updateData) => {
    try {
      const res = await axios.put(`${apiUrl}/chats/${chatId}`, updateData, authHeaders);
      const updatedChat = res.data.chat || res.data;
      setChats((prev) => prev.map((c) => (c.id === chatId ? { ...c, ...updatedChat } : c)));
      if (selectedChat?.id === chatId) {
        setSelectedChat((prev) => ({ ...prev, ...updatedChat }));
      }
      return true;
    } catch (err) {
      console.error("Update Chat Error:", err.response?.data || err.message);
      return false;
    }
  };

  const deleteChat = async (chatId) => {
    if (!window.confirm("Supprimer cette discussion ?")) return;
    try {
      await axios.delete(`${apiUrl}/chats/${chatId}`, authHeaders);
      setChats((prev) => prev.filter((c) => c.id !== chatId));
      if (selectedChat?.id === chatId) setSelectedChat(null);
      return true;
    } catch (err) {
      console.error("Delete Chat Error:", err.message);
      return false;
    }
  };

  // --- MESSAGE ACTIONS ---

  const fetchMessages = useCallback(async (chatId, showSpinner = true) => {
    if (!chatId) return;
    if (showSpinner) setLoading(true);
    try {
      const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
      let data = res.data.messages || res.data || [];
      if (!Array.isArray(data)) data = [];
      const formattedData = [...data].reverse();
      setMessages((prev) => (JSON.stringify(prev) === JSON.stringify(formattedData) ? prev : formattedData));
      await fetchChats();
    } catch (err) {
      console.error("Fetch Error:", err.message);
    } finally {
      setLoading(false);
    }
  }, [authHeaders, fetchChats, apiUrl]);

  const sendMessage = async (content) => {
    if (!content.trim() || !selectedChat) return;
    const payload = { chat_id: selectedChat.id, content: content };
    try {
      const res = await axios.post(`${apiUrl}/messages`, payload, authHeaders);
      const sentMsg = res.data.message || res.data;
      setMessages((prev) => [...prev, { 
        ...sentMsg, 
        sender_id: currentUserId, 
        created_at: sentMsg.created_at || new Date().toISOString() 
      }]);
      fetchChats();
      return true;
    } catch (err) {
      console.error("Send Error:", err.message);
      return false;
    }
  };

  const deleteMessage = async (messageId) => {
    if (!window.confirm("Supprimer ce message ?")) return;
    try {
      await axios.delete(`${apiUrl}/messages/${messageId}`, authHeaders);
      setMessages((prev) => prev.filter((msg) => msg.id !== messageId));
      return true;
    } catch (err) {
      console.error("Delete Message Error:", err.message);
      return false;
    }
  };

  const getHistory = useCallback(async (chatId) => {
    if (!chatId) return [];
    setLoading(true);
    try {
      const res = await axios.get(`http://localhost:8082/api/v1/messages/${chatId}`, authHeaders);
      const data = res.data.messages || res.data.data || res.data;
      const historyMessages = Array.isArray(data) ? data : [];
      setMessages(historyMessages);
      return historyMessages;
    } catch (err) {
      console.error("Get History Error:", err.message);
      return [];
    } finally {
      setLoading(false);
    }
  }, [authHeaders]);

  // --- NOTIFICATIONS ---

 // 1. Récupérer les notifications
const fetchNotifications = useCallback(async () => {
  // On vérifie que currentUserId existe avant de lancer l'appel
  if (!currentUserId) return;

  try {
    // Utilisation de apiUrl (Port 8080) + /notification (singulier) + /user/ID
    const res = await axios.get(`${apiUrl}/notification/user/${currentUserId}`, authHeaders);
    
    // Ton Backend Go renvoie : { "notifications": [...], "user_id": "...", "count": X }
    // On extrait donc spécifiquement le tableau "notifications"
    setNotifications(res.data.notifications || []);
  } catch (err) {
    console.error("Error fetching notifications:", err.message);
  }
}, [authHeaders, apiUrl, currentUserId]); // Ajout des dépendances manquantes


// 2. Marquer comme lu
const markNotificationsAsRead = async (chatId) => {
  try {
    // Note 1 : Utilisation de apiUrl pour passer par la Gateway
    // Note 2 : Ton backend Go a une route PUT /user/:userId/read-all 
    // ou PUT /:id/read. Ici on utilise la version globale pour l'utilisateur :
    await axios.put(`${apiUrl}/notification/user/${currentUserId}/read-all`, {}, authHeaders);
    
    // Mise à jour locale de l'UI
    setNotifications((prev) => prev.filter((n) => n.chat_id !== chatId));
  } catch (err) {
    console.error("Error marking notifications as read:", err.message);
  }
};
  // --- EFFECTS ---

  useEffect(() => {
    fetchAllUsers();
  }, [fetchAllUsers]);

  useEffect(() => {
    fetchChats();
    const interval = setInterval(fetchChats, 10000);
    return () => clearInterval(interval);
  }, [fetchChats]);

  useEffect(() => {
    const token = localStorage.getItem("token"); // FIX: Define token inside the effect
    if (token) {
      fetchNotifications();
      const interval = setInterval(fetchNotifications, 10000);
      return () => clearInterval(interval);
    }
  }, [fetchNotifications]);

  useEffect(() => {
    let interval;
    if (selectedChat) {
      fetchMessages(selectedChat.id, true);
      interval = setInterval(() => {
        fetchMessages(selectedChat.id, false);
      }, 2000);
    } else {
      setMessages([]);
    }
    return () => { if (interval) clearInterval(interval); };
  }, [selectedChat, fetchMessages]);

  return (
    <AppContext.Provider
      value={{
        chats, selectedChat, setSelectedChat, messages, loading, currentUserId,
        sendMessage, createChat, fetchChats, deleteMessage, fetchAllUsers,
        getUserById, updateUser, deleteUser, users, updateChat, deleteChat,
        getHistory, notifications, fetchNotifications, markNotificationsAsRead,
      }}
    >
      {children}
    </AppContext.Provider>
  );
};

export const useApp = () => useContext(AppContext);