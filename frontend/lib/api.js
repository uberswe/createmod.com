/**
 * API utility functions for communicating with the PocketBase backend
 */

// Base URL for API requests
// Use environment variable if available, otherwise default to relative path
const API_BASE_URL = typeof window === 'undefined' 
  ? (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090/api')
  : '/api';

/**
 * Fetch data from the API with error handling
 * @param {string} endpoint - API endpoint to fetch from
 * @param {Object} options - Fetch options
 * @returns {Promise<Object>} - Response data
 */
export async function fetchAPI(endpoint, options = {}) {
  // Construct the full URL, ensuring it's absolute when running on the server
  let url;
  
  console.log(`[API] fetchAPI called for endpoint: ${endpoint} with method: ${options.method || 'GET'}`);
  
  try {
    // Handle server-side rendering
    if (typeof window === 'undefined') {
      // For server-side, ensure we have a complete URL with the /api prefix
      const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
      console.log(`[API] Server-side rendering detected, using baseUrl: ${baseUrl}`);
      
      // Create a URL object with a base URL to ensure proper URL construction
      const base = new URL(baseUrl);
      
      // Remove the endpoint's leading slash if present
      const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
      
      // Ensure the path includes /api/ prefix
      const apiPath = '/api/';
      
      // Construct the full URL
      url = new URL(apiPath + cleanEndpoint, base).toString();
      console.log(`[API] Constructed server-side URL: ${url}`);
    } else {
      // For client-side, use relative URL with /api prefix
      // Ensure the API path has a trailing slash for proper URL joining
      const apiPath = '/api/';
      
      // Remove the endpoint's leading slash if present
      const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
      
      // Construct the full URL
      url = apiPath + cleanEndpoint;
      console.log(`[API] Constructed client-side URL: ${url}`);
    }
  } catch (error) {
    console.error('[API] Error constructing URL:', error);
    throw new Error(`Invalid URL construction: ${endpoint}`);
  }
  
  // Log the constructed URL for debugging
  console.log(`[API] Request URL: ${url}`);
  
  // Set default headers
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  
  // Include credentials for authentication
  const fetchOptions = {
    ...options,
    headers,
    credentials: 'include',
  };
  
  console.log(`[API] Request options:`, {
    method: fetchOptions.method || 'GET',
    headers: fetchOptions.headers,
    credentials: fetchOptions.credentials,
    bodyLength: fetchOptions.body ? fetchOptions.body.length : 0
  });
  
  try {
    console.log(`[API] Sending request to ${url}...`);
    const response = await fetch(url, fetchOptions);
    console.log(`[API] Received response from ${url} with status: ${response.status}`);
    
    // Handle non-2xx responses
    if (!response.ok) {
      console.error(`[API] Request failed with status ${response.status} for ${url}`);
      
      const error = await response.json().catch(() => {
        console.error(`[API] Failed to parse error response as JSON for ${url}`);
        return {
          message: `API request failed with status ${response.status}`,
        };
      });
      
      console.error(`[API] Error response:`, error);
      throw new Error(error.message || `API request failed with status ${response.status}`);
    }
    
    // Parse JSON response
    console.log(`[API] Parsing JSON response from ${url}...`);
    const data = await response.json();
    console.log(`[API] Successfully parsed response from ${url}`);
    return data;
  } catch (error) {
    console.error(`[API] Request error for ${url}:`, error);
    throw error;
  }
}

/**
 * Get a list of records from a collection
 * @param {string} collection - Collection name
 * @param {Object} params - Query parameters
 * @returns {Promise<Object>} - Response data
 */
export async function getRecords(collection, params = {}) {
  const queryParams = new URLSearchParams();
  
  // Add query parameters
  Object.entries(params).forEach(([key, value]) => {
    queryParams.append(key, value);
  });
  
  const queryString = queryParams.toString();
  const endpoint = `/collections/${collection}/records${queryString ? `?${queryString}` : ''}`;
  
  return fetchAPI(endpoint);
}

/**
 * Get a single record from a collection
 * @param {string} collection - Collection name
 * @param {string} id - Record ID
 * @returns {Promise<Object>} - Response data
 */
export async function getRecord(collection, id) {
  return fetchAPI(`/collections/${collection}/records/${id}`);
}

/**
 * Create a record in a collection
 * @param {string} collection - Collection name
 * @param {Object} data - Record data
 * @returns {Promise<Object>} - Response data
 */
export async function createRecord(collection, data) {
  return fetchAPI(`/collections/${collection}/records`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * Update a record in a collection
 * @param {string} collection - Collection name
 * @param {string} id - Record ID
 * @param {Object} data - Record data
 * @returns {Promise<Object>} - Response data
 */
export async function updateRecord(collection, id, data) {
  return fetchAPI(`/collections/${collection}/records/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

/**
 * Delete a record from a collection
 * @param {string} collection - Collection name
 * @param {string} id - Record ID
 * @returns {Promise<Object>} - Response data
 */
export async function deleteRecord(collection, id) {
  return fetchAPI(`/collections/${collection}/records/${id}`, {
    method: 'DELETE',
  });
}

/**
 * Authenticate a user with identity (email or username) and password
 * @param {string} identity - User email or username
 * @param {string} password - User password
 * @returns {Promise<Object>} - Response data
 */
export async function authenticateUser(identity, password) {
  console.log(`[AUTH] authenticateUser called with identity: ${identity}`);
  
  try {
    console.log(`[AUTH] Attempting to authenticate user with identity: ${identity}`);
    const response = await fetchAPI('/collections/users/auth-with-password', {
      method: 'POST',
      body: JSON.stringify({ identity, password }),
    });
    
    console.log(`[AUTH] Authentication successful for user: ${identity}`);
    console.log(`[AUTH] User data:`, {
      id: response.record?.id,
      username: response.record?.username,
      email: response.record?.email,
      created: response.record?.created,
      verified: response.record?.verified
    });
    
    return response;
  } catch (error) {
    console.error(`[AUTH] Authentication failed for user: ${identity}`, error);
    throw error;
  }
}

/**
 * Authenticate a user with OAuth2
 * @param {string} provider - OAuth provider (e.g., 'discord', 'github')
 * @returns {Promise<Object>} - Response data
 */
export async function authWithOAuth2(provider) {
  console.log(`[AUTH] authWithOAuth2 called with provider: ${provider}`);
  
  // For OAuth2 authentication, we need to:
  // 1. Get the authorization URL from PocketBase
  // 2. Redirect the user to that URL
  // 3. Handle the callback from the provider
  
  try {
    // Construct the redirect URL
    const redirectUrl = typeof window !== 'undefined' 
      ? `${window.location.origin}/auth-callback`
      : `${process.env.NEXT_PUBLIC_SITE_URL}/auth-callback`;
    
    console.log(`[AUTH] OAuth2 redirect URL: ${redirectUrl}`);
    
    // Construct the query parameters
    const queryParams = new URLSearchParams({
      provider,
      redirectUrl
    }).toString();
    
    console.log(`[AUTH] OAuth2 query parameters: ${queryParams}`);
    
    // Step 1: Get the authorization URL
    console.log(`[AUTH] Requesting OAuth2 authorization URL for provider: ${provider}`);
    const authUrlData = await fetchAPI(`/collections/users/auth-with-oauth2?${queryParams}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    });
    
    console.log(`[AUTH] Received OAuth2 authorization URL:`, authUrlData);
    
    // Step 2: Redirect the user to the authorization URL
    if (typeof window !== 'undefined' && authUrlData.authUrl) {
      console.log(`[AUTH] Redirecting to OAuth2 authorization URL: ${authUrlData.authUrl}`);
      window.location.href = authUrlData.authUrl;
    } else {
      console.log(`[AUTH] Not redirecting to OAuth2 authorization URL (server-side or missing URL)`);
    }
    
    // Return a promise that will never resolve since we're redirecting
    return new Promise(() => {
      console.log(`[AUTH] OAuth2 authentication process initiated, waiting for redirect...`);
    });
  } catch (error) {
    console.error(`[AUTH] OAuth2 authentication failed for provider: ${provider}`, error);
    throw error;
  }
}

/**
 * Refresh authentication token
 * @returns {Promise<Object>} - Response data
 */
export async function refreshAuth() {
  console.log(`[AUTH] refreshAuth called`);
  
  try {
    console.log(`[AUTH] Attempting to refresh authentication token`);
    const response = await fetchAPI('/collections/users/auth-refresh', {
      method: 'POST',
    });
    
    console.log(`[AUTH] Authentication token refreshed successfully`);
    console.log(`[AUTH] User data:`, {
      id: response.record?.id,
      username: response.record?.username,
      email: response.record?.email,
      created: response.record?.created,
      verified: response.record?.verified
    });
    
    return response;
  } catch (error) {
    console.error(`[AUTH] Failed to refresh authentication token`, error);
    throw error;
  }
}

/**
 * Get the current authenticated user
 * @returns {Promise<Object|null>} - User data or null if not authenticated
 */
export async function getCurrentUser() {
  console.log(`[AUTH] getCurrentUser called`);
  
  try {
    console.log(`[AUTH] Attempting to get current user via token refresh`);
    const data = await refreshAuth();
    
    if (data && data.record) {
      console.log(`[AUTH] Successfully retrieved current user:`, {
        id: data.record.id,
        username: data.record.username,
        email: data.record.email,
        created: data.record.created,
        verified: data.record.verified
      });
      return data.record;
    } else {
      console.log(`[AUTH] Token refresh successful but no user record returned`);
      return null;
    }
  } catch (error) {
    console.log(`[AUTH] Failed to get current user, user is not authenticated`, error);
    return null;
  }
}

/**
 * Get user by username
 * @param {string} username - Username to search for
 * @returns {Promise<Object>} - Response data
 */
export async function getUserByUsername(username) {
  // Use case-insensitive comparison for username
  return getRecords('users', {
    filter: `username~"${username}"`,
    limit: 1
  });
}

/**
 * Get schematics by author ID
 * @param {string} authorId - Author ID
 * @param {Object} params - Additional query parameters
 * @returns {Promise<Object>} - Response data
 */
export async function getSchematicsByAuthor(authorId, params = {}) {
  return getRecords('schematics', {
    filter: `author="${authorId}"${params.filter ? ` && ${params.filter}` : ''}`,
    sort: params.sort || '-created',
    expand: params.expand || 'author',
    page: params.page || 1,
    perPage: params.perPage || 12
  });
}

/**
 * Search for schematics
 * @param {string} term - Search term
 * @param {Object} filters - Search filters
 * @returns {Promise<Object>} - Response data
 */
export async function searchSchematics(term, filters = {}) {
  const queryParams = new URLSearchParams();
  
  // Add search term
  if (term) {
    queryParams.append('search', term);
  }
  
  // Add filters
  Object.entries(filters).forEach(([key, value]) => {
    if (value && value !== 'all') {
      queryParams.append(key, value);
    }
  });
  
  const queryString = queryParams.toString();
  const endpoint = `/collections/schematics/records${queryString ? `?${queryString}` : ''}`;
  
  return fetchAPI(endpoint);
}

/**
 * Get categories
 * @returns {Promise<Object>} - Response data
 */
export async function getCategories() {
  return getRecords('schematic_categories', { sort: 'name' });
}

/**
 * Get tags
 * @returns {Promise<Object>} - Response data
 */
export async function getTags() {
  return getRecords('schematic_tags', { sort: 'name' });
}

/**
 * Get Minecraft versions
 * @returns {Promise<Object>} - Response data
 */
export async function getMinecraftVersions() {
  return getRecords('minecraft_versions', { sort: '-version' });
}

/**
 * Get CreateMod versions
 * @returns {Promise<Object>} - Response data
 */
export async function getCreateModVersions() {
  return getRecords('createmod_versions', { sort: '-version' });
}

/**
 * Upload a file to a record
 * @param {string} collection - Collection name
 * @param {string} id - Record ID
 * @param {string} field - Field name
 * @param {File} file - File to upload
 * @returns {Promise<Object>} - Response data
 */
export async function uploadFile(collection, id, field, file) {
  const formData = new FormData();
  formData.append(field, file);
  
  return fetchAPI(`/collections/${collection}/records/${id}`, {
    method: 'PATCH',
    body: formData,
    headers: {
      // Don't set Content-Type here, it will be set automatically with the boundary
    },
  });
}

/**
 * Request password reset
 * @param {string} email - User email
 * @returns {Promise<Object>} - Response data
 */
export async function requestPasswordReset(email) {
  console.log(`[AUTH] requestPasswordReset called for email: ${email}`);
  
  try {
    console.log(`[AUTH] Sending password reset request for email: ${email}`);
    const response = await fetchAPI('/collections/users/request-password-reset', {
      method: 'POST',
      body: JSON.stringify({ email }),
    });
    
    console.log(`[AUTH] Password reset request successful for email: ${email}`);
    return response;
  } catch (error) {
    console.error(`[AUTH] Password reset request failed for email: ${email}`, error);
    throw error;
  }
}

/**
 * Confirm password reset
 * @param {string} token - Reset token
 * @param {string} password - New password
 * @param {string} passwordConfirm - Confirm new password
 * @returns {Promise<Object>} - Response data
 */
export async function confirmPasswordReset(token, password, passwordConfirm) {
  console.log(`[AUTH] confirmPasswordReset called with token: ${token.substring(0, 8)}...`);
  
  try {
    console.log(`[AUTH] Confirming password reset with token: ${token.substring(0, 8)}...`);
    const response = await fetchAPI('/collections/users/confirm-password-reset', {
      method: 'POST',
      body: JSON.stringify({ token, password, passwordConfirm }),
    });
    
    console.log(`[AUTH] Password reset confirmation successful`);
    if (response && response.record) {
      console.log(`[AUTH] User data after password reset:`, {
        id: response.record.id,
        username: response.record.username,
        email: response.record.email
      });
    }
    
    return response;
  } catch (error) {
    console.error(`[AUTH] Password reset confirmation failed`, error);
    throw error;
  }
}

/**
 * Register a new user
 * @param {Object} userData - User data
 * @returns {Promise<Object>} - Response data
 */
export async function registerUser(userData) {
  console.log(`[AUTH] registerUser called for username: ${userData.username}, email: ${userData.email}`);
  
  try {
    console.log(`[AUTH] Attempting to register new user:`, {
      username: userData.username,
      email: userData.email,
      emailVisibility: userData.emailVisibility,
      verified: userData.verified
    });
    
    const response = await createRecord('users', userData);
    
    console.log(`[AUTH] User registration successful:`, {
      id: response.id,
      username: response.username,
      email: response.email,
      created: response.created,
      verified: response.verified
    });
    
    return response;
  } catch (error) {
    console.error(`[AUTH] User registration failed for username: ${userData.username}, email: ${userData.email}`, error);
    throw error;
  }
}

/**
 * Verify user email
 * @param {string} token - Verification token
 * @returns {Promise<Object>} - Response data
 */
export async function verifyEmail(token) {
  console.log(`[AUTH] verifyEmail called with token: ${token.substring(0, 8)}...`);
  
  try {
    console.log(`[AUTH] Attempting to verify email with token: ${token.substring(0, 8)}...`);
    const response = await fetchAPI('/collections/users/confirm-verification', {
      method: 'POST',
      body: JSON.stringify({ token }),
    });
    
    console.log(`[AUTH] Email verification successful`);
    if (response && response.record) {
      console.log(`[AUTH] User data after email verification:`, {
        id: response.record.id,
        username: response.record.username,
        email: response.record.email,
        verified: response.record.verified
      });
    }
    
    return response;
  } catch (error) {
    console.error(`[AUTH] Email verification failed`, error);
    throw error;
  }
}

/**
 * Request email verification
 * @param {string} email - User email
 * @returns {Promise<Object>} - Response data
 */
export async function requestEmailVerification(email) {
  console.log(`[AUTH] requestEmailVerification called for email: ${email}`);
  
  try {
    console.log(`[AUTH] Sending email verification request for email: ${email}`);
    const response = await fetchAPI('/collections/users/request-verification', {
      method: 'POST',
      body: JSON.stringify({ email }),
    });
    
    console.log(`[AUTH] Email verification request successful for email: ${email}`);
    return response;
  } catch (error) {
    console.error(`[AUTH] Email verification request failed for email: ${email}`, error);
    throw error;
  }
}

/**
 * Get schematic by name
 * @param {string} name - Schematic name
 * @returns {Promise<Object>} - Response data
 */
export async function getSchematicByName(name) {
  return getRecords('schematics', {
    filter: `name="${name}"`,
    expand: 'author,categories,tags,createmod_version,minecraft_version',
    limit: 1
  });
}

/**
 * Get comments for a schematic
 * @param {string} schematicId - Schematic ID
 * @returns {Promise<Object>} - Response data
 */
export async function getSchematicComments(schematicId) {
  return getRecords('comments', {
    filter: `schematic="${schematicId}" && approved=true`,
    sort: '-created',
    expand: 'author',
    limit: 1000
  });
}

/**
 * Post a comment
 * @param {Object} commentData - Comment data
 * @returns {Promise<Object>} - Response data
 */
export async function postComment(commentData) {
  return createRecord('comments', commentData);
}

/**
 * Rate a schematic
 * @param {string} schematicId - Schematic ID
 * @param {number} rating - Rating value (1-5)
 * @returns {Promise<Object>} - Response data
 */
export async function rateSchematic(schematicId, rating) {
  return createRecord('schematic_ratings', {
    schematic: schematicId,
    rating: rating,
    rated_at: new Date().toISOString()
  });
}