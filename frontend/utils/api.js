/**
 * API utility for communicating with the PocketBase backend
 */

// Base URL for API requests
const API_BASE_URL = '/api';

/**
 * Fetch data from the API with error handling
 * @param {string} endpoint - API endpoint to fetch from
 * @param {Object} options - Fetch options
 * @returns {Promise<Object>} - Response data
 */
export async function fetchAPI(endpoint, options = {}) {
  const url = `${API_BASE_URL}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`;
  
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
 * Get categories
 * @returns {Promise<Array>} - Categories
 */
export async function getCategories() {
  const data = await getRecords('schematic_categories', { sort: 'name' });
  return data.items || [];
}

/**
 * Get tags
 * @returns {Promise<Array>} - Tags
 */
export async function getTags() {
  const data = await getRecords('schematic_tags', { sort: 'name' });
  return data.items || [];
}

/**
 * Get Minecraft versions
 * @returns {Promise<Array>} - Minecraft versions
 */
export async function getMinecraftVersions() {
  const data = await getRecords('minecraft_versions', { sort: '-version' });
  return data.items || [];
}

/**
 * Get CreateMod versions
 * @returns {Promise<Array>} - CreateMod versions
 */
export async function getCreateModVersions() {
  const data = await getRecords('createmod_versions', { sort: '-version' });
  return data.items || [];
}