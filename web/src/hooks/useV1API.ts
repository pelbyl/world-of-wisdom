import { useAPIPolling } from './useAPI'
import { apiClient } from '../api/client'

// V1 API hooks with proper polling intervals as per CLAUDE.md

// Challenges - 2s polling
export const useChallenges = (limit?: number, status?: string, algorithm?: string) => {
  return useAPIPolling(
    () => apiClient.getChallenges(limit, status, algorithm),
    { interval: 2000 } // 2s polling for challenges
  )
}

// Metrics - 5s polling  
export const useMetrics = (metric?: string, interval?: string, start?: string, end?: string) => {
  return useAPIPolling(
    () => apiClient.getMetrics(metric, interval, start, end),
    { interval: 5000 } // 5s polling for metrics
  )
}

// Connections - 2s polling (same as challenges for real-time feel)
export const useConnections = (status?: string) => {
  return useAPIPolling(
    () => apiClient.getConnections(status),
    { interval: 2000 } // 2s polling for connections
  )
}