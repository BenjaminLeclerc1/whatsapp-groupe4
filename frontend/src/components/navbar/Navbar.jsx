import React, { useState, useEffect, useRef } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import '../styles/navbar.css';

const Navbar = () => {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isMobileOpen, setIsMobileOpen] = useState(false);
  const [userName, setUserName] = useState('');
  const dropdownRef = useRef(null);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const token = localStorage.getItem('token');
    setIsLoggedIn(!!token);
    setUserName(localStorage.getItem('username') || '');
  }, [location]);

  useEffect(() => {
    const handleClickOutside = (e) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user_id');
    localStorage.removeItem('username');
    setIsLoggedIn(false);
    setIsDropdownOpen(false);
    navigate('/login');
  };

  const isActive = (path) => location.pathname === path;

  return (
    <nav className="g4-navbar">
      <div className="g4-navbar-inner">
        <Link to="/" className="g4-logo">
          <div className="g4-logo-icon">G4</div>
          <span className="g4-logo-text">Groupe 4</span>
        </Link>

        {isLoggedIn && (
          <div className={`g4-nav-links ${isMobileOpen ? 'open' : ''}`}>
            <Link
              to="/chats"
              className={`g4-nav-link ${isActive('/chats') ? 'active' : ''}`}
              onClick={() => setIsMobileOpen(false)}
            >
              <span>💬</span>
              <span>Discussions</span>
            </Link>
            <Link
              to="/contacts"
              className={`g4-nav-link ${isActive('/contacts') ? 'active' : ''}`}
              onClick={() => setIsMobileOpen(false)}
            >
              <span>👥</span>
              <span>Contacts</span>
            </Link>
          </div>
        )}

        <div className="g4-nav-right">
          {!isLoggedIn ? (
            <Link to="/login" className="g4-login-btn">Se connecter</Link>
          ) : (
            <div className="g4-profile-wrap" ref={dropdownRef}>
              <button
                className="g4-profile-trigger"
                onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              >
                <img
                  src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${userName || 'default'}`}
                  alt="avatar"
                  className="g4-avatar"
                />
                <span className="g4-username">{userName || 'Profil'}</span>
                <span className="g4-chevron">
                  {isDropdownOpen ? '▲' : '▼'}
                </span>
              </button>

              {isDropdownOpen && (
                <div className="g4-dropdown">
                  <div className="g4-dropdown-header">
                    <img
                      src={`https://api.dicebear.com/9.x/adventurer/svg?seed=${userName || 'default'}`}
                      alt="avatar"
                      className="g4-dropdown-avatar"
                    />
                    <div>
                      <p className="g4-dropdown-name">{userName || 'Utilisateur'}</p>
                      <p className="g4-dropdown-id">
                        ID: {localStorage.getItem('user_id')?.substring(0, 8)}...
                      </p>
                    </div>
                  </div>

                  <div className="g4-dropdown-divider" />

                  <button className="g4-dropdown-item" onClick={() => { navigate('/profile'); setIsDropdownOpen(false); }}>
                    Mon profil
                  </button>

                  <div className="g4-dropdown-divider" />

                  <button className="g4-dropdown-item g4-logout" onClick={handleLogout}>
                    Déconnexion
                  </button>
                </div>
              )}
            </div>
          )}

          {isLoggedIn && (
            <button className="g4-mobile-toggle" onClick={() => setIsMobileOpen(!isMobileOpen)}>
              {isMobileOpen ? '✕' : '☰'}
            </button>
          )}
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
