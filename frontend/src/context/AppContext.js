import React, { createContext, useState, useEffect, useCallback, useMemo, useContext } from "react";
import axios from "axios";

const AppContext = createContext();

export const AppProvider = ({ children }) => {
  const [users, setUsers] = useState([]); // NEW: State for contacts
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  
  const currentUserId = localStorage.getItem("user_id");
  const apiUrl = "http://localhost:8080/api/v1";
  const userApiUrl = "http://localhost:8081/api/v1"; // NEW: User service URL

  // Memoized Headers
  const authHeaders = useMemo(() => {
    const token = localStorage.getItem("token");
    const userId = localStorage.getItem("user_id");
    return {
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
        "X-User-ID": userId
      },
    };
  }, []);


  // --- USER ACTIONS ---

  // 1. Get all users
const fetchAllUsers = useCallback(async () => {
  setLoading(true);
  try {
    // 1. Double check this URL in your browser!
    const res = await axios.get(`http://localhost:8081/api/v1/users`, authHeaders);
    
    console.log("DEBUG: Raw Response Data ->", res.data);

    // 2. Flexible extraction: handle different backend formats
    const data = res.data.users || res.data.data || res.data;

    if (Array.isArray(data)) {
      setUsers(data);
    } else {
      console.error("API returned success but no array found. Check console log above.");
      setUsers([]);
    }
  } catch (err) {
    console.error("Fetch Users Error:", err.response?.data || err.message);
    setUsers([]); // Clear state on error
  } finally {
    setLoading(false);
  }
}, [authHeaders]);



  // 2. Get User By ID
  const getUserById = async (userId) => {
    try {
      const res = await axios.get(`${userApiUrl}/users/${userId}`, authHeaders);
      return res.data;
    } catch (err) {
      console.error("Get User Error:", err.message);
    }
  };

  // 3. Update User
  const updateUser = async (userId, updateData) => {
    try {
      const res = await axios.put(`${userApiUrl}/users/${userId}`, updateData, authHeaders);
      // Update local state so UI reflects change immediately
      setUsers(prev => prev.map(u => u.id === userId ? { ...u, ...updateData } : u));
      return true;
    } catch (err) {
      console.error("Update User Error:", err.message);
      return false;
    }
  };

  // 4. Delete User
  const deleteUser = async (userId) => {
    if (!window.confirm("Supprimer cet utilisateur définitivement ?")) return;
    try {
      await axios.delete(`${userApiUrl}/users/${userId}`, authHeaders);
      setUsers(prev => prev.filter(u => u.id !== userId));
      return true;
    } catch (err) {
      console.error("Delete User Error:", err.message);
      return false;
    }
  };


const createChat = async (newChatData) => {
  let participantsArray = [];

  // Vérifie si participants est une string ou déjà un tableau
  if (typeof newChatData.participants === "string") {
    participantsArray = newChatData.participants
      .split(",")
      .map((id) => id.trim())
      .filter((id) => id.length > 10);
  } else if (Array.isArray(newChatData.participants)) {
    participantsArray = newChatData.participants;
  }

  // Correction du type pour Go
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
    console.error("Détails Erreur Backend:", err.response?.data);
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
  }, [authHeaders]);


  // 1. Update Chat (Usually for renaming a group)
const updateChat = async (chatId, updateData) => {
  try {
    // Go Backend expects: PUT /api/v1/chats/:id
    const res = await axios.put(`${apiUrl}/chats/${chatId}`, updateData, authHeaders);
    const updatedChat = res.data.chat || res.data;

    setChats((prev) =>
      prev.map((c) => (c.id === chatId ? { ...c, ...updatedChat } : c))
    );
    
    if (selectedChat?.id === chatId) {
      setSelectedChat(prev => ({ ...prev, ...updatedChat }));
    }
    return true;
  } catch (err) {
    console.error("Update Chat Error:", err.response?.data || err.message);
    return false;
  }
};

// 2. Delete Chat
const deleteChat = async (chatId) => {
  if (!window.confirm("Supprimer cette discussion ? Cette action est irréversible.")) return;

  try {
    // Go Backend expects: DELETE /api/v1/chats/:id
    await axios.delete(`${apiUrl}/chats/${chatId}`, authHeaders);

    // Remove from local state
    setChats((prev) => prev.filter((c) => c.id !== chatId));
    
    // Clear selection if the deleted chat was open
    if (selectedChat?.id === chatId) {
      setSelectedChat(null);
    }
    return true;
  } catch (err) {
    console.error("Delete Chat Error:", err.response?.data || err.message);
    alert("Erreur lors de la suppression du chat.");
    return false;
  }
};


  useEffect(() => {
  fetchAllUsers();
}, [fetchAllUsers]);

  // --- ACTIONS ---

  
 const fetchMessages = useCallback(async (chatId) => {
  if (!chatId) return;
  setLoading(true); // Optional: show a small spinner
  try {
    const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
    const data = res.data.messages || res.data || [];
    setMessages(Array.isArray(data) ? data : []);

    // FIX: Also refresh the chat list so the sidebar knows there are new messages
    // This ensures the "updated_at" timestamp moves the chat to the top
    await fetchChats(); 
    
  } catch (err) {
    console.error("Fetch Messages Error:", err.message);
  } finally {
    setLoading(false);
  }
}, [authHeaders, fetchChats]); // Make sure fetchChats is in the dependency array

  const sendMessage = async (content) => {
    if (!content.trim() || !selectedChat) return;
    const payload = { chat_id: selectedChat.id, content };
    try {
      const res = await axios.post(`${apiUrl}/messages`, payload, authHeaders);
      const sentMsg = res.data.message || res.data;
      setMessages((prev) => [...prev, { ...sentMsg, sender_id: currentUserId }]);
      return true;
    } catch (err) {
      console.error("Send Error:", err.message);
      return false;
    }
  };

  const deleteMessage = async (messageId) => {
  // Optional: Add a confirmation dialog
  if (!window.confirm("Supprimer ce message ?")) return;

  try {
    // Make sure your backend route matches: DELETE /messages/:id
    await axios.delete(`${apiUrl}/messages/${messageId}`, authHeaders);

    // Update the UI immediately by filtering out the message
    setMessages((prevMessages) => 
      prevMessages.filter((msg) => msg.id !== messageId)
    );

    return true;
  } catch (err) {
    console.error("Delete Message Error:", err.response?.data?.error || err.message);
    alert("Impossible de supprimer ce message.");
    return false;
  }
};








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

  return (
    <AppContext.Provider value={{
      chats, selectedChat, setSelectedChat, 
      messages, loading, currentUserId,
      sendMessage, createChat, fetchChats,
      deleteMessage, fetchAllUsers, getUserById,
      updateUser, deleteUser, users, updateChat, deleteChat
     
    }}>
      {children}
    </AppContext.Provider>
  );
};

// Custom hook for easy access
export const useApp = () => useContext(AppContext);