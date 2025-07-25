/**
 * Authentication utility functions
 */

import { pb, refreshAuth as pbRefreshAuth, logout as pbLogout } from './pocketbase';

/**
 * Validate authentication on the server side
 * @param {Object} req - Next.js request object with cookies
 * @returns {Promise<{isAuthenticated: boolean, user: Object|null}>} - Authentication status and user data
 */
export async function validateServerAuth(req) {
  // Default return value
  const result = {
    isAuthenticated: false,
    user: null
  };

  try {
    // Check if auth cookie exists
    const authCookie = req.cookies['create-mod-auth'];
    if (!authCookie) {
      console.log('[SERVER-AUTH] No auth cookie found');
      return result;
    }

    // Set the token in PocketBase's authStore
    pb.authStore.save(authCookie, null);
    
    // Check if the token is valid
    if (!pb.authStore.isValid) {
      console.log('[SERVER-AUTH] Auth token is invalid');
      return result;
    }
    
    console.log('[SERVER-AUTH] Auth token loaded into PocketBase authStore');

    try {
      // Refresh the token to validate it
      const authData = await pb.collection('users').authRefresh();
      console.log('[SERVER-AUTH] Auth token validated successfully');
      
      return {
        isAuthenticated: true,
        user: authData.record
      };
    } catch (error) {
      console.log('[SERVER-AUTH] Auth token validation failed:', error);
      // Clear the invalid token
      pb.authStore.clear();
      return result;
    }
  } catch (error) {
    console.error('[SERVER-AUTH] Error validating auth token:', error);
    return result;
  }
}

/**
 * Set authentication cookie with proper attributes
 * @param {string} token - Authentication token
 * @param {number} maxAge - Cookie max age in seconds (default: 30 days)
 */
export function setAuthCookie(token, maxAge = 30 * 24 * 60 * 60) {
  if (typeof document === 'undefined') {
    console.log('[CLIENT-AUTH] Cannot set cookie: document is undefined (server-side)');
    return;
  }

  // Set the token in PocketBase's authStore if it's not already set
  if (!pb.authStore.isValid) {
    console.log('[CLIENT-AUTH] Setting token in PocketBase authStore');
    pb.authStore.save(token, null);
  }

  // Set the cookie with proper attributes
  // For local development, don't set domain to ensure cookie works with localhost
  // SameSite=Lax allows the cookie to be sent with same-site requests and top-level navigations
  // HttpOnly flag prevents JavaScript access to the cookie, protecting against XSS attacks
  // Secure flag ensures the cookie is only sent over HTTPS connections
  const isProduction = process.env.NODE_ENV === 'production';
  document.cookie = `create-mod-auth=${token}; Path=/; Max-Age=${maxAge}; SameSite=Lax; HttpOnly${isProduction ? '; Secure' : ''}`;
  console.log('[CLIENT-AUTH] Auth cookie set with Max-Age:', maxAge, 'HttpOnly:', true, 'Secure:', isProduction);
  
  // Log the cookie for debugging
  const cookies = document.cookie.split(';').map(cookie => cookie.trim());
  const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
  console.log('[CLIENT-AUTH] Auth cookie after setting:', authCookie ? `${authCookie.substring(0, 30)}...` : 'Not found');
}

/**
 * Clear authentication cookie
 */
export function clearAuthCookie() {
  if (typeof document === 'undefined') {
    console.log('[CLIENT-AUTH] Cannot clear cookie: document is undefined (server-side)');
    return;
  }

  // Log the cookies before clearing
  const cookiesBeforeClear = document.cookie.split(';').map(cookie => cookie.trim());
  const authCookieBeforeClear = cookiesBeforeClear.find(cookie => cookie.startsWith('create-mod-auth='));
  console.log('[CLIENT-AUTH] Auth cookie before clearing:', authCookieBeforeClear ? `${authCookieBeforeClear.substring(0, 30)}...` : 'Not found');

  // Clear the PocketBase authStore
  console.log('[CLIENT-AUTH] Clearing PocketBase authStore');
  pbLogout();

  // Clear the cookie by setting an expiration date in the past
  // Use multiple approaches to ensure the cookie is cleared in all environments
  // Include the same flags (HttpOnly, Secure) as when setting the cookie
  const isProduction = process.env.NODE_ENV === 'production';
  const secureFlag = isProduction ? '; Secure' : '';
  document.cookie = `create-mod-auth=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT; HttpOnly; SameSite=Lax${secureFlag}`;
  document.cookie = `create-mod-auth=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax${secureFlag}`;
  
  // Log the cookies after clearing
  const cookiesAfterClear = document.cookie.split(';').map(cookie => cookie.trim());
  const authCookieAfterClear = cookiesAfterClear.find(cookie => cookie.startsWith('create-mod-auth='));
  console.log('[CLIENT-AUTH] Auth cookie after clearing:', authCookieAfterClear ? `${authCookieAfterClear.substring(0, 30)}...` : 'Successfully cleared');
}