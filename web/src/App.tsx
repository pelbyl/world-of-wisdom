import { MantineProvider, Container, Title, Grid, Paper, Group, Badge, Text, Stack } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { useEffect, useState } from 'react'
import { BlockchainVisualizer } from './components/BlockchainVisualizer'
import { MiningVisualizer } from './components/MiningVisualizer'
import { ConnectionsPanel } from './components/ConnectionsPanel'
import { StatsPanel } from './components/StatsPanel'
import { ChallengePanel } from './components/ChallengePanel'
import { useWebSocket } from './hooks/useWebSocket'
import { Block, Challenge, ClientConnection, MiningStats } from './types'

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

  const { sendMessage, lastMessage, readyState } = useWebSocket('ws://localhost:8081/ws')

  useEffect(() => {
    if (lastMessage) {
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
        case 'init':
          setBlocks(data.blocks || [])
          setConnections(data.connections || [])
          setStats(data.stats || stats)
          break
      }
    }
  }, [lastMessage])

  return (
    <MantineProvider defaultColorScheme="dark">
      <Notifications />
      <Container size="xl" py="md">
        <Stack gap="lg">
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

          <Grid>
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
                <Title order={3} mb="md">Challenge Monitor</Title>
                <ChallengePanel 
                  challenges={currentChallenges} 
                  onSimulateClient={() => sendMessage(JSON.stringify({ type: 'simulate_client' }))}
                />
              </Paper>
            </Grid.Col>
          </Grid>
        </Stack>
      </Container>
    </MantineProvider>
  )
}