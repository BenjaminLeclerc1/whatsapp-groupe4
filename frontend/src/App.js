// import { Route, Routes, Navigate } from 'react-router-dom';
// import Login from './pages/login/Login';
// import Register from './pages/register/Register';
// import Chats from './pages/chats/Chat';
// import Navbar from './components/navbar/Navbar';
// import { AppProvider } from './context/AppContext';

// // 🛡️ Helper: If no token, kick them to login
// const ProtectedRoute = ({ children }) => {
//   const token = localStorage.getItem('token');
//   const hasToken = token && token !== "null" && token !== "undefined";
//   return hasToken ? children : <Navigate to="/login" replace />;
// };

// // 🛡️ Helper: If already logged in, don't show login/register pages
// const PublicRoute = ({ children }) => {
//   const token = localStorage.getItem('token');
//   const hasToken = token && token !== "null" && token !== "undefined";
//   return hasToken ? <Navigate to="/chats" replace /> : children;
// };

// function App() {
//   return (
//     <div className="app-container">
//       {/* Navbar is now always visible on every page */}
//       <Navbar />

//       <main className="main-content">
//         <Routes>
//           {/* 1. Root Path now shows Login directly (restricted to public) */}
//           <Route path="/" element={
//             <PublicRoute>
//               <Login />
//             </PublicRoute>
//           } />

//           {/* 2. Login & Register (Public only) */}
//           <Route path="/login" element={
//             <PublicRoute>
//               <Login />
//             </PublicRoute>
//           } />
          
//           <Route path="/register" element={
//             <PublicRoute>
//               <Register />
//             </PublicRoute>
//           } />

//           {/* 3. Chat Page (Protected) */}
//           <Route path="/chats" element={
//             <ProtectedRoute>
//               <AppProvider>

//               <Chats />
              
//               </AppProvider>
//             </ProtectedRoute>
//           } />

//           <Route path="/contacts" element={
//             <ProtectedRoute>
//               <AppProvider>

//               <Contacts />
              
//               </AppProvider>
//             </ProtectedRoute>
//           } />

//           {/* 4. Catch-all: Redirect unknown routes to home */}
//           <Route path="*" element={<Navigate to="/" replace />} />
//         </Routes>
//       </main>
//     </div>
//   );
// }

// export default App;




import { Route, Routes, Navigate } from 'react-router-dom';
import Login from './pages/login/Login';
import Register from './pages/register/Register';
import Chats from './pages/chats/Chat';
import Navbar from './components/navbar/Navbar';
import { AppProvider } from './context/AppContext';
import Contacts from './pages/contacts/Contacts';

// --- AJOUTE CES DEUX FONCTIONS ICI (Elles manquaient) ---

// 🛡️ Helper: Si pas de token, redirection vers login
const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? children : <Navigate to="/login" replace />;
};

// 🛡️ Helper: Si déjà connecté, redirection vers /chats
const PublicRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? <Navigate to="/chats" replace /> : children;
};

// --------------------------------------------------------

function App() {
  return (
    <AppProvider>
      <div className="app-container">
        <Navbar />

        <main className="main-content">
          <Routes>
            <Route path="/" element={
              <PublicRoute>
                <Login />
              </PublicRoute>
            } />

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

            <Route path="/chats" element={
              <ProtectedRoute>
                <Chats />
              </ProtectedRoute>
            } />

            <Route path="/contacts" element={
              <ProtectedRoute>
                <Contacts />
              </ProtectedRoute>
            } />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>
      </div>
    </AppProvider>
  );
}

export default App;