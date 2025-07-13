import React, { useState, useEffect } from 'react'

interface Props {
  miningActive: boolean
  onStartDemo: () => void
  disabled?: boolean
}

const DemoModeButton: React.FC<Props> = ({ miningActive, onStartDemo, disabled = false }) => {
  const [isStarting, setIsStarting] = useState(false)
  const [progress, setProgress] = useState(0)
  const [timeRemaining, setTimeRemaining] = useState(0)
  const [demoStartTime, setDemoStartTime] = useState<number | null>(null)

  // Demo duration in seconds
  const DEMO_DURATION = 60

  useEffect(() => {
    if (miningActive && demoStartTime) {
      const interval = setInterval(() => {
        const elapsed = (Date.now() - demoStartTime) / 1000
        const remaining = Math.max(0, DEMO_DURATION - elapsed)
        const progressPercent = Math.min(100, (elapsed / DEMO_DURATION) * 100)
        
        setTimeRemaining(remaining)
        setProgress(progressPercent)
        
        if (remaining <= 0) {
          setDemoStartTime(null)
          setProgress(0)
          setTimeRemaining(0)
        }
      }, 100)

      return () => clearInterval(interval)
    } else if (!miningActive) {
      setDemoStartTime(null)
      setProgress(0)
      setTimeRemaining(0)
    }
  }, [miningActive, demoStartTime])

  const handleDemoStart = async () => {
    setIsStarting(true)
    setDemoStartTime(Date.now())
    
    // Add a small delay for better UX
    setTimeout(() => {
      onStartDemo()
      setIsStarting(false)
    }, 500)
  }

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const isDisabled = disabled || miningActive || isStarting

  return (
    <div className="relative w-full">
      <button
        onClick={handleDemoStart}
        disabled={isDisabled}
        className={`
          relative overflow-hidden w-full px-4 py-2 rounded-lg font-semibold text-white transition-all duration-300 transform text-sm
          ${isDisabled 
            ? 'bg-gray-400 cursor-not-allowed' 
            : 'bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 hover:scale-105 shadow-lg hover:shadow-xl'
          }
          ${isStarting ? 'animate-pulse' : ''}
        `}
        title={miningActive ? `Demo running - ${formatTime(timeRemaining)} remaining` : 'Start a 60-second demo with automatic mining'}
      >
        {/* Progress bar background */}
        {miningActive && demoStartTime && (
          <div 
            className="absolute inset-0 bg-gradient-to-r from-green-400 to-blue-500 transition-all duration-100"
            style={{ width: `${progress}%` }}
          />
        )}
        
        {/* Button content */}
        <div className="relative z-10 flex items-center space-x-2">
          {isStarting && (
            <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
          )}
          
          <span className="flex items-center space-x-1">
            <span>🚀</span>
            <span>
              {isStarting 
                ? 'Initializing...' 
                : miningActive && demoStartTime
                  ? `Demo (${formatTime(timeRemaining)})`
                  : 'Demo Mode'
              }
            </span>
          </span>
          
          {miningActive && demoStartTime && (
            <span className="text-xs opacity-90">
              {progress.toFixed(0)}%
            </span>
          )}
        </div>
      </button>
      
      {/* Demo info panel */}
      {!miningActive && !isStarting && (
        <div className="absolute top-full left-0 mt-2 p-3 bg-white rounded-lg shadow-lg border z-20 w-64 text-sm text-gray-700">
          <h4 className="font-semibold text-purple-600 mb-2">🎯 Demo Mode</h4>
          <ul className="space-y-1 text-xs">
            <li>• 60-second automated mining session</li>
            <li>• Optimized difficulty for quick results</li>
            <li>• Real-time progress tracking</li>
            <li>• Perfect for testing the system</li>
          </ul>
          <div className="mt-2 pt-2 border-t border-gray-200">
            <span className="text-xs text-gray-500">Click to start immediately</span>
          </div>
        </div>
      )}
    </div>
  )
}

export default DemoModeButton