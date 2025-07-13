import React from 'react'

interface ConnectionState {
  isConnected: boolean
  isConnecting: boolean
  isError: boolean
  reconnectAttempts: number
  lastError?: string
  isHighLoad?: boolean
  degradedMode?: boolean
}

interface Props {
  readyState: number
  connectionState: ConnectionState
  onReconnect: () => void
}

const ConnectionStatus: React.FC<Props> = ({ readyState, connectionState, onReconnect }) => {
  const getStatusInfo = () => {
    if (connectionState.degradedMode) {
      return {
        text: 'API Mode (High Load)',
        color: 'text-orange-600',
        bgColor: 'bg-orange-100',
        icon: '🔥'
      }
    }
    
    if (connectionState.isConnected && readyState === 1) {
      return {
        text: connectionState.isHighLoad ? 'Connected (High Load)' : 'Connected',
        color: connectionState.isHighLoad ? 'text-yellow-600' : 'text-green-600',
        bgColor: connectionState.isHighLoad ? 'bg-yellow-100' : 'bg-green-100',
        icon: connectionState.isHighLoad ? '🟡' : '🟢'
      }
    }
    
    if (connectionState.isConnecting) {
      return {
        text: `Connecting${connectionState.reconnectAttempts > 0 ? ` (attempt ${connectionState.reconnectAttempts})` : ''}`,
        color: 'text-yellow-600',
        bgColor: 'bg-yellow-100',
        icon: '🟡'
      }
    }
    
    if (connectionState.isError) {
      return {
        text: 'Connection Error',
        color: 'text-red-600',
        bgColor: 'bg-red-100',
        icon: '🔴'
      }
    }
    
    return {
      text: 'Disconnected',
      color: 'text-gray-600',
      bgColor: 'bg-gray-100',
      icon: '⚫'
    }
  }

  const status = getStatusInfo()
  const showReconnectButton = !connectionState.isConnected && !connectionState.isConnecting

  return (
    <div className={`inline-flex items-center space-x-2 px-3 py-1 rounded-full ${status.bgColor}`}>
      <span className="text-sm">{status.icon}</span>
      <span className={`text-sm font-medium ${status.color}`}>
        {status.text}
      </span>
      
      {showReconnectButton && (
        <button
          onClick={onReconnect}
          className="ml-2 px-2 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors"
          title="Click to reconnect"
        >
          Reconnect
        </button>
      )}
      
      {connectionState.lastError && (
        <span className="text-xs text-red-500 ml-2" title={connectionState.lastError}>
          ⚠️
        </span>
      )}
    </div>
  )
}

export default ConnectionStatus