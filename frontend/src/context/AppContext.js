import React, { createContext, useState, useEffect, useCallback, useMemo } from "react";
import axios from "axios";

export const AppContext = createContext();

export const AppProvider = ({ children }) => {

  const apiUrl = "http://localhost:8080/api/v1";

  const [chats, setChats] = useState([]);
  const [messages, setMessages] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [loading, setLoading] = useState(true);

  const currentUserId = localStorage.getItem("user_id");

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

  /*
  ==========================
  FETCH CHATS
  ==========================
  */

  const fetchChats = useCallback(async () => {
    try {
      const res = await axios.get(`${apiUrl}/chats`, authHeaders);
      setChats(res.data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [authHeaders]);

  /*
  ==========================
  FETCH MESSAGES
  ==========================
  */

  const fetchMessages = useCallback(async (chatId) => {
    if (!chatId) return;

    try {
      const res = await axios.get(`${apiUrl}/messages/chat/${chatId}`, authHeaders);
      const data = res.data.messages || res.data || [];
      setMessages(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error(err);
    }
  }, [authHeaders]);

  /*
  ==========================
  SEND MESSAGE
  ==========================
  */

  const sendMessage = async (chatId, content) => {

    const payload = {
      chat_id: chatId,
      content: content
    };

    try {

      const res = await axios.post(`${apiUrl}/messages`, payload, authHeaders);

      const msg = res.data.message || res.data;

      setMessages(prev => [
        ...prev,
        {
          ...msg,
          sender_id: currentUserId
        }
      ]);

    } catch (err) {
      console.error(err);
    }
  };

  /*
  ==========================
  CREATE CHAT
  ==========================
  */

  const createChat = async (chatData) => {

    try {

      const inputIds = chatData.participants
        .split(",")
        .map((id) => id.trim());

      const participantsArray = [
        ...new Set([...inputIds, currentUserId])
      ].filter((id) => id && id.length > 5);

      const payload = {
        participants: participantsArray,
        type: chatData.type,
        name: chatData.type === "group"
          ? chatData.name
          : "Private Chat",
      };

      const response = await axios.post(`${apiUrl}/chats`, payload, authHeaders);

      await fetchChats();

      setSelectedChat(response.data);

    } catch (err) {
      console.error(err);
    }
  };

  /*
  ==========================
  AUTO LOAD
  ==========================
  */

  useEffect(() => {
    fetchChats();

    const interval = setInterval(fetchChats, 10000);

    return () => clearInterval(interval);

  }, [fetchChats]);

  useEffect(() => {

    if (selectedChat) {
      fetchMessages(selectedChat.id);
    }

  }, [selectedChat, fetchMessages]);

  return (
    <AppContext.Provider
      value={{
        chats,
        messages,
        selectedChat,
        setSelectedChat,
        sendMessage,
        createChat,
        loading
      }}
    >
      {children}
    </AppContext.Provider>
  );
};