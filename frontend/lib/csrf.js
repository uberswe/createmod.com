/**
 * CSRF protection utilities
 */

/**
 * Generate a random CSRF token
 * @returns {string} - Random CSRF token
 */
export function generateCSRFToken() {
  // Generate a random string of 32 characters
  const randomBytes = new Uint8Array(32);
  if (typeof window !== 'undefined' && window.crypto) {
    window.crypto.getRandomValues(randomBytes);
  } else {
    // Fallback for server-side rendering
    for (let i = 0; i < randomBytes.length; i++) {
      randomBytes[i] = Math.floor(Math.random() * 256);
    }
  }
  
  // Convert to base64 and remove non-alphanumeric characters
  const token = btoa(String.fromCharCode.apply(null, randomBytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
  
  // Store the token in sessionStorage for validation
  if (typeof window !== 'undefined') {
    sessionStorage.setItem('csrfToken', token);
  }
  
  return token;
}

/**
 * Validate a CSRF token against the stored token
 * @param {string} token - CSRF token to validate
 * @returns {boolean} - Whether the token is valid
 */
export function validateCSRFToken(token) {
  if (typeof window === 'undefined') {
    console.warn('[CSRF] Cannot validate token on server side');
    return false;
  }
  
  const storedToken = sessionStorage.getItem('csrfToken');
  
  // If no token is stored, validation fails
  if (!storedToken) {
    console.warn('[CSRF] No stored token found for validation');
    return false;
  }
  
  // Compare the tokens using a constant-time comparison to prevent timing attacks
  if (token.length !== storedToken.length) {
    return false;
  }
  
  let result = 0;
  for (let i = 0; i < token.length; i++) {
    result |= token.charCodeAt(i) ^ storedToken.charCodeAt(i);
  }
  
  return result === 0;
}

/**
 * Get the current CSRF token or generate a new one
 * @returns {string} - CSRF token
 */
export function getCSRFToken() {
  if (typeof window === 'undefined') {
    // For server-side rendering, generate a new token
    return generateCSRFToken();
  }
  
  // For client-side, get the stored token or generate a new one
  const storedToken = sessionStorage.getItem('csrfToken');
  if (storedToken) {
    return storedToken;
  }
  
  return generateCSRFToken();
}