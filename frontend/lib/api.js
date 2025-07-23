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
  
  try {
    // Handle server-side rendering
    if (typeof window === 'undefined') {
      // For server-side, ensure we have a complete URL with the /api prefix
      const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
      
      // Create a URL object with a base URL to ensure proper URL construction
      const base = new URL(baseUrl);
      
      // Remove the endpoint's leading slash if present
      const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
      
      // Ensure the path includes /api/ prefix
      const apiPath = '/api/';
      
      // Construct the full URL
      url = new URL(apiPath + cleanEndpoint, base).toString();
    } else {
      // For client-side, use relative URL with /api prefix
      // Ensure the API path has a trailing slash for proper URL joining
      const apiPath = '/api/';
      
      // Remove the endpoint's leading slash if present
      const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
      
      // Construct the full URL
      url = apiPath + cleanEndpoint;
    }
  } catch (error) {
    console.error('Error constructing URL:', error);
    throw new Error(`Invalid URL construction: ${endpoint}`);
  }
  
  // Log the constructed URL for debugging
  console.log(`API Request URL: ${url}`);
  
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
  
  try {
    const response = await fetch(url, fetchOptions);
    
    // Handle non-2xx responses
    if (!response.ok) {
      const error = await response.json().catch(() => ({
        message: `API request failed with status ${response.status}`,
      }));
      
      throw new Error(error.message || `API request failed with status ${response.status}`);
    }
    
    // Parse JSON response
    return await response.json();
  } catch (error) {
    console.error(`API request error for ${url}:`, error);
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
 * Authenticate a user with email and password
 * @param {string} email - User email
 * @param {string} password - User password
 * @returns {Promise<Object>} - Response data
 */
export async function authenticateUser(email, password) {
  return fetchAPI('/collections/users/auth-with-password', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

/**
 * Authenticate a user with OAuth2
 * @param {string} provider - OAuth provider (e.g., 'discord', 'github')
 * @returns {Promise<Object>} - Response data
 */
export async function authWithOAuth2(provider) {
  // For OAuth2 authentication, we need to:
  // 1. Get the authorization URL from PocketBase
  // 2. Redirect the user to that URL
  // 3. Handle the callback from the provider
  
  // Construct the redirect URL
  const redirectUrl = typeof window !== 'undefined' 
    ? `${window.location.origin}/auth-callback`
    : `${process.env.NEXT_PUBLIC_SITE_URL}/auth-callback`;
  
  // Construct the query parameters
  const queryParams = new URLSearchParams({
    provider,
    redirectUrl
  }).toString();
  
  // Step 1: Get the authorization URL
  const authUrlData = await fetchAPI(`/collections/users/auth-with-oauth2?${queryParams}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    }
  });
  
  // Step 2: Redirect the user to the authorization URL
  if (typeof window !== 'undefined' && authUrlData.authUrl) {
    window.location.href = authUrlData.authUrl;
  }
  
  // Return a promise that will never resolve since we're redirecting
  return new Promise(() => {});
}

/**
 * Refresh authentication token
 * @returns {Promise<Object>} - Response data
 */
export async function refreshAuth() {
  return fetchAPI('/collections/users/auth-refresh', {
    method: 'POST',
  });
}

/**
 * Get the current authenticated user
 * @returns {Promise<Object|null>} - User data or null if not authenticated
 */
export async function getCurrentUser() {
  try {
    const data = await refreshAuth();
    return data.record;
  } catch (error) {
    return null;
  }
}

/**
 * Get user by username
 * @param {string} username - Username to search for
 * @returns {Promise<Object>} - Response data
 */
export async function getUserByUsername(username) {
  return getRecords('users', {
    filter: `username="${username}"`,
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
  return fetchAPI('/collections/users/request-password-reset', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}

/**
 * Confirm password reset
 * @param {string} token - Reset token
 * @param {string} password - New password
 * @param {string} passwordConfirm - Confirm new password
 * @returns {Promise<Object>} - Response data
 */
export async function confirmPasswordReset(token, password, passwordConfirm) {
  return fetchAPI('/collections/users/confirm-password-reset', {
    method: 'POST',
    body: JSON.stringify({ token, password, passwordConfirm }),
  });
}

/**
 * Register a new user
 * @param {Object} userData - User data
 * @returns {Promise<Object>} - Response data
 */
export async function registerUser(userData) {
  return createRecord('users', userData);
}

/**
 * Verify user email
 * @param {string} token - Verification token
 * @returns {Promise<Object>} - Response data
 */
export async function verifyEmail(token) {
  return fetchAPI('/collections/users/confirm-verification', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
}

/**
 * Request email verification
 * @param {string} email - User email
 * @returns {Promise<Object>} - Response data
 */
export async function requestEmailVerification(email) {
  return fetchAPI('/collections/users/request-verification', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}

/**
 * Get schematic by name
 * @param {string} name - Schematic name
 * @returns {Promise<Object>} - Response data
 */
export async function getSchematicByName(name) {
  return getRecords('schematics', {
    filter: `name="${name}" && deleted=null`,
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