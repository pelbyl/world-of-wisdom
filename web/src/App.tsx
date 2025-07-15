import { MantineProvider, Container, Title, Grid, Paper, Stack, Text, Tabs } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { MetricsDashboard } from './components/MetricsDashboard'
import { CategorizedLogs } from './components/CategorizedLogs'
import { LiveLogs } from './components/LiveLogs'
import { ChallengePanel } from './components/ChallengePanel'
import { ConnectionsPanel } from './components/ConnectionsPanel'
import { ExperimentAnalytics } from './components/ExperimentAnalytics'
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

          <Tabs defaultValue="dashboard">
            <Tabs.List>
              <Tabs.Tab value="dashboard">📊 Dashboard</Tabs.Tab>
              <Tabs.Tab value="experiments">🧪 Experiments</Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="dashboard" pt="lg">
              <Grid>
                <Grid.Col span={12}>
                  <MetricsDashboard />
                </Grid.Col>

                <Grid.Col span={12}>
                  <Grid>
                    <Grid.Col span={4}>
                      <Paper shadow="xs" p="md" withBorder style={{ height: '800px', display: 'flex', flexDirection: 'column' }}>
                        <Title order={3} mb="md">Recent Challenges</Title>
                        <div style={{ flex: 1, overflow: 'hidden' }}>
                          <ChallengePanel />
                        </div>
                      </Paper>
                    </Grid.Col>

                    <Grid.Col span={4}>
                      <Paper shadow="xs" p="md" withBorder style={{ height: '800px' }}>
                        <Title order={3} mb="md">Active Connections</Title>
                        <ConnectionsPanel />
                      </Paper>
                    </Grid.Col>

                    <Grid.Col span={4}>
                      <div style={{ height: '800px' }}>
                        <LiveLogs />
                      </div>
                    </Grid.Col>
                  </Grid>
                </Grid.Col>

                <Grid.Col span={12}>
                  <Paper shadow="xs" p="md" withBorder>
                    <Title order={3} mb="md">System Logs</Title>
                    <CategorizedLogs logs={logs} />
                  </Paper>
                </Grid.Col>
              </Grid>
            </Tabs.Panel>

            <Tabs.Panel value="experiments" pt="lg">
              <ExperimentAnalytics />
            </Tabs.Panel>

          </Tabs>
        </Stack>
      </Container>
    </MantineProvider>
  )
}

export default App