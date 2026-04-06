import axios from 'axios';

const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL_AUTH || "http://localhost:8081/api/v1",
});

// This automatically attaches the token to EVERY request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
}, (error) => {
  return Promise.reject(error);
});

export default api;