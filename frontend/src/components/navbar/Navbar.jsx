// import React, { useState, useEffect } from 'react';
// import { Link, useNavigate } from 'react-router-dom';
// import '../styles/navbar.css';

// const Navbar = () => {
//   const [isMenuOpen, setIsMenuOpen] = useState(false);
//   const [isDropdownOpen, setIsDropdownOpen] = useState(false);
//   const [isLoggedIn, setIsLoggedIn] = useState(false);
//   const navigate = useNavigate();

//   // Check auth state on mount and when localStorage changes
//   useEffect(() => {
//     const token = localStorage.getItem('token');
//     setIsLoggedIn(!!token);
//   }, []);

//   const handleLogout = () => {
//     localStorage.removeItem('token');
//     localStorage.removeItem('user_id');
//     setIsLoggedIn(false);
//     setIsDropdownOpen(false);
//     navigate('/login');
//   };

//   return (
//     <nav className="navbar">
//       <div className="navbar-container">
//         {/* Logo */}
//         <Link to="/" className="navbar-logo">
//           <span className="logo-icon">💬</span>
//           <span className="logo-text">Groupe 4</span>
//         </Link>

//         {/* Desktop Menu */}
//         <div className={`nav-elements ${isMenuOpen ? 'active' : ''}`}>
//           <ul className="nav-links">
//             <li><Link to="/chats">Chats</Link></li>
//             <li><Link to="/contacts">Contacts</Link></li>
//           </ul>

//           <div className="nav-auth">
//             {!isLoggedIn ? (
//               <Link to="/login" className="login-btn">Login</Link>
//             ) : (
//               <div className="profile-container">
//                 <button 
//                   className="profile-trigger"
//                   onClick={() => setIsDropdownOpen(!isDropdownOpen)}
//                 >
//                   <img 
//                     src={`https://ui-avatars.com/api/?name=User&background=00a884&color=fff`} 
//                     alt="Profile" 
//                     className="avatar"
//                   />
//                 </button>

//                 {isDropdownOpen && (
//                   <div className="dropdown-menu">
//                     <div className="dropdown-header">
//                       <p className="user-name">Mon Profil</p>
//                       <p className="user-id">ID: {localStorage.getItem('user_id')?.substring(0, 8)}...</p>
//                     </div>
//                     <hr />
//                     <button className="dropdown-item">Paramètres</button>
//                     <button className="dropdown-item logout-action" onClick={handleLogout}>
//                       Déconnexion
//                     </button>
//                   </div>
//                 )}
//               </div>
//             )}
//           </div>
//         </div>

//         {/* Mobile Toggle */}
//         <button className="mobile-menu-icon" onClick={() => setIsMenuOpen(!isMenuOpen)}>
//           {isMenuOpen ? '✕' : '☰'}
//         </button>
//       </div>
//     </nav>
//   );
// };

// export default Navbar;


import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useApp } from '../../context/AppContext';
import '../styles/navbar.css';

const Navbar = () => {
  const { updateUser, deleteUser } = useApp();

  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  
  // Utilisation de la clé 'username' et valeur par défaut vide ou 'Username'
  const [userName, setUserName] = useState(localStorage.getItem('username') || '');
  
  const navigate = useNavigate();

  useEffect(() => {
    const token = localStorage.getItem('token');
    setIsLoggedIn(!!token);
    
    // Récupération avec la clé correcte 'username'
    const storedName = localStorage.getItem('username');
    if (storedName) setUserName(storedName);
  }, []);

  const handleUpdateClick = async () => {
    const userId = localStorage.getItem('user_id');
    const newName = prompt("Entrez votre nouveau nom :", userName);
    if (!newName || newName === userName) return;

    const success = await updateUser(userId, { username: newName });
    if (success) {
      // Mise à jour avec la clé 'username'
      localStorage.setItem('username', newName);
      setUserName(newName);
      alert("Profil mis à jour !");
    }
  };

  const handleDeleteClick = async () => {
    const userId = localStorage.getItem('user_id');
    const success = await deleteUser(userId);
    if (success) {
      handleLogout();
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user_id');
    localStorage.removeItem('username'); // Nettoyage de la bonne clé
    setIsLoggedIn(false);
    setIsDropdownOpen(false);
    navigate('/login');
  };

  return (
    <nav className="navbar">
      <div className="navbar-container">
        <Link to="/" className="navbar-logo">
          <span className="logo-icon">💬</span>
          <span className="logo-text">Groupe 4</span>
        </Link>

        <div className={`nav-elements ${isMenuOpen ? 'active' : ''}`}>
          <ul className="nav-links">
            <li><Link to="/chats">Chats</Link></li>
            <li><Link to="/contacts">Contacts</Link></li>
          </ul>

          <div className="nav-auth">
            {!isLoggedIn ? (
              <Link to="/login" className="login-btn">Login</Link>
            ) : (
              <div className="profile-container">
                <button 
                  className="profile-trigger"
                  onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                >
                  <div className="avatar-wrapper">
                    <img
                      src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${userName || 'default'}`}
                      alt="avatar"
                      className="avatar"
                    />
                    {/* Affiche le username dynamique */}
                    <span className="avatar-name">{userName || 'Mon Profil'}</span>
                  </div>
                </button>

                {isDropdownOpen && (
                  <div className="dropdown-menu">
                    <div className="dropdown-header">
                      <p className="user-name">{userName || 'Utilisateur'}</p>
                      <p className="user-id">ID: {localStorage.getItem('user_id')?.substring(0, 8)}...</p>
                    </div>
                    <hr />
                    
                    <button 
                      className="dropdown-item" 
                      onClick={() => setIsSettingsOpen(!isSettingsOpen)}
                    >
                      {isSettingsOpen ? '▼ Paramètres' : '▶ Paramètres'}
                    </button>

                    {isSettingsOpen && (
                      <div className="settings-submenu">
                        <button className="submenu-item" onClick={handleUpdateClick}>Modifier le profil</button>
                        <button className="submenu-item delete-text" onClick={handleDeleteClick}>Supprimer mon compte</button>
                      </div>
                    )}

                    <button className="dropdown-item logout-action" onClick={handleLogout}>
                      Déconnexion
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        <button className="mobile-menu-icon" onClick={() => setIsMenuOpen(!isMenuOpen)}>
          {isMenuOpen ? '✕' : '☰'}
        </button>
      </div>
    </nav>
  );
};

export default Navbar;