import { useEffect, useRef, useState, useCallback } from 'react'

interface WebSocketState {
  isConnected: boolean
  isConnecting: boolean
  isError: boolean
  reconnectAttempts: number
  lastError?: string
  isHighLoad?: boolean
  degradedMode?: boolean
}

export const useWebSocket = (url: string) => {
  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null)
  const [readyState, setReadyState] = useState<number>(0)
  const [connectionState, setConnectionState] = useState<WebSocketState>({
    isConnected: false,
    isConnecting: false,
    isError: false,
    reconnectAttempts: 0,
    isHighLoad: false,
    degradedMode: false
  })
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const degradedModeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000
  const degradedModeDelay = 10000 // Switch to degraded mode after 10 seconds of failures

  const connect = useCallback(() => {
    if (connectionState.reconnectAttempts >= maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      setConnectionState(prev => ({
        ...prev,
        isConnecting: false,
        isError: true,
        lastError: 'Max reconnection attempts reached'
      }))
      return
    }

    setConnectionState(prev => ({
      ...prev,
      isConnecting: true,
      isError: false,
      lastError: undefined
    }))

    try {
      // Close existing connection if any
      if (ws.current && ws.current.readyState === WebSocket.OPEN) {
        ws.current.close()
      }

      ws.current = new WebSocket(url)
      setReadyState(0) // CONNECTING

      ws.current.onopen = () => {
        console.log('WebSocket connected successfully')
        setReadyState(1) // OPEN
        
        // Clear degraded mode timer if connection successful
        if (degradedModeTimeoutRef.current) {
          clearTimeout(degradedModeTimeoutRef.current)
          degradedModeTimeoutRef.current = null
        }
        
        setConnectionState({
          isConnected: true,
          isConnecting: false,
          isError: false,
          reconnectAttempts: 0,
          degradedMode: false,
          isHighLoad: false
        })
      }

      ws.current.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason)
        setReadyState(3) // CLOSED
        setConnectionState(prev => ({
          ...prev,
          isConnected: false,
          isConnecting: false
        }))

        // Start degraded mode timer if repeated failures
        if (connectionState.reconnectAttempts >= 3 && !degradedModeTimeoutRef.current) {
          degradedModeTimeoutRef.current = setTimeout(() => {
            console.log('🔥 Switching to degraded mode - using API polling instead of WebSocket')
            setConnectionState(prev => ({
              ...prev,
              degradedMode: true,
              isHighLoad: true
            }))
          }, degradedModeDelay)
        }

        // Only attempt to reconnect if it wasn't a clean close
        if (event.code !== 1000 && connectionState.reconnectAttempts < maxReconnectAttempts) {
          const delay = Math.min(baseReconnectDelay * Math.pow(1.5, connectionState.reconnectAttempts), 30000) // Cap at 30s
          console.log(`Attempting to reconnect in ${delay}ms (attempt ${connectionState.reconnectAttempts + 1}/${maxReconnectAttempts})`)
          
          reconnectTimeoutRef.current = setTimeout(() => {
            setConnectionState(prevState => ({
              ...prevState,
              reconnectAttempts: prevState.reconnectAttempts + 1
            }))
            connect()
          }, delay)
        } else if (connectionState.reconnectAttempts >= maxReconnectAttempts) {
          console.log('🔥 Max reconnection attempts reached - switching to API-only mode')
          setConnectionState(prev => ({
            ...prev,
            degradedMode: true,
            isError: true
          }))
        }
      }

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error)
        setReadyState(3) // CLOSED
        setConnectionState(prev => ({
          ...prev,
          isConnected: false,
          isConnecting: false,
          isError: true,
          lastError: 'Connection error occurred'
        }))
      }

      ws.current.onmessage = (message) => {
        setLastMessage(message)
      }
    } catch (error) {
      console.error('WebSocket connection error:', error)
      setConnectionState(prev => ({
        ...prev,
        isConnecting: false,
        isError: true,
        lastError: `Failed to create WebSocket: ${error}`
      }))
    }
  }, [url, connectionState.reconnectAttempts])

  useEffect(() => {
    connect()

    return () => {
      // Clear all timeouts
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (degradedModeTimeoutRef.current) {
        clearTimeout(degradedModeTimeoutRef.current)
      }
      
      // Close WebSocket connection
      if (ws.current) {
        ws.current.close(1000, 'Component unmounting')
      }
    }
  }, [connect])

  const sendMessage = useCallback((message: string) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      try {
        ws.current.send(message)
        return true
      } catch (error) {
        console.error('Failed to send message:', error)
        return false
      }
    }
    console.warn('WebSocket is not connected. Message not sent:', message)
    return false
  }, [])

  const forceReconnect = useCallback(() => {
    setConnectionState({
      isConnected: false,
      isConnecting: false,
      isError: false,
      reconnectAttempts: 0
    })
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    
    if (ws.current) {
      ws.current.close(1000, 'Manual reconnection')
    }
    
    // Start fresh connection
    setTimeout(connect, 100)
  }, [connect])

  return { 
    sendMessage, 
    lastMessage, 
    readyState, 
    connectionState,
    forceReconnect
  }
}