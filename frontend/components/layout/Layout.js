import React, { useState, useEffect } from 'react';
import Head from 'next/head';
import Sidebar from './Sidebar';
import Header from './Header';
import Footer from './Footer';
import { useRouter } from 'next/router';
import { logCookies, logNavigation, logAuth, logError } from '../../utils/logger';
import { setAuthCookie } from '../../lib/auth';

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
        // Log the auth cookie before sending the request
        if (typeof document !== 'undefined') {
          const cookies = document.cookie.split(';').map(cookie => cookie.trim());
          const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
          if (authCookie) {
            logAuth('LAYOUT', 'AUTH_COOKIE_FOUND_BEFORE_REFRESH', { 
              cookieStart: authCookie.substring(0, 30) + '...',
              cookieLength: authCookie.length
            });
          } else {
            logAuth('LAYOUT', 'NO_AUTH_COOKIE_FOUND_BEFORE_REFRESH');
          }
        }
        
        logAuth('LAYOUT', 'AUTH_REFRESH_REQUEST_SENDING');
        const response = await fetch('/api/collections/users/auth-refresh', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
        });
        
        // Log detailed information about the response
        const responseHeaders = {};
        response.headers.forEach((value, name) => {
          responseHeaders[name] = value;
        });
        
        logAuth('LAYOUT', 'AUTH_REFRESH_RESPONSE_RECEIVED', { 
          status: response.status,
          statusText: response.statusText,
          headers: responseHeaders
        });
        
        if (response.ok) {
          const userData = await response.json();
          
          // Log detailed information about the user data
          logAuth('LAYOUT', 'AUTH_REFRESH_SUCCESS', {
            id: userData.record?.id,
            username: userData.record?.username,
            email: userData.record?.email,
            verified: userData.record?.verified
          });
          
          // Log the structure of the userData object to help identify where the token is
          console.log('[LAYOUT] Auth refresh response userData structure:', {
            hasToken: !!userData.token,
            tokenLength: userData.token ? userData.token.length : 0,
            hasRecord: !!userData.record,
            recordKeys: userData.record ? Object.keys(userData.record) : [],
            otherKeys: Object.keys(userData).filter(key => key !== 'token' && key !== 'record')
          });
          
          // Check if the response contains a token in the body
          if (userData.token) {
            setAuthCookie(userData.token);
            logAuth('LAYOUT', 'AUTH_COOKIE_SET_WITH_PROPER_ATTRIBUTES');
          } else {
            // If no token in body, try to extract from headers (less reliable)
            const authToken = response.headers.get('set-cookie')?.match(/create-mod-auth=([^;]+)/)?.[1];
            if (authToken) {
              setAuthCookie(authToken);
              logAuth('LAYOUT', 'AUTH_COOKIE_SET_FROM_HEADERS');
            } else {
              logAuth('LAYOUT', 'NO_AUTH_TOKEN_FOUND_IN_RESPONSE');
            }
          }
          
          setIsAuthenticated(true);
          setUser(userData.record);
          logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: true });
          logCookies('LAYOUT', 'AFTER_AUTH_SUCCESS');
        } else {
          logAuth('LAYOUT', 'AUTH_REFRESH_FAILED', { status: response.status });
          
          // Use the logout endpoint to properly clear the HttpOnly cookie
          logAuth('LAYOUT', 'CALLING_LOGOUT_ENDPOINT_TO_CLEAR_COOKIE');
          try {
            const logoutResponse = await fetch('/api/collections/users/auth-logout', {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
              credentials: 'include',
            });
            logAuth('LAYOUT', 'LOGOUT_RESPONSE_RECEIVED', { status: logoutResponse.status });
          } catch (logoutError) {
            logError('LAYOUT', 'Error calling logout endpoint', logoutError);
          }
          
          setIsAuthenticated(false);
          setUser(null);
          logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false });
          logCookies('LAYOUT', 'AFTER_AUTH_FAILURE');
        }
      } catch (error) {
        logError('LAYOUT', 'Error checking authentication', error);
        
        // Use the logout endpoint to properly clear the HttpOnly cookie
        logAuth('LAYOUT', 'CALLING_LOGOUT_ENDPOINT_TO_CLEAR_COOKIE_ON_ERROR');
        try {
          const logoutResponse = await fetch('/api/collections/users/auth-logout', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            credentials: 'include',
          });
          logAuth('LAYOUT', 'LOGOUT_RESPONSE_RECEIVED_ON_ERROR', { status: logoutResponse.status });
        } catch (logoutError) {
          logError('LAYOUT', 'Error calling logout endpoint on auth error', logoutError);
        }
        
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
      logAuth('LAYOUT', 'LOGOUT_REQUEST_SENDING');
      // Use the proper logout endpoint to clear the HttpOnly cookie on the server side
      const response = await fetch('/api/collections/users/auth-logout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });
      
      logAuth('LAYOUT', 'LOGOUT_RESPONSE_RECEIVED', { status: response.status });
      
      // The server should have cleared the HttpOnly cookie in the response
      // No need to try clearing it client-side as that won't work with HttpOnly cookies
      logCookies('LAYOUT', 'AFTER_SERVER_COOKIE_CLEAR');
      
      logAuth('LAYOUT', 'UPDATING_AUTH_STATE');
      setIsAuthenticated(false);
      setUser(null);
      logAuth('LAYOUT', 'AUTH_STATE_UPDATED', { isAuthenticated: false });
      
      logNavigation('LAYOUT', router.pathname, '/', 'logout');
      router.push('/');
    } catch (error) {
      logError('LAYOUT', 'Error during logout', error);
      
      // Try again to call the logout endpoint to clear the cookie
      try {
        logAuth('LAYOUT', 'RETRY_LOGOUT_REQUEST_ON_ERROR');
        await fetch('/api/collections/users/auth-logout', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
        });
      } catch (retryError) {
        logError('LAYOUT', 'Error during logout retry', retryError);
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