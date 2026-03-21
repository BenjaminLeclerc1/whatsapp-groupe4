import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import '../styles/navbar.css';

const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const navigate = useNavigate();

  // Check auth state on mount and when localStorage changes
  useEffect(() => {
    const token = localStorage.getItem('token');
    setIsLoggedIn(!!token);
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user_id');
    setIsLoggedIn(false);
    setIsDropdownOpen(false);
    navigate('/login');
  };

  return (
    <nav className="navbar">
      <div className="navbar-container">
        {/* Logo */}
        <Link to="/" className="navbar-logo">
          <span className="logo-icon">💬</span>
          <span className="logo-text">Groupe 4</span>
        </Link>

        {/* Desktop Menu */}
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
                  <img 
                    src={`https://ui-avatars.com/api/?name=User&background=00a884&color=fff`} 
                    alt="Profile" 
                    className="avatar"
                  />
                </button>

                {isDropdownOpen && (
                  <div className="dropdown-menu">
                    <div className="dropdown-header">
                      <p className="user-name">Mon Profil</p>
                      <p className="user-id">ID: {localStorage.getItem('user_id')?.substring(0, 8)}...</p>
                    </div>
                    <hr />
                    <button className="dropdown-item">Paramètres</button>
                    <button className="dropdown-item logout-action" onClick={handleLogout}>
                      Déconnexion
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Mobile Toggle */}
        <button className="mobile-menu-icon" onClick={() => setIsMenuOpen(!isMenuOpen)}>
          {isMenuOpen ? '✕' : '☰'}
        </button>
      </div>
    </nav>
  );
};

export default Navbar;