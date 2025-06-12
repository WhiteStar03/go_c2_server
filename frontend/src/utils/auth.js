import API_CONFIG from '../config/api';

// Utility function to handle authentication errors consistently
export const handleAuthError = (response, navigate, displayNotification) => {
  if (response.status === 401 || response.status === 403) {
    localStorage.removeItem("token");
    displayNotification("Session expired. Please log in again.", "error", 3000);
    setTimeout(() => navigate('/login'), 3000);
    return true; // Indicates auth error was handled
  }
  return false; // No auth error
};

// Utility function to make authenticated requests
export const makeAuthenticatedRequest = async (url, options = {}) => {
  const token = localStorage.getItem("token");
  
  if (!token) {
    throw new Error("No authentication token found");
  }

  // If URL is relative, prepend with API base URL
  const fullUrl = url.startsWith('http') ? url : url;

  const defaultHeaders = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  const mergedOptions = {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
  };

  return fetch(fullUrl, mergedOptions);
};

// Check if user is authenticated
export const isAuthenticated = () => {
  return !!localStorage.getItem("token");
};
