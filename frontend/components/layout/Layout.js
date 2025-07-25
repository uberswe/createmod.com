import React, { useState, useEffect } from 'react';
import Head from 'next/head';
import Sidebar from './Sidebar';
import Header from './Header';
import Footer from './Footer';
import { useRouter } from 'next/router';
import { logCookies, logNavigation, logAuth, logError } from '../../utils/logger';
import { setAuthCookie, clearAuthCookie } from '../../lib/auth';
import { pb, isAuthenticated as pbIsAuthenticated, getCurrentUser, refreshAuth, logout } from '../../lib/pocketbase';

/**
 * Main layout component for the application
 * 
 * @param {Object} props - Component props
 * @param {React.ReactNode} props.children - Page content
 * @param {string} props.title - Page title
 * @param {string} props.description - Page description
 * @param {string} props.thumbnail - Thumbnail image URL for social sharing
 * @param {string} props.slug - Page slug for canonical URL
 * @param {string} props.subCategory - Page subcategory for header
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function Layout({ 
  children, 
  title = 'CreateMod.com', 
  description = 'Share and discover schematics for the Create mod',
  thumbnail = '',
  slug = '',
  subCategory = '',
  categories = []
}) {
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState(null);
  const [theme, setTheme] = useState('light');
  
  // Log authentication state changes
  useEffect(() => {
    console.log('[LAYOUT] Authentication state changed:', { 
      isAuthenticated, 
      userId: user?.id,
      username: user?.username
    });
  }, [isAuthenticated, user]);

  useEffect(() => {
    // Check if user is authenticated
    const checkAuth = async () => {
      logAuth('LAYOUT', 'AUTH_CHECK_STARTED');
      logCookies('LAYOUT', 'BEFORE_AUTH_CHECK');
      
      try {
        // Check if PocketBase has a valid token in its authStore
        const isValid = pbIsAuthenticated();
        logAuth('LAYOUT', 'POCKETBASE_AUTH_CHECK', { isValid });
        
        if (isValid) {
          // Get the current user from PocketBase's authStore
          const currentUser = getCurrentUser();
          
          // Log detailed information about the user data
          logAuth('LAYOUT', 'POCKETBASE_AUTH_VALID', {
            id: currentUser?.id,
            username: currentUser?.username,
            email: currentUser?.email,
            verified: currentUser?.verified
          });
          
          // Try to refresh the token to ensure it's still valid
          try {
            logAuth('LAYOUT', 'POCKETBASE_AUTH_REFRESH_STARTED');
            const authData = await refreshAuth();
            
            logAuth('LAYOUT', 'POCKETBASE_AUTH_REFRESH_SUCCESS', {
              id: authData.record?.id,
              username: authData.record?.username,
              email: authData.record?.email,
              verified: authData.record?.verified
            });
            
            // Ensure the token is also set in the cookie for backward compatibility
            if (authData.token) {
              setAuthCookie(authData.token);
              logAuth('LAYOUT', 'AUTH_COOKIE_SET_WITH_PROPER_ATTRIBUTES');
            }
            
            setIsAuthenticated(true);
            setUser(authData.record);
            logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: true });
            logCookies('LAYOUT', 'AFTER_AUTH_SUCCESS');
          } catch (refreshError) {
            logError('LAYOUT', 'Error refreshing authentication', refreshError);
            
            // Clear the auth state
            logout();
            clearAuthCookie();
            
            setIsAuthenticated(false);
            setUser(null);
            logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'refresh_error' });
            logCookies('LAYOUT', 'AFTER_AUTH_REFRESH_ERROR');
          }
        } else {
          // Check if there's an auth cookie that PocketBase doesn't know about
          if (typeof document !== 'undefined') {
            const cookies = document.cookie.split(';').map(cookie => cookie.trim());
            const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
            
            if (authCookie) {
              logAuth('LAYOUT', 'AUTH_COOKIE_FOUND_BUT_POCKETBASE_NOT_AUTHENTICATED', { 
                cookieStart: authCookie.substring(0, 30) + '...',
                cookieLength: authCookie.length
              });
              
              // Try to set the token in PocketBase's authStore
              const token = authCookie.split('=')[1];
              pb.authStore.save(token, null);
              
              // Check if the token is now valid
              if (pb.authStore.isValid) {
                logAuth('LAYOUT', 'POCKETBASE_AUTH_SET_FROM_COOKIE_SUCCESS');
                
                // Try to refresh the token
                try {
                  const authData = await refreshAuth();
                  
                  setIsAuthenticated(true);
                  setUser(authData.record);
                  logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: true, source: 'cookie' });
                } catch (refreshError) {
                  logError('LAYOUT', 'Error refreshing authentication from cookie', refreshError);
                  
                  // Clear the auth state
                  logout();
                  clearAuthCookie();
                  
                  setIsAuthenticated(false);
                  setUser(null);
                  logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'cookie_refresh_error' });
                }
              } else {
                logAuth('LAYOUT', 'POCKETBASE_AUTH_SET_FROM_COOKIE_FAILED');
                clearAuthCookie();
                setIsAuthenticated(false);
                setUser(null);
                logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'invalid_cookie' });
              }
            } else {
              logAuth('LAYOUT', 'NO_AUTH_COOKIE_FOUND');
              setIsAuthenticated(false);
              setUser(null);
              logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'no_cookie' });
            }
          } else {
            // Server-side rendering, can't check cookies
            logAuth('LAYOUT', 'SERVER_SIDE_RENDERING_NO_COOKIE_CHECK');
            setIsAuthenticated(false);
            setUser(null);
            logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'server_side' });
          }
        }
      } catch (error) {
        logError('LAYOUT', 'Error checking authentication', error);
        
        // Clear the auth state
        logout();
        clearAuthCookie();
        
        setIsAuthenticated(false);
        setUser(null);
        logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'error' });
        logCookies('LAYOUT', 'AFTER_AUTH_ERROR');
      }
    };

    // Check for theme preference
    if (typeof window !== 'undefined') {
      const savedTheme = localStorage.getItem('createmodTheme') || 'light';
      console.log('[LAYOUT] Theme preference loaded:', savedTheme);
      setTheme(savedTheme);
      document.documentElement.setAttribute('data-bs-theme', savedTheme);
    }

    checkAuth();
    console.log('[LAYOUT] Initial authentication check triggered');
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('createmodTheme', newTheme);
    document.documentElement.setAttribute('data-bs-theme', newTheme);
  };

  const handleLogout = async () => {
    logAuth('LAYOUT', 'LOGOUT_STARTED');
    logCookies('LAYOUT', 'BEFORE_LOGOUT');
    
    try {
      logAuth('LAYOUT', 'POCKETBASE_LOGOUT_STARTED');
      
      // Use PocketBase's logout function to clear the authStore
      logout();
      
      // Also clear the HttpOnly cookie for backward compatibility
      clearAuthCookie();
      
      logAuth('LAYOUT', 'POCKETBASE_LOGOUT_COMPLETED');
      logCookies('LAYOUT', 'AFTER_LOGOUT');
      
      // Update the component state
      logAuth('LAYOUT', 'UPDATING_AUTH_STATE');
      setIsAuthenticated(false);
      setUser(null);
      logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false });
      
      // For backward compatibility, also try to call the server logout endpoint
      try {
        logAuth('LAYOUT', 'LEGACY_LOGOUT_REQUEST_SENDING');
        const response = await fetch('/api/collections/users/auth-logout', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
        });
        logAuth('LAYOUT', 'LEGACY_LOGOUT_RESPONSE_RECEIVED', { status: response.status });
      } catch (legacyError) {
        // Ignore errors from the legacy logout endpoint
        logError('LAYOUT', 'Error during legacy logout', legacyError);
      }
      
      logNavigation('LAYOUT', router.pathname, '/', 'logout');
      router.push('/');
    } catch (error) {
      logError('LAYOUT', 'Error during logout', error);
      
      // Try to clear the auth state anyway
      try {
        logout();
        clearAuthCookie();
        logAuth('LAYOUT', 'POCKETBASE_LOGOUT_COMPLETED_ON_ERROR');
      } catch (clearError) {
        logError('LAYOUT', 'Error clearing auth state on logout error', clearError);
      }
      
      // Update the authentication state regardless of the outcome
      setIsAuthenticated(false);
      setUser(null);
      logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false, reason: 'error' });
      
      // Still redirect to home page
      router.push('/');
    }
  };

  // Construct canonical URL
  const canonicalUrl = slug ? `https://createmod.com/${slug}` : 'https://createmod.com';
  
  return (
    <div className="page">
      <Head>
        <title>{title}</title>
        <meta name="description" content={description} />
        <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
        <link rel="shortcut icon" href="/favicon-192x192.png" type="image/x-icon" />
        
        {/* Canonical URL */}
        <link rel="canonical" href={canonicalUrl} />
        
        {/* Open Graph / Social Media Meta Tags */}
        <meta property="og:title" content={title} />
        <meta property="og:description" content={description} />
        <meta property="og:type" content="website" />
        <meta property="og:url" content={canonicalUrl} />
        {thumbnail && <meta property="og:image" content={thumbnail} />}
        
        {/* Twitter Card Meta Tags */}
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content={title} />
        <meta name="twitter:description" content={description} />
        {thumbnail && <meta name="twitter:image" content={thumbnail} />}
      </Head>
      
      {/* Sidebar */}
      <Sidebar 
        categories={categories}
        isAuthenticated={isAuthenticated}
        user={user}
        handleLogout={handleLogout}
        toggleTheme={toggleTheme}
        theme={theme}
      />
      
      <div className="page-wrapper">
        {/* Header */}
        <Header 
          title={title}
          subCategory={subCategory}
          isAuthenticated={isAuthenticated}
          user={user}
          handleLogout={handleLogout}
          toggleTheme={toggleTheme}
          theme={theme}
        />
        
        {/* Main content */}
        <div className="page-body">
          <div className="container-xl">
            {children}
          </div>
        </div>
        
        {/* Footer */}
        <Footer />
      </div>
    </div>
  );
}