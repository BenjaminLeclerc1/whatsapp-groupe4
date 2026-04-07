import { Route, Routes, Navigate } from 'react-router-dom';
import Login from './pages/login/Login';
import Register from './pages/register/Register';
import Chats from './pages/chats/Chat';
import Contacts from './pages/contacts/Contacts';
import Profile from './pages/profile/Profile';
import Navbar from './components/navbar/Navbar';
import { AppProvider } from './context/AppContext';

const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? children : <Navigate to="/login" replace />;
};

const PublicRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? <Navigate to="/chats" replace /> : children;
};

function App() {
  return (
    <AppProvider>
      <div className="app-container">
        <Navbar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<PublicRoute><Login /></PublicRoute>} />
            <Route path="/login" element={<PublicRoute><Login /></PublicRoute>} />
            <Route path="/register" element={<PublicRoute><Register /></PublicRoute>} />
            <Route path="/chats" element={<ProtectedRoute><Chats /></ProtectedRoute>} />
            <Route path="/contacts" element={<ProtectedRoute><Contacts /></ProtectedRoute>} />
            <Route path="/profile" element={<ProtectedRoute><Profile /></ProtectedRoute>} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>
      </div>
    </AppProvider>
  );
}

export default App;
