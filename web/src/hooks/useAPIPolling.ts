import { useEffect, useRef, useState, useCallback } from 'react'

interface PollingOptions {
  interval?: number // polling interval in milliseconds (default: 2000)
  enabled?: boolean // whether polling is enabled (default: true)
}

interface APIState {
  data: any
  loading: boolean
  error: string | null
  lastUpdated: number | null
}

export const useAPIPolling = (endpoint: string, options: PollingOptions = {}) => {
  const { interval = 2000, enabled = true } = options
  const [state, setState] = useState<APIState>({
    data: null,
    loading: true,
    error: null,
    lastUpdated: null
  })
  
  const intervalRef = useRef<number>()
  const abortControllerRef = useRef<AbortController>()

  const fetchData = useCallback(async () => {
    if (!enabled) return

    try {
      // Cancel any ongoing request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }

      // Create new abort controller
      abortControllerRef.current = new AbortController()

      const response = await fetch(endpoint, {
        signal: abortControllerRef.current.signal,
        headers: {
          'Accept': 'application/json',
        }
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const result = await response.json()
      
      setState(prev => ({
        ...prev,
        data: result,
        loading: false,
        error: null,
        lastUpdated: Date.now()
      }))
    } catch (error: any) {
      // Don't update state if request was aborted
      if (error.name === 'AbortError') {
        return
      }

      console.error(`API polling error for ${endpoint}:`, error)
      setState(prev => ({
        ...prev,
        loading: false,
        error: error.message || 'Failed to fetch data',
        lastUpdated: Date.now()
      }))
    }
  }, [endpoint, enabled])

  // Start polling
  useEffect(() => {
    if (!enabled) {
      setState(prev => ({ ...prev, loading: false }))
      return
    }

    // Initial fetch
    fetchData()

    // Set up interval
    intervalRef.current = setInterval(fetchData, interval)

    return () => {
      // Cleanup
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [fetchData, interval, enabled])

  const refresh = useCallback(() => {
    setState(prev => ({ ...prev, loading: true, error: null }))
    fetchData()
  }, [fetchData])

  return {
    ...state,
    refresh
  }
}

// Specialized hooks for different endpoints (v1 API)
import config from '../config'

export const useStatsPolling = (options?: PollingOptions) => {
  return useAPIPolling(`${config.api.baseURL}/stats`, options)
}

export const useHealthPolling = (options?: PollingOptions) => {
  return useAPIPolling(`${config.api.baseURL}/health`, options)
}

export const useLogsPolling = (options?: PollingOptions) => {
  return useAPIPolling(`${config.api.baseURL}/logs`, options)
}

export const useRecentSolvesPolling = (options?: PollingOptions) => {
  return useAPIPolling(`${config.api.baseURL}/recent-solves`, options)
}