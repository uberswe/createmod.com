import React, { useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import Image from 'next/image';

/**
 * Sidebar component with navigation links and categories
 * 
 * @param {Object} props - Component props
 * @param {Array} props.categories - Categories for navigation
 * @param {boolean} props.isAuthenticated - Whether user is authenticated
 * @param {Object} props.user - User data if authenticated
 * @param {Function} props.handleLogout - Function to handle user logout
 * @param {Function} props.toggleTheme - Function to toggle theme
 * @param {string} props.theme - Current theme ('light' or 'dark')
 */
export default function Sidebar({ 
  categories = [], 
  isAuthenticated = false, 
  user = null,
  handleLogout,
  toggleTheme,
  theme = 'light'
}) {
  const router = useRouter();
  
  // Log component render with authentication state
  useEffect(() => {
    console.log('[SIDEBAR] Sidebar component rendered with:', {
      isAuthenticated,
      userId: user?.id,
      username: user?.username,
      categoriesCount: categories.length,
      path: router.pathname
    });
    
    // Log conditional rendering decisions
    console.log('[SIDEBAR] Sidebar rendering decisions:', {
      showLoginButton: !isAuthenticated,
      showUserMenu: isAuthenticated,
      showLogoutButton: isAuthenticated
    });
  }, [isAuthenticated, user, categories.length, router.pathname]);
  
  /**
   * Check if the current path matches the given path
   * @param {string} path - Path to check
   * @returns {boolean} - Whether the path matches
   */
  const isActive = (path) => {
    return router.pathname === path || router.asPath === path;
  };
  
  return (
    <aside className="navbar navbar-vertical navbar-expand-lg" data-bs-theme="dark">
      <div className="container-fluid">
        {/* Mobile toggle button */}
        <button 
          className="navbar-toggler" 
          type="button" 
          data-bs-toggle="collapse" 
          data-bs-target="#sidebar-menu" 
          aria-controls="sidebar-menu" 
          aria-expanded="false" 
          aria-label="Toggle navigation"
        >
          <span className="navbar-toggler-icon"></span>
        </button>
        
        {/* Logo */}
        <h1 className="navbar-brand navbar-brand-autodark ms-3">
          <Link href="/">
            <Image 
              src="/logo.png" 
              alt="CreateMod.com logo" 
              width={150} 
              height={40} 
              priority
            />
          </Link>
        </h1>
        
        {/* Mobile user menu */}
        <div className="navbar-nav flex-row d-lg-none">
          {isAuthenticated ? (
            <div className="nav-item dropdown auth-section">
              <a href="#" className="nav-link d-flex lh-1 text-reset p-0 dropdown" data-bs-toggle="dropdown" aria-label="Open user menu">
                {user?.avatar && (
                  <Image 
                    className="avatar avatar-sm auth-avatar" 
                    src={user?.avatar} 
                    alt={user?.username} 
                    width={32} 
                    height={32} 
                  />
                )}
                <div className="d-none d-xl-block ps-2">
                  <div className="auth-username">{user?.username}</div>
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
                  className="dropdown-item logout-button" 
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
            <a 
              href="#" 
              className="nav-link"
              onClick={(e) => {
                e.preventDefault();
                router.push('/login/');
              }}
            >
              Login
            </a>
          )}
        </div>
        
        {/* Sidebar menu */}
        <div className="collapse navbar-collapse" id="sidebar-menu">
          <ul className="navbar-nav pt-lg-3">
            {/* Home */}
            <li className="nav-item">
              <Link 
                href="/" 
                className={`nav-link ${isActive('/') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M5 12l-2 0l9 -9l9 9l-2 0" />
                    <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7" />
                    <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Home
                </span>
              </Link>
            </li>
            
            {/* Search */}
            <li className="nav-item">
              <Link 
                href="/search/?sort=6&rating=-1&category=all&tag=all" 
                className={`nav-link ${isActive('/search') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-search">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0" />
                    <path d="M21 21l-6 -6" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Search
                </span>
              </Link>
            </li>
            
            {/* Categories */}
            {categories.map((category) => (
              <li className="nav-item" key={category.id}>
                <Link 
                  href={`/search?category=${category.key}&sort=6`}
                  className="nav-link"
                  style={{ marginLeft: '15px' }}
                >
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" className="bi bi-dash-lg" viewBox="0 0 16 16">
                      <path fillRule="evenodd" d="M2 8a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11A.5.5 0 0 1 2 8"/>
                    </svg>
                  </span>
                  <span className="nav-link-title">{category.name}</span>
                </Link>
              </li>
            ))}
            
            {/* Upload */}
            <li className="nav-item">
              <Link 
                href="/upload" 
                className={`nav-link ${isActive('/upload') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M9 11l3 3l8 -8" />
                    <path d="M20 12v6a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12a2 2 0 0 1 2 -2h9" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Upload
                </span>
              </Link>
            </li>
            
            {/* Rules */}
            <li className="nav-item">
              <Link 
                href="/rules" 
                className={`nav-link ${isActive('/rules') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-ruler-2">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M17 3l4 4l-14 14l-4 -4z" />
                    <path d="M16 7l-1.5 -1.5" />
                    <path d="M13 10l-1.5 -1.5" />
                    <path d="M10 13l-1.5 -1.5" />
                    <path d="M7 16l-1.5 -1.5" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Rules
                </span>
              </Link>
            </li>
            
            {/* News */}
            <li className="nav-item">
              <Link 
                href="/news" 
                className={`nav-link ${isActive('/news') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-news">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11" />
                    <path d="M8 8l4 0" />
                    <path d="M8 12l4 0" />
                    <path d="M8 16l4 0" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  News
                </span>
              </Link>
            </li>
            
            {/* Guide */}
            <li className="nav-item">
              <Link 
                href="/guide" 
                className={`nav-link ${isActive('/guide') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-list-check">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M3.5 5.5l1.5 1.5l2.5 -2.5" />
                    <path d="M3.5 11.5l1.5 1.5l2.5 -2.5" />
                    <path d="M3.5 17.5l1.5 1.5l2.5 -2.5" />
                    <path d="M11 6l9 0" />
                    <path d="M11 12l9 0" />
                    <path d="M11 18l9 0" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Guide
                </span>
              </Link>
            </li>
            
            {/* Explore */}
            <li className="nav-item">
              <Link 
                href="/explore" 
                className={`nav-link ${isActive('/explore') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-library-photo">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M7 3m0 2.667a2.667 2.667 0 0 1 2.667 -2.667h8.666a2.667 2.667 0 0 1 2.667 2.667v8.666a2.667 2.667 0 0 1 -2.667 2.667h-8.666a2.667 2.667 0 0 1 -2.667 -2.667z" />
                    <path d="M4.012 7.26a2.005 2.005 0 0 0 -1.012 1.737v10c0 1.1 .9 2 2 2h10c.75 0 1.158 -.385 1.5 -1" />
                    <path d="M17 7h.01" />
                    <path d="M7 13l3.644 -3.644a1.21 1.21 0 0 1 1.712 0l3.644 3.644" />
                    <path d="M15 12l1.644 -1.644a1.21 1.21 0 0 1 1.712 0l2.644 2.644" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Explore
                </span>
              </Link>
            </li>
            
            {/* Contact */}
            <li className="nav-item">
              <Link 
                href="/contact" 
                className={`nav-link ${isActive('/contact') ? 'active' : ''}`}
              >
                <span className="nav-link-icon d-md-none d-lg-inline-block">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-mail">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M3 7a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-10z" />
                    <path d="M3 7l9 6l9 -6" />
                  </svg>
                </span>
                <span className="nav-link-title">
                  Contact
                </span>
              </Link>
            </li>
            
            {/* Authentication */}
            {!isAuthenticated ? (
              <li className="nav-item">
                <a 
                  href="#" 
                  className={`nav-link ${isActive('/login/') ? 'active' : ''}`}
                  onClick={(e) => {
                    e.preventDefault();
                    router.push('/login/');
                  }}
                >
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-login">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M15 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2" />
                      <path d="M21 12h-13l3 -3" />
                      <path d="M11 15l-3 -3" />
                    </svg>
                  </span>
                  <span className="nav-link-title">
                    Login
                  </span>
                </a>
              </li>
            ) : (
              <>
                {/* Settings - Mobile only */}
                <li className="nav-item d-xl-none d-inline-block">
                  <Link 
                    href="/settings" 
                    className={`nav-link ${isActive('/settings') ? 'active' : ''}`}
                  >
                    <span className="nav-link-icon d-md-none d-lg-inline-block">
                      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-settings">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z" />
                        <path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" />
                      </svg>
                    </span>
                    <span className="nav-link-title">
                      Settings
                    </span>
                  </Link>
                </li>
                
                {/* Profile - Mobile only */}
                <li className="nav-item d-xl-none d-inline-block">
                  <Link 
                    href={`/author/${user?.username?.toLowerCase()}`} 
                    className={`nav-link ${isActive(`/author/${user?.username?.toLowerCase()}`) ? 'active' : ''}`}
                  >
                    <span className="nav-link-icon d-md-none d-lg-inline-block">
                      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-user">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0" />
                        <path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" />
                      </svg>
                    </span>
                    <span className="nav-link-title">
                      Profile
                    </span>
                  </Link>
                </li>
                
                {/* Logout */}
                <li className="nav-item">
                  <a 
                    href="#" 
                    className="nav-link" 
                    onClick={(e) => {
                      e.preventDefault();
                      handleLogout();
                    }}
                  >
                    <span className="nav-link-icon d-md-none d-lg-inline-block">
                      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-logout">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2" />
                        <path d="M7 12h14l-3 -3m0 6l3 -3" />
                      </svg>
                    </span>
                    <span className="nav-link-title">
                      Logout
                    </span>
                  </a>
                </li>
              </>
            )}
          </ul>
        </div>
      </div>
    </aside>
  );
}