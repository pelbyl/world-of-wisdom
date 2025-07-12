import { useEffect, useRef, useState } from 'react'

export const useWebSocket = (url: string) => {
  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null)
  const [readyState, setReadyState] = useState<number>(0)
  const ws = useRef<WebSocket | null>(null)

  useEffect(() => {
    const connect = () => {
      try {
        ws.current = new WebSocket(url)

        ws.current.onopen = () => {
          console.log('WebSocket connected')
          setReadyState(1)
        }

        ws.current.onclose = () => {
          console.log('WebSocket disconnected')
          setReadyState(3)
          setTimeout(connect, 3000)
        }

        ws.current.onerror = (error) => {
          console.error('WebSocket error:', error)
          setReadyState(3)
        }

        ws.current.onmessage = (message) => {
          setLastMessage(message)
        }
      } catch (error) {
        console.error('WebSocket connection error:', error)
      }
    }

    connect()

    return () => {
      if (ws.current) {
        ws.current.close()
      }
    }
  }, [url])

  const sendMessage = (message: string) => {
    if (ws.current && ws.current.readyState === 1) {
      ws.current.send(message)
    }
  }

  return { sendMessage, lastMessage, readyState }
}