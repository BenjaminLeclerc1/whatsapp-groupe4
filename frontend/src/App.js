import { Route, Routes, Navigate } from 'react-router-dom';
import Login from './components/login/Login';
import Register from './components/register/Register';
import Chats from './components/chats/Chat';
import Navbar from './components/navbar/Navbar';

// 🛡️ Helper: If no token, kick them to login
const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? children : <Navigate to="/login" replace />;
};

// 🛡️ Helper: If already logged in, don't show login/register pages
const PublicRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  const hasToken = token && token !== "null" && token !== "undefined";
  return hasToken ? <Navigate to="/chats" replace /> : children;
};

function App() {
  return (
    <div className="app-container">
      {/* Navbar is now always visible on every page */}
      <Navbar />

      <main className="main-content">
        <Routes>
          {/* 1. Root Path now shows Login directly (restricted to public) */}
          <Route path="/" element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          } />

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
          <Route path="/chats" element={
            <ProtectedRoute>
              <Chats />
            </ProtectedRoute>
          } />

          {/* 4. Catch-all: Redirect unknown routes to home */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;


// import { Route, Routes, Navigate, useLocation } from 'react-router-dom';
// import Login from './components/login/Login';
// import Register from './components/register/Register';
// import Chats from './components/chats/Chat';
// import Navbar from './components/navbar/Navbar'; // Import your new Navbar

// // 🛡️ Helper: If no token, kick them to login
// const ProtectedRoute = ({ children }) => {
//   const token = localStorage.getItem('token');
//   const hasToken = token && token !== "null" && token !== "undefined";
//   return hasToken ? children : <Navigate to="/login" replace />;
// };

// // 🛡️ Helper: If already logged in, don't show login page
// const PublicRoute = ({ children }) => {
//   const token = localStorage.getItem('token');
//   const hasToken = token && token !== "null" && token !== "undefined";
//   return hasToken ? <Navigate to="/chats" replace /> : children;
// };

// function App() {
//   const location = useLocation();
  
//   // Optional: Hide Navbar on Login and Register pages
//   const hideNavbarPaths = ['/login', '/register'];
//   const shouldShowNavbar = !hideNavbarPaths.includes(location.pathname);

//   return (
//     <div className="app-container">
//       {/* Show Navbar only if not on login/register */}
//       {shouldShowNavbar && <Navbar />}

//       <Routes>
//         {/* 1. Default Home Page */}
//         <Route path="/" element={<Navigate to="/login" replace />} />

//         {/* 2. Login & Register (Public only) */}
//         <Route path="/login" element={
//           <PublicRoute>
//             <Login />
//           </PublicRoute>
//         } />
        
//         <Route path="/register" element={
//           <PublicRoute>
//             <Register />
//           </PublicRoute>
//         } />

//         {/* 3. Chat Page (Protected) */}
//         <Route path="/chats" element={
//           <ProtectedRoute>
//             <Chats />
//           </ProtectedRoute>
//         } />

//         {/* 4. Catch-all */}
//         <Route path="*" element={<Navigate to="/" replace />} />
//       </Routes>
//     </div>
//   );
// }

// export default App;