import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import Image from 'next/image';

/**
 * Header component with navigation, search, and user controls
 * 
 * @param {Object} props - Component props
 * @param {string} props.title - Page title
 * @param {string} props.subCategory - Page subcategory
 * @param {boolean} props.isAuthenticated - Whether user is authenticated
 * @param {Object} props.user - User data if authenticated
 * @param {Function} props.handleLogout - Function to handle user logout
 * @param {Function} props.toggleTheme - Function to toggle theme
 * @param {string} props.theme - Current theme ('light' or 'dark')
 */
export default function Header({ 
  title, 
  subCategory, 
  isAuthenticated, 
  user, 
  handleLogout, 
  toggleTheme, 
  theme = 'light' 
}) {
  const router = useRouter();
  const [searchTerm, setSearchTerm] = useState('');
  
  // Log component render with authentication state
  useEffect(() => {
    console.log('[HEADER] Header component rendered with:', {
      isAuthenticated,
      userId: user?.id,
      username: user?.username,
      path: router.pathname
    });
  }, [isAuthenticated, user, router.pathname]);
  
  /**
   * Handle search form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSearch = (e) => {
    e.preventDefault();
    if (searchTerm.trim()) {
      const slug = searchTerm.trim()
        .toLowerCase()
        .replace(/[^a-z0-9 -]/g, '')
        .replace(/\s+/g, '-')
        .replace(/-+/g, '-');
      
      router.push(`/search/${slug}`);
    }
  };
  
  return (
    <div className="page-header d-print-none">
      <div className="container-xl">
        <div className="row g-2 align-items-center">
          {/* Title */}
          <div className="col d-none d-lg-block">
            <div className="page-pretitle">
              {subCategory}
            </div>
            <h2 className="page-title">
              {title}
            </h2>
          </div>
          
          {/* Search */}
          <div className="col">
            <div className="my-2 my-md-0 flex-grow-1 flex-md-grow-0 order-first order-md-last d-none d-md-block">
              <form onSubmit={handleSearch} autoComplete="off" noValidate>
                <div className="input-icon">
                  <span className="input-icon-addon">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24"
                         stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <circle cx="10" cy="10" r="7"/>
                      <line x1="21" y1="21" x2="15" y2="15"/>
                    </svg>
                  </span>
                  <input 
                    type="text" 
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="form-control" 
                    placeholder="Search…"
                    aria-label="Search CreateMod.com"
                  />
                </div>
              </form>
            </div>
          </div>
          
          {/* Theme toggle */}
          <div className="col-auto">
            <div className="d-none d-md-flex">
              {theme === 'light' ? (
                <a 
                  href="#" 
                  className="nav-link px-0" 
                  title="Enable dark mode" 
                  onClick={(e) => {
                    e.preventDefault();
                    toggleTheme();
                  }}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" 
                       stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"/>
                  </svg>
                </a>
              ) : (
                <a 
                  href="#" 
                  className="nav-link px-0" 
                  title="Enable light mode" 
                  onClick={(e) => {
                    e.preventDefault();
                    toggleTheme();
                  }}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" 
                       stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <circle cx="12" cy="12" r="4"/>
                    <path d="M3 12h1m8 -9v1m8 8h1m-9 8v1m-6.4 -15.4l.7 .7m12.1 -.7l-.7 .7m0 11.4l.7 .7m-12.1 -.7l-.7 .7"/>
                  </svg>
                </a>
              )}
            </div>
          </div>
          
          {/* User menu */}
          <div className="col col-auto d-none d-lg-block">
            {isAuthenticated ? (
              <div className="nav-item dropdown">
                <a href="#" className="nav-link d-flex lh-1 text-reset p-0" data-bs-toggle="dropdown" aria-label="Open user menu">
                  {user?.avatar && (
                    <div className="avatar avatar-sm" style={{ backgroundImage: `url(${user.avatar})` }}></div>
                  )}
                  <div className="d-none d-xl-block ps-2">
                    <div>{user?.username}</div>
                  </div>
                </a>
                <div className="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
                  <Link href={`/author/${user?.username?.toLowerCase()}`} className="dropdown-item">
                    Profile
                  </Link>
                  <div className="dropdown-divider"></div>
                  <Link href="/settings" className="dropdown-item">
                    Settings
                  </Link>
                  <a 
                    href="#" 
                    className="dropdown-item" 
                    onClick={(e) => {
                      e.preventDefault();
                      handleLogout();
                    }}
                  >
                    Logout
                  </a>
                </div>
              </div>
            ) : (
              <button 
                onClick={() => router.push('/login/')} 
                className="btn btn-primary"
              >
                Login
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}