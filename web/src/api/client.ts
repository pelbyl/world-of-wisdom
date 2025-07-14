import {
  APIConfig,
  WordOfWisdomAPI,
  HealthResponse,
  StatsResponse,
  RecentSolvesResponse,
  LogsResponse,
  ErrorResponse
} from '../types/api'

export class WordOfWisdomAPIClient implements WordOfWisdomAPI {
  private config: APIConfig

  constructor(config: APIConfig) {
    this.config = {
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      ...config
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.config.baseURL}${endpoint}`
    
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout)

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          ...this.config.headers,
          ...options.headers
        },
        signal: controller.signal
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const data = await response.json()
      
      // Check if API returned an error response
      if (data.status === 'error') {
        throw new Error(data.message || 'API returned an error')
      }

      return data as T
    } catch (error) {
      clearTimeout(timeoutId)
      
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new Error(`Request timeout after ${this.config.timeout}ms`)
        }
        throw error
      }
      
      throw new Error('Unknown error occurred')
    }
  }

  // Health endpoint
  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health')
  }

  // Stats endpoint
  async getStats(): Promise<StatsResponse> {
    return this.request<StatsResponse>('/api/v1/stats')
  }

  // Blockchain endpoints
  async getRecentSolves(): Promise<RecentSolvesResponse> {
    return this.request<RecentSolvesResponse>('/api/v1/recent-solves')
  }

  // Logs endpoint
  async getLogs(limit?: number): Promise<LogsResponse> {
    const params = limit ? `?limit=${limit}` : ''
    return this.request<LogsResponse>(`/api/v1/logs${params}`)
  }

  // Challenges endpoint
  async getChallenges(limit?: number, status?: string, algorithm?: string): Promise<any> {
    const params = new URLSearchParams()
    if (limit) params.append('limit', limit.toString())
    if (status) params.append('status', status)
    if (algorithm) params.append('algorithm', algorithm)
    const queryString = params.toString()
    return this.request<any>(`/api/v1/challenges${queryString ? `?${queryString}` : ''}`)
  }

  // Metrics endpoint
  async getMetrics(metric?: string, interval?: string, start?: string, end?: string): Promise<any> {
    const params = new URLSearchParams()
    if (metric) params.append('metric', metric)
    if (interval) params.append('interval', interval)
    if (start) params.append('start', start)
    if (end) params.append('end', end)
    const queryString = params.toString()
    return this.request<any>(`/api/v1/metrics${queryString ? `?${queryString}` : ''}`)
  }

  // Connections endpoint
  async getConnections(status?: string): Promise<any> {
    const params = status ? `?status=${status}` : ''
    return this.request<any>(`/api/v1/connections${params}`)
  }
}

import config from '../config'

// Default API client instance
export const apiClient = new WordOfWisdomAPIClient({
  baseURL: config.api.baseURL
})

// Error handling utilities
export class APIError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: ErrorResponse
  ) {
    super(message)
    this.name = 'APIError'
  }
}

export const handleAPIError = (error: unknown): APIError => {
  if (error instanceof APIError) {
    return error
  }
  
  if (error instanceof Error) {
    return new APIError(error.message)
  }
  
  return new APIError('Unknown API error occurred')
}