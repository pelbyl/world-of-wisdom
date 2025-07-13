import { MantineProvider, Container, Title, Grid, Paper, Group, Badge, Stack, Text } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { useEffect, useState, useCallback } from 'react'
import { BlockchainVisualizer } from './components/BlockchainVisualizer'
import { MiningVisualizer } from './components/MiningVisualizer'
import { ConnectionsPanel } from './components/ConnectionsPanel'
import { StatsPanel } from './components/StatsPanel'
import { MiningConfigPanel } from './components/MiningConfigPanel'
import { MetricsDashboard } from './components/MetricsDashboard'
import { LogsPanel } from './components/LogsPanel'
import ConnectionStatus from './components/ConnectionStatus'
import { useWebSocket } from './hooks/useWebSocket'
// Removed localStorage persistence - service is now stateless
import { Block, Challenge, ClientConnection, MiningStats, MetricsData, LogMessage } from './types'
import { apiClient, convertApiLogToLogMessage } from './utils/api'

function App() {
  // Stateless initialization - all data comes from server
  const [blocks, setBlocks] = useState<Block[]>([])
  const [currentChallenges, setCurrentChallenges] = useState<Challenge[]>([])
  const [connections, setConnections] = useState<ClientConnection[]>([])
  const [stats, setStats] = useState<MiningStats>({
    totalChallenges: 0,
    completedChallenges: 0,
    averageSolveTime: 0,
    currentDifficulty: 2,
    hashRate: 0,
    liveConnections: 0,
    totalConnections: 0,
    networkIntensity: 1,
    ddosProtectionActive: false,
    activeMinerCount: 0,
  })
  const [metrics, setMetrics] = useState<MetricsData | null>(null)
  const [miningActive, setMiningActive] = useState(false)
  const [logs, setLogs] = useState<LogMessage[]>([])
  const [isRecovering, setIsRecovering] = useState(false)

  const { sendMessage, lastMessage, readyState, connectionState, forceReconnect } = useWebSocket('ws://localhost:8081/ws')

  // Fetch logs from API
  const fetchLogs = useCallback(async () => {
    try {
      const apiLogs = await apiClient.getRecentLogs(100)
      const logs = apiLogs.map(convertApiLogToLogMessage)
      setLogs(logs)
    } catch (error) {
      console.error('Failed to fetch logs from API:', error)
    }
  }, [])

  // Initial data load and adaptive polling based on connection state
  useEffect(() => {
    fetchLogs()
    
    // Adaptive polling: faster during degraded mode, slower when WebSocket is working
    const pollInterval = connectionState.degradedMode || connectionState.isError ? 5000 : 30000
    const interval = setInterval(fetchLogs, pollInterval)
    
    return () => clearInterval(interval)
  }, [fetchLogs, connectionState.degradedMode, connectionState.isError])

  useEffect(() => {
    if (lastMessage) {
      try {
        const data = JSON.parse(lastMessage.data)

        switch (data.type) {
          case 'block':
            setBlocks(prev => [...prev, data.block])
            break
          case 'challenge':
            setCurrentChallenges(prev => [...prev, data.challenge])
            break
          case 'challenge_update':
            setCurrentChallenges(prev =>
              prev.map(c => c.id === data.challenge.id ? data.challenge : c)
            )
            break
          case 'connection':
            setConnections(prev => {
              const existing = prev.find(c => c.id === data.connection.id)
              if (existing) {
                return prev.map(c => c.id === data.connection.id ? data.connection : c)
              }
              return [...prev, data.connection]
            })
            break
          case 'stats':
            setStats(data.stats)
            break
          case 'metrics':
            setMetrics(data.metrics)
            break
          case 'mining_status':
            setMiningActive(data.miningActive || false)
            // If mining stopped unexpectedly, ensure UI is responsive
            if (!data.miningActive) {
              setIsRecovering(false)
            }
            break
          case 'log':
            if (data.log) {
              // Add new log immediately for real-time feedback
              setLogs(prev => [...prev.slice(-99), data.log]) // Keep last 100 logs
              // Note: Database is the source of truth, periodic refresh will sync any differences
            }
            break
          case 'init':
            // Stateless: Always use server data as the source of truth
            setBlocks(data.blocks || [])
            setConnections(data.connections || [])
            setCurrentChallenges(data.challenges || [])
            // Fetch logs from API instead of WebSocket to ensure database consistency
            fetchLogs()
            if (data.stats) {
              setStats(data.stats)
            }
            setMiningActive(data.miningActive || false)
            setIsRecovering(false)
            break
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error)
      }
    }
  }, [lastMessage])

  // Stateless service - no persistence needed

  // Handle connection recovery
  useEffect(() => {
    if (connectionState.isConnected && isRecovering) {
      // Request current state from server after reconnection
      sendMessage(JSON.stringify({ type: 'get_state' }))
    }
  }, [connectionState.isConnected, isRecovering, sendMessage])

  // Detect when we need to recover state
  useEffect(() => {
    if (connectionState.isError || (!connectionState.isConnected && connectionState.reconnectAttempts > 0)) {
      setIsRecovering(true)
    }
  }, [connectionState])

  const handleStartMining = useCallback((config?: any) => {
    const success = sendMessage(config
      ? JSON.stringify({ type: 'start_mining', config })
      : JSON.stringify({ type: 'start_mining' }))
    
    if (!success && !connectionState.isConnected) {
      // If WebSocket is down, show user feedback
      console.warn('Cannot start mining: WebSocket not connected')
    }
  }, [sendMessage, connectionState.isConnected])

  const handleStopMining = useCallback(() => {
    const success = sendMessage(JSON.stringify({ type: 'stop_mining' }))
    
    if (!success && miningActive) {
      // Force-stop locally if WebSocket is down
      setMiningActive(false)
      console.warn('WebSocket disconnected - forcing local stop')
    }
  }, [sendMessage, miningActive])

  // Emergency state reset function
  const handleEmergencyReset = useCallback(() => {
    setMiningActive(false)
    setIsRecovering(false)
    forceReconnect()
  }, [forceReconnect])

  return (
    <MantineProvider defaultColorScheme="dark">
      <Notifications />
      <Container size="xl" py="md">
        <Stack gap="lg">
          <Stack gap="xs">
            <Group justify="space-between" align="center">
              <Title order={1}>Word of Wisdom - Blockchain Visualizer</Title>
              <Group gap="md">
                {isRecovering && (
                  <Badge color="yellow" size="lg" variant="dot">
                    Recovering...
                  </Badge>
                )}
                <ConnectionStatus 
                  readyState={readyState}
                  connectionState={connectionState}
                  onReconnect={handleEmergencyReset}
                />
              </Group>
            </Group>
            <Text size="sm" c="dimmed">
              Real-time visualization of a proof-of-work protected TCP server with adaptive DDoS protection
            </Text>
          </Stack>

          <Grid>
            <Grid.Col span={12}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Smart Mining Control</Title>
                <MiningConfigPanel
                  onStartMining={handleStartMining}
                  onStopMining={handleStopMining}
                  miningActive={miningActive}
                  connectionState={connectionState}
                  isRecovering={isRecovering}
                />
              </Paper>
            </Grid.Col>

            <Grid.Col span={12}>
              <MetricsDashboard metrics={metrics} />
            </Grid.Col>

            <Grid.Col span={12}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Blockchain</Title>
                <BlockchainVisualizer blocks={blocks} />
              </Paper>
            </Grid.Col>

            <Grid.Col span={8}>
              <Stack gap="md">
                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Active Mining Operations</Title>
                  <MiningVisualizer challenges={currentChallenges} />
                </Paper>

                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Network Activity Logs</Title>
                  <LogsPanel logs={logs} />
                </Paper>
              </Stack>
            </Grid.Col>

            <Grid.Col span={4}>
              <Stack gap="md">
                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Network Stats</Title>
                  <StatsPanel stats={stats} metrics={metrics} />
                </Paper>

                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Connected Clients</Title>
                  <ConnectionsPanel connections={connections} />
                </Paper>
              </Stack>
            </Grid.Col>
          </Grid>
        </Stack>
      </Container>
    </MantineProvider>
  )
}

export default App