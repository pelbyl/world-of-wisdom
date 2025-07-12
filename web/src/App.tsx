import { MantineProvider, Container, Title, Grid, Paper, Group, Badge, Stack, Text } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { useEffect, useState } from 'react'
import { BlockchainVisualizer } from './components/BlockchainVisualizer'
import { MiningVisualizer } from './components/MiningVisualizer'
import { ConnectionsPanel } from './components/ConnectionsPanel'
import { StatsPanel } from './components/StatsPanel'
import { ChallengePanel } from './components/ChallengePanel'
import { MetricsDashboard } from './components/MetricsDashboard'
import { useWebSocket } from './hooks/useWebSocket'
import { Block, Challenge, ClientConnection, MiningStats, MetricsData } from './types'

function App() {
  const [blocks, setBlocks] = useState<Block[]>([])
  const [currentChallenges, setCurrentChallenges] = useState<Challenge[]>([])
  const [connections, setConnections] = useState<ClientConnection[]>([])
  const [stats, setStats] = useState<MiningStats>({
    totalChallenges: 0,
    completedChallenges: 0,
    averageSolveTime: 0,
    currentDifficulty: 2,
    hashRate: 0,
  })
  const [metrics, setMetrics] = useState<MetricsData | null>(null)
  const [miningActive, setMiningActive] = useState(false)

  const { sendMessage, lastMessage, readyState } = useWebSocket('ws://localhost:8081/ws')

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
            break
          case 'init':
            setBlocks(data.blocks || [])
            setConnections(data.connections || [])
            setStats(data.stats || stats)
            break
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error)
      }
    }
  }, [lastMessage])

  const handleStartMining = () => {
    sendMessage(JSON.stringify({ type: 'start_mining' }))
  }

  const handleStopMining = () => {
    sendMessage(JSON.stringify({ type: 'stop_mining' }))
  }

  return (
    <MantineProvider defaultColorScheme="dark">
      <Notifications />
      <Container size="xl" py="md">
        <Stack gap="lg">
          <Stack gap="xs">
            <Group justify="space-between" align="center">
              <Title order={1}>Word of Wisdom - Blockchain Visualizer</Title>
              <Badge 
                color={readyState === 1 ? 'green' : 'red'} 
                size="lg"
                variant="dot"
              >
                {readyState === 1 ? 'Connected' : 'Disconnected'}
              </Badge>
            </Group>
            <Text size="sm" c="dimmed">
              Real-time visualization of a proof-of-work protected TCP server with adaptive DDoS protection
            </Text>
          </Stack>

          <Grid>
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
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Active Mining Operations</Title>
                <MiningVisualizer challenges={currentChallenges} />
              </Paper>
            </Grid.Col>

            <Grid.Col span={4}>
              <Stack gap="md">
                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Network Stats</Title>
                  <StatsPanel stats={stats} />
                </Paper>

                <Paper shadow="xs" p="md" withBorder>
                  <Title order={3} mb="md">Connected Clients</Title>
                  <ConnectionsPanel connections={connections} />
                </Paper>
              </Stack>
            </Grid.Col>

            <Grid.Col span={12}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Mining Simulation</Title>
                <ChallengePanel 
                  challenges={currentChallenges} 
                  onSimulateClient={() => sendMessage(JSON.stringify({ type: 'simulate_client' }))}
                  onStartMining={handleStartMining}
                  onStopMining={handleStopMining}
                  miningActive={miningActive}
                />
              </Paper>
            </Grid.Col>
          </Grid>
        </Stack>
      </Container>
    </MantineProvider>
  )
}

export default App