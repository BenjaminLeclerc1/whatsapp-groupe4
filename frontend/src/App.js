import { Route, Routes, Navigate } from 'react-router-dom';
import Login from './components/login/Login';
import Register from './components/register/Register';
import Chat from './components/chat/Chat';

// 🛡️ Helper: If no token, kick them to login
// 🛡️ Helper: If no token, kick them to login
const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  // Check if token exists AND is not the string "null" or "undefined"
  const hasToken = token && token !== "null" && token !== "undefined";
  
  return hasToken ? children : <Navigate to="/login" replace />;
};

// 🛡️ Helper: If already logged in, don't show login page (send to chat)
const PublicRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  
  return hasToken ? <Navigate to="/chat" replace /> : children;
};

function App() {
  return (
    <Routes>
      {/* 1. Default Home Page (Redirects to login) */}
      <Route path="/" element={<Navigate to="/login" replace />} />

      {/* 2. Login & Register (Public only) */}
      <Route path="/login" element={
        <PublicRoute>
          <Login />
        </PublicRoute>
      } />
      <Route path="/register" element={
        <PublicRoute>
          <Register />
        </PublicRoute>
      } />

      {/* 3. Chat Page (Protected) */}
      <Route path="/chat" element={
        <ProtectedRoute>
          <Chat />
        </ProtectedRoute>
      } />

      {/* 4. Catch-all: Send unknown URLs back to login or chat */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;