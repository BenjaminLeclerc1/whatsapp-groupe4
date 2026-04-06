
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
  const apiUrl = process.env.REACT_APP_API_URL || "http://localhost:8080/api/v1";

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
      const res = await axios.get(`${apiUrl}/users`, authHeaders);
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
  }, [authHeaders, apiUrl]);

  const getUserById = async (userId) => {
    try {
      const res = await axios.get(`${apiUrl}/users/${userId}`, authHeaders);
      return res.data;
    } catch (err) {
      console.error("Get User Error:", err.message);
    }
  };

  const searchUsers = useCallback(async (query) => {
    if (!query || !query.trim()) {
      await fetchAllUsers();
      return;
    }
    try {
      const res = await axios.get(`${apiUrl}/users/search?q=${encodeURIComponent(query.trim())}`, authHeaders);
      const data = res.data;
      setUsers(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("Search Users Error:", err.message);
      setUsers([]);
    }
  }, [authHeaders, apiUrl, fetchAllUsers]);

  const updateUser = async (userId, updateData) => {
    try {
      await axios.put(`${apiUrl}/users/${userId}`, updateData, authHeaders);
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
      await axios.delete(`${apiUrl}/users/${userId}`, authHeaders);
      setUsers((prev) => prev.filter((u) => u.id !== userId));
      return true;
    } catch (err) {
      console.error("Delete User Error:", err.message);
      return false;
    }
  };

  // --- CHAT ACTIONS ---

  // const createChat = async (newChatData) => {
  //   let participantsArray = [];
  //   if (typeof newChatData.participants === "string") {
  //     participantsArray = newChatData.participants
  //       .split(",")
  //       .map((id) => id.trim())
  //       .filter((id) => id.length > 10);
  //   } else if (Array.isArray(newChatData.participants)) {
  //     participantsArray = newChatData.participants;
  //   }

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
  //     throw new Error(errorMsg);
  //   }
  // };

const createChat = async (newChatData) => {
  let participantsArray = [];
  
  if (Array.isArray(newChatData.participants)) {
    participantsArray = newChatData.participants;
  } else if (typeof newChatData.participants === "string") {
    participantsArray = [newChatData.participants];
  }

  const payload = {
    participants: participantsArray,
    type: newChatData.type === "groupe" ? "group" : newChatData.type,
    name: newChatData.name || "",
  };

  try {
    const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);
    const newChat = response.data.chat || response.data;
    
    setChats((prev) => [newChat, ...prev]);
    setSelectedChat(newChat); // This tells the app which chat to show
    return newChat;
  } catch (err) {
    throw new Error(err.response?.data?.error || "Erreur serveur");
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
      const res = await axios.get(`${apiUrl}/messages/${chatId}`, authHeaders);
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
        getUserById, updateUser, deleteUser, users, searchUsers, updateChat, deleteChat,
        getHistory, notifications, fetchNotifications, markNotificationsAsRead,
      }}
    >
      {children}
    </AppContext.Provider>
  );
};

export const useApp = () => useContext(AppContext);