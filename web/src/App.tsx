import { MantineProvider, Container, Title, Grid, Paper, Stack, Text } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { MetricsDashboard } from './components/MetricsDashboard'
import { LogsPanel } from './components/LogsPanel'
import { ChallengePanel } from './components/ChallengePanel'
import { ConnectionsPanel } from './components/ConnectionsPanel'
import { useLogs } from './hooks/useAPI'

function App() {
  const { data: logsResponse } = useLogs(100, { interval: 2000 })
  const logs = logsResponse?.status === 'success' && logsResponse.data ? logsResponse.data : []

  return (
    <MantineProvider defaultColorScheme="dark">
      <Notifications />
      <Container size="xl" py="md">
        <Stack gap="lg">
          <Stack gap="xs">
            <Title order={1}>Word of Wisdom - System Monitor</Title>
            <Text size="sm" c="dimmed">
              Real-time monitoring of proof-of-work protected TCP server
            </Text>
          </Stack>

          <Grid>
            <Grid.Col span={12}>
              <MetricsDashboard />
            </Grid.Col>

            <Grid.Col span={6}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Recent Challenges</Title>
                <ChallengePanel />
              </Paper>
            </Grid.Col>

            <Grid.Col span={6}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">Active Connections</Title>
                <ConnectionsPanel />
              </Paper>
            </Grid.Col>

            <Grid.Col span={12}>
              <Paper shadow="xs" p="md" withBorder>
                <Title order={3} mb="md">System Logs</Title>
                <LogsPanel logs={logs} />
              </Paper>
            </Grid.Col>
          </Grid>
        </Stack>
      </Container>
    </MantineProvider>
  )
}

export default App