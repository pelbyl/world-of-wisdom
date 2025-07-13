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
import { loadState, startAutoSave, setupBeforeUnloadSave } from './utils/persistence'
import { Block, Challenge, ClientConnection, MiningStats, MetricsData, LogMessage } from './types'

function App() {
  // Initialize state with persisted data if available
  const initializeState = () => {
    const persistedState = loadState()
    return {
      blocks: persistedState?.blocks || [],
      connections: persistedState?.connections || [],
      stats: {
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
        ...persistedState?.stats
      },
      logs: (persistedState?.logs || []).map(log => ({
        ...log,
        timestamp: typeof log.timestamp === 'string' ? Date.parse(log.timestamp) : log.timestamp
      })),
      metrics: persistedState?.metrics || null
    }
  }

  const initialState = initializeState()
  const [blocks, setBlocks] = useState<Block[]>(initialState.blocks)
  const [currentChallenges, setCurrentChallenges] = useState<Challenge[]>([])
  const [connections, setConnections] = useState<ClientConnection[]>(initialState.connections)
  const [stats, setStats] = useState<MiningStats>(initialState.stats)
  const [metrics, setMetrics] = useState<MetricsData | null>(initialState.metrics)
  const [miningActive, setMiningActive] = useState(false)
  const [logs, setLogs] = useState<LogMessage[]>(initialState.logs)
  const [isRecovering, setIsRecovering] = useState(false)

  const { sendMessage, lastMessage, readyState, connectionState, forceReconnect } = useWebSocket('ws://localhost:8081/ws')

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
              setLogs(prev => [...prev.slice(-99), data.log]) // Keep last 100 logs
            }
            break
          case 'init':
            // Merge server data with persisted data intelligently
            if (data.blocks && data.blocks.length > blocks.length) {
              setBlocks(data.blocks)
            }
            if (data.connections) {
              setConnections(data.connections)
            }
            if (data.stats) {
              setStats(prev => ({ ...prev, ...data.stats }))
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

  // Get current state for persistence
  const getCurrentState = useCallback(() => ({
    stats,
    blocks,
    connections,
    logs,
    metrics
  }), [stats, blocks, connections, logs, metrics])

  // Set up auto-save and before-unload save
  useEffect(() => {
    const stopAutoSave = startAutoSave(getCurrentState)
    const stopBeforeUnloadSave = setupBeforeUnloadSave(getCurrentState)
    
    return () => {
      stopAutoSave()
      stopBeforeUnloadSave()
    }
  }, [getCurrentState])

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