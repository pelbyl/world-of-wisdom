// Get API base URL from environment or default to relative URL
const getAPIBaseURL = () => {
  // In production builds, VITE_API_URL is embedded at build time
  if (import.meta.env.VITE_API_URL && import.meta.env.VITE_API_URL.trim() !== '') {
    return import.meta.env.VITE_API_URL;
  }
  
  // Always use relative URL - nginx will proxy to API server container
  // This ensures requests go through the nginx proxy instead of trying to reach port 8081 directly
  return '';
};

export const config = {
  api: {
    baseURL: getAPIBaseURL(),
    timeout: 10000,
  },
  app: {
    name: 'World of Wisdom',
    version: '1.0.0',
  },
  development: {
    enablePolling: true,
    pollingInterval: 2000,
  },
}

export default config 