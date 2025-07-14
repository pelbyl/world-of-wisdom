// WebSocket-only communication - no REST API endpoints needed
export interface LogResponse {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  icon?: string;
  metadata?: any;
  created_at: string;
}

export const apiClient = {
  async getLogs(): Promise<LogResponse[]> {
    // Logs are now served via WebSocket real-time connection
    console.info('Logs are served via WebSocket - check real-time updates');
    return [];
  },

  async getLogsByLevel(): Promise<LogResponse[]> {
    // Logs are now served via WebSocket real-time connection
    console.info('Logs are served via WebSocket - check real-time updates');
    return [];
  },

  async createLog(): Promise<LogResponse | null> {
    // Logs are now served via WebSocket real-time connection
    console.info('Logs are served via WebSocket - check real-time updates');
    return null;
  },

  async getRecentLogs(): Promise<LogResponse[]> {
    // Logs are now served via WebSocket real-time connection
    console.info('Logs are served via WebSocket - check real-time updates');
    return [];
  }
};

// Convert API log response to frontend LogMessage format
export function convertApiLogToLogMessage(apiLog: LogResponse): import('../types').LogMessage {
  return {
    timestamp: new Date(apiLog.timestamp).getTime(),
    level: apiLog.level as 'info' | 'success' | 'warning' | 'error',
    message: apiLog.message,
    icon: apiLog.icon || '',
  };
}