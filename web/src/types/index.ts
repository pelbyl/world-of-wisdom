export interface Challenge {
  id: string
  seed: string
  difficulty: number
  timestamp: number
  clientId: string
  status: 'pending' | 'solving' | 'completed' | 'failed'
}

export interface Solution {
  challengeId: string
  nonce: string
  hash: string
  attempts: number
  timeToSolve: number
  timestamp: number
}

export interface Block {
  index: number
  timestamp: number
  challenge: Challenge
  solution?: Solution
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

export interface ClientConnection {
  id: string
  address: string
  connectedAt: number
  status: 'connected' | 'solving' | 'disconnected'
  challengesCompleted: number
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