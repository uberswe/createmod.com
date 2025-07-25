/**
 * Logging utilities for debugging
 */

/**
 * Log the current state of cookies
 * @param {string} context - The context where the logging is happening (e.g., 'LOGIN', 'LAYOUT')
 * @param {string} action - The action being performed (e.g., 'INIT', 'AFTER_LOGIN')
 */
export function logCookies(context, action) {
  if (typeof document === 'undefined') {
    console.log(`[${context}] Cannot log cookies: document is undefined (server-side)`);
    return;
  }
  
  const cookies = document.cookie.split(';').map(cookie => cookie.trim());
  const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
  
  // Only log the existence of cookies, not their values
  console.log(`[${context}] Cookies ${action}:`, {
    cookieCount: cookies.length,
    authCookieExists: !!authCookie
  });
}

/**
 * Log a navigation/redirect event
 * @param {string} context - The context where the logging is happening (e.g., 'LOGIN', 'LAYOUT')
 * @param {string} from - The path navigating from
 * @param {string} to - The path navigating to
 * @param {string} reason - The reason for the navigation
 */
export function logNavigation(context, from, to, reason) {
  console.log(`[${context}] Navigation:`, {
    from,
    to,
    reason
  });
}

/**
 * Log an authentication event
 * @param {string} context - The context where the logging is happening (e.g., 'LOGIN', 'LAYOUT')
 * @param {string} event - The authentication event (e.g., 'LOGIN_ATTEMPT', 'LOGIN_SUCCESS', 'LOGOUT')
 * @param {Object} data - Additional data to log
 */
export function logAuth(context, event, data = {}) {
  // Create a sanitized copy of the data object to avoid logging sensitive information
  const sanitizedData = { ...data };
  
  // Remove sensitive fields if they exist
  const sensitiveFields = [
    'token', 'password', 'authToken', 'cookieValue', 'cookieStart', 
    'tokenLength', 'email', 'identity'
  ];
  
  sensitiveFields.forEach(field => {
    if (field in sanitizedData) {
      sanitizedData[field] = '[REDACTED]';
    }
  });
  
  // For user data, only log non-sensitive information
  if (sanitizedData.user) {
    sanitizedData.user = {
      id: sanitizedData.user.id,
      username: sanitizedData.user.username,
      verified: sanitizedData.user.verified
    };
  }
  
  // For record data, only log non-sensitive information
  if (sanitizedData.record) {
    sanitizedData.record = {
      id: sanitizedData.record.id,
      username: sanitizedData.record.username,
      verified: sanitizedData.record.verified
    };
  }
  
  console.log(`[${context}] Auth event: ${event}`, sanitizedData);
}

/**
 * Log an error with context
 * @param {string} context - The context where the logging is happening (e.g., 'LOGIN', 'LAYOUT')
 * @param {string} message - Error message
 * @param {Error} error - The error object
 */
export function logError(context, message, error) {
  console.error(`[${context}] ${message}:`, error);
  
  // Log additional details if available
  if (error.response) {
    console.error(`[${context}] Error response:`, {
      status: error.response.status,
      data: error.response.data
    });
  }
}