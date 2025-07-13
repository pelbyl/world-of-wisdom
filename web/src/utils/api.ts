const API_BASE_URL = 'http://localhost:8082/api/v1';

export interface ApiResponse<T> {
  data: T;
  timestamp: string;
}

export interface LogResponse {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  icon?: string;
  metadata?: any;
  created_at: string;
}

export interface LogsListResponse {
  data: LogResponse[];
  timestamp: string;
}

export const apiClient = {
  async getLogs(limit: number = 100): Promise<LogResponse[]> {
    try {
      const response = await fetch(`${API_BASE_URL}/logs?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result: LogsListResponse = await response.json();
      return result.data || [];
    } catch (error) {
      console.error('Failed to fetch logs:', error);
      return [];
    }
  },

  async getLogsByLevel(level: string, limit: number = 100): Promise<LogResponse[]> {
    try {
      const response = await fetch(`${API_BASE_URL}/logs/level/${level}?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result: LogsListResponse = await response.json();
      return result.data || [];
    } catch (error) {
      console.error(`Failed to fetch logs for level ${level}:`, error);
      return [];
    }
  },

  async createLog(log: { level: string; message: string; icon?: string; metadata?: any }): Promise<LogResponse | null> {
    try {
      const response = await fetch(`${API_BASE_URL}/logs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(log),
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result: ApiResponse<LogResponse> = await response.json();
      return result.data;
    } catch (error) {
      console.error('Failed to create log:', error);
      return null;
    }
  },

  async getRecentLogs(limit: number = 50): Promise<LogResponse[]> {
    return this.getLogs(limit);
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