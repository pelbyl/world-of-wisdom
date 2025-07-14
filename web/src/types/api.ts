// Generated types based on OpenAPI specification
// API Version: v1

// Base API Response wrapper
export interface APIResponse<T = any> {
  status: 'success' | 'error'
  data?: T
  message?: string
}

export interface ErrorResponse extends APIResponse {
  status: 'error'
  message: string
}

// Health endpoint types
export interface HealthData {
  status: 'healthy' | 'degraded' | 'unhealthy'
  miningActive: boolean
  totalBlocks: number
  activeChallenges: number
  liveConnections: number
  algorithm: 'sha256' | 'argon2'
  difficulty: number
}

export type HealthResponse = APIResponse<HealthData>

// Stats endpoint types
export interface MiningStats {
  totalChallenges: number
  completedChallenges: number
  averageSolveTime: number
  currentDifficulty: number
  hashRate: number
}

export interface ConnectionStats {
  total: number
  active: number
}

export interface BlockchainStats {
  blocks: number
  lastBlock: Block | null
}

export interface ChallengeStats {
  active: number
}

export interface SystemStats {
  algorithm: string
  intensity: number
  activeMiners: number
}

export interface StatsData {
  stats: MiningStats
  miningActive: boolean
  connections: ConnectionStats
  blockchain: BlockchainStats
  challenges: ChallengeStats
  system: SystemStats
}

export type StatsResponse = APIResponse<StatsData>

// Blockchain types
export interface Challenge {
  id: string
  seed: string
  difficulty: number
  timestamp: number
  clientId: string
  status: 'solving' | 'completed' | 'failed'
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

export type RecentSolvesResponse = APIResponse<Block[]>

// Logs endpoint types
export interface LogMessage {
  timestamp: number
  level: 'info' | 'success' | 'warning' | 'error'
  message: string
  icon?: string
}

export type LogsResponse = APIResponse<LogMessage[]>

// Challenges endpoint types
export interface ChallengeDetail {
  id: string
  clientId: string
  algorithm: 'sha256' | 'argon2'
  difficulty: number
  seed: string
  status: 'pending' | 'solving' | 'completed' | 'failed' | 'expired'
  createdAt: string
  expiresAt: string
  solvedAt?: string
  solveTimeMs?: number
}

export interface ChallengesData {
  challenges: ChallengeDetail[]
  total: number
}

export type ChallengesResponse = APIResponse<ChallengesData>

// Connections endpoint types
export interface ClientConnection {
  id: string
  clientId: string
  algorithm: 'sha256' | 'argon2'
  remoteAddr: string
  status: 'connected' | 'disconnected' | 'solving' | 'failed'
  connectedAt: string
  disconnectedAt?: string
  challengesAttempted: number
  challengesCompleted: number
  totalSolveTimeMs: number
}

export interface ConnectionsData {
  connections: ClientConnection[]
  active: number
  total: number
}

export type ConnectionsResponse = APIResponse<ConnectionsData>

// Metrics endpoint types
export interface MetricData {
  metricName: string
  value: number
  time: string
  labels?: Record<string, any>
  avgValue?: number
  minValue?: number
  maxValue?: number
}

export interface MetricsData {
  metrics: MetricData[]
}

export type MetricsResponse = APIResponse<MetricsData>

// API Client configuration
export interface APIConfig {
  baseURL: string
  timeout?: number
  headers?: Record<string, string>
}

// API Client methods interface
export interface WordOfWisdomAPI {
  // Health
  getHealth(): Promise<HealthResponse>
  
  // V1 API endpoints
  getStats(): Promise<StatsResponse>
  getChallenges(limit?: number, status?: string, algorithm?: string): Promise<ChallengesResponse>
  getMetrics(metric?: string, interval?: string, start?: string, end?: string): Promise<MetricsResponse>
  getConnections(status?: string): Promise<ConnectionsResponse>
  getLogs(limit?: number): Promise<LogsResponse>
  getRecentSolves(): Promise<RecentSolvesResponse>
}