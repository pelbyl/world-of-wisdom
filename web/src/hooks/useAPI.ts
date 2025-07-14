import { useEffect, useState, useCallback } from 'react'
import { apiClient, handleAPIError } from '../api/client'

interface APIState<T> {
  data: T | null
  loading: boolean
  error: string | null
  lastUpdated: number | null
}

interface UseAPIOptions {
  interval?: number // polling interval in milliseconds
  enabled?: boolean // whether to start polling immediately
}

// Generic hook for API polling
export function useAPIPolling<T>(
  apiCall: () => Promise<T>,
  options: UseAPIOptions = {}
): APIState<T> & { refresh: () => void } {
  const { interval = 2000, enabled = true } = options
  const [state, setState] = useState<APIState<T>>({
    data: null,
    loading: true,
    error: null,
    lastUpdated: null
  })

  const fetchData = useCallback(async () => {
    try {
      const response = await apiCall()
      setState(prev => ({
        ...prev,
        data: response,
        loading: false,
        error: null,
        lastUpdated: Date.now()
      }))
    } catch (error) {
      const apiError = handleAPIError(error)
      setState(prev => ({
        ...prev,
        loading: false,
        error: apiError.message,
        lastUpdated: Date.now()
      }))
    }
  }, [apiCall])

  const refresh = useCallback(() => {
    setState(prev => ({ ...prev, loading: true, error: null }))
    fetchData()
  }, [fetchData])

  useEffect(() => {
    if (!enabled) {
      setState(prev => ({ ...prev, loading: false }))
      return
    }

    // Initial fetch
    fetchData()

    // Set up polling
    const intervalId = setInterval(fetchData, interval)

    return () => {
      clearInterval(intervalId)
    }
  }, [fetchData, interval, enabled])

  return { ...state, refresh }
}

// Specific API hooks
export const useHealth = (options?: UseAPIOptions) => {
  return useAPIPolling(() => apiClient.getHealth(), options)
}

export const useStats = (options?: UseAPIOptions) => {
  return useAPIPolling(() => apiClient.getStats(), options)
}

export const useRecentSolves = (options?: UseAPIOptions) => {
  return useAPIPolling(() => apiClient.getRecentSolves(), options)
}

export const useLogs = (limit?: number, options?: UseAPIOptions) => {
  return useAPIPolling(() => apiClient.getLogs(limit), options)
}

// Utility hook for one-time API calls
export const useAPICall = <T>(apiCall: () => Promise<T>) => {
  const [state, setState] = useState<APIState<T>>({
    data: null,
    loading: false,
    error: null,
    lastUpdated: null
  })

  const execute = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }))
    try {
      const response = await apiCall()
      setState({
        data: response,
        loading: false,
        error: null,
        lastUpdated: Date.now()
      })
      return response
    } catch (error) {
      const apiError = handleAPIError(error)
      setState({
        data: null,
        loading: false,
        error: apiError.message,
        lastUpdated: Date.now()
      })
      throw apiError
    }
  }, [apiCall])

  return { ...state, execute }
}