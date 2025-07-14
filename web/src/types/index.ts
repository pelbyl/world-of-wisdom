export interface Block {
  index: number
  timestamp: number
  challenge: any
  solution?: any
  quote?: string
  previousHash: string
  hash: string
}

export interface MiningStats {
  totalChallenges: number
  completedChallenges: number
  averageSolveTime: number
  currentDifficulty: number
  hashRate: number
}

export interface MetricsData {
  timestamp: number
  connectionsTotal: number
  currentDifficulty: number
  puzzlesSolvedTotal: number
  puzzlesFailedTotal: number
  averageSolveTime: number
  connectionRate: number
  difficultyAdjustments: number
  activeConnections: number
}

export interface LogMessage {
  timestamp: number
  level: 'info' | 'success' | 'warning' | 'error'
  message: string
  icon?: string
}