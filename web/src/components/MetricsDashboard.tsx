import { Paper, Title, Grid, Group, Badge, Text, Stack, RingProgress, ThemeIcon, Loader, Alert, Progress, ScrollArea, Table } from '@mantine/core'
import { AreaChart, LineChart } from '@mantine/charts'
import { IconShield, IconClock, IconNetwork, IconInfoCircle, IconRefresh, IconUser, IconAlertTriangle } from '@tabler/icons-react'
import { useEffect, useState, useRef } from 'react'
import { useStats } from '../hooks/useAPI'
import { StatsData } from '../types/api'

interface MetricsData {
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

interface ClientBehavior {
  ip: string
  difficulty: number
  connectionCount: number
  failureRate: number
  avgSolveTime: number
  reconnectRate: number
  reputation: number
  suspicious: number
  lastConnection: string
  isAggressive: boolean
  successfulChallenges: number
  failedChallenges: number
  totalChallenges: number
}

export function MetricsDashboard() {
  const { data: statsResponse, loading, error } = useStats({ interval: 1000 }) // 1s polling for stats
  const [clientsResponse, setClientsResponse] = useState<{ data: { clients: ClientBehavior[] } } | null>(null)
  const [metricsHistory, setMetricsHistory] = useState<MetricsData[]>([])
  const difficultyChartRef = useRef<HTMLDivElement>(null)
  const solveTimeChartRef = useRef<HTMLDivElement>(null)
  const connectionChartRef = useRef<HTMLDivElement>(null)

  const stats: StatsData | null = statsResponse?.status === 'success' ? statsResponse.data ?? null : null;

  // Fetch client behaviors
  useEffect(() => {
    const fetchClients = async () => {
      try {
        const response = await fetch('http://localhost:8081/api/v1/client-behaviors');
        if (response.ok) {
          const data = await response.json();
          setClientsResponse(data);
        }
      } catch (err) {
        console.error('Failed to fetch client behaviors:', err);
      }
    };

    fetchClients();
    const interval = setInterval(fetchClients, 5000); // Poll every 5 seconds
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (stats) {
      // Convert stats to metrics format for charts
      const metrics: MetricsData = {
        timestamp: Date.now(),
        connectionsTotal: stats.connections?.total ?? 0,
        currentDifficulty: stats.stats?.currentDifficulty ?? 0,
        puzzlesSolvedTotal: stats.stats?.completedChallenges ?? 0,
        puzzlesFailedTotal: (stats.stats?.totalChallenges ?? 0) - (stats.stats?.completedChallenges ?? 0),
        averageSolveTime: stats.stats?.averageSolveTime ?? 0,
        connectionRate: 0, // This would need to be calculated from connection history
        difficultyAdjustments: 0, // This would need to be tracked separately
        activeConnections: stats.connections?.active ?? 0
      };
      setMetricsHistory(prev => {
        const newHistory = [...prev, metrics];
        // Keep all historical data
        return newHistory;
      });
    }
  }, [stats]);

  // Remove auto-scroll - charts will fit in container

  if (loading || !stats) {
    return (
      <Paper p="lg" withBorder>
        <Stack gap="md">
          <Group justify="space-between">
            <Title order={3}>Live Metrics Dashboard</Title>
            <Badge color="orange" variant="light" leftSection={<Loader size={12} />}>
              {loading ? 'Loading' : 'Connecting'}
            </Badge>
          </Group>
          {error ? (
            <Alert icon={<IconInfoCircle size={16} />} color="red">
              <Text size="sm">
                <strong>Connection Error:</strong> {error}
                <br />
                Retrying automatically...
              </Text>
            </Alert>
          ) : (
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              <Text size="sm">
                <strong>Loading dashboard metrics...</strong>
                <br />
                • Connecting to API server
                <br />
                • Fetching real-time statistics
                <br />
                • Loading current system status
              </Text>
            </Alert>
          )}
          <Progress size="sm" animated color={error ? "red" : "blue"} value={100} />
          <Text size="sm" c="dimmed" ta="center">
            {error ? 'Connection failed - retrying...' : 'This may take a few seconds on first load'}
          </Text>
        </Stack>
      </Paper>
    );
  }

  const currentMetrics: MetricsData = {
    timestamp: Date.now(),
    connectionsTotal: stats.connections?.total ?? 0,
    currentDifficulty: stats.stats?.currentDifficulty ?? 0,
    puzzlesSolvedTotal: stats.stats?.completedChallenges ?? 0,
    puzzlesFailedTotal: (stats.stats?.totalChallenges ?? 0) - (stats.stats?.completedChallenges ?? 0),
    averageSolveTime: stats.stats?.averageSolveTime ?? 0,
    connectionRate: 0,
    difficultyAdjustments: 0,
    activeConnections: stats.connections?.active ?? 0
  };

  const successRate = currentMetrics.puzzlesSolvedTotal + currentMetrics.puzzlesFailedTotal > 0
    ? (currentMetrics.puzzlesSolvedTotal / (currentMetrics.puzzlesSolvedTotal + currentMetrics.puzzlesFailedTotal)) * 100
    : 0

  const lineChartData = metricsHistory.map((m) => ({
    time: new Date(m.timestamp).toLocaleTimeString(),
    timestamp: m.timestamp,
    difficulty: m.currentDifficulty,
    solveTime: m.averageSolveTime,
    connectionRate: m.connectionRate,
    connections: m.connectionsTotal,
    activeConnections: m.activeConnections
  }))

  // Format average solve time: show in seconds if > 1000ms, otherwise milliseconds
  const formatSolveTime = (timeMs: number) => {
    if (timeMs > 1000) {
      return `${(timeMs / 1000).toFixed(2)}s`
    }
    return `${timeMs.toFixed(1)}ms`
  }

  const difficultyColor = currentMetrics.currentDifficulty >= 4 ? 'red' : currentMetrics.currentDifficulty >= 3 ? 'orange' : 'green'

  return (
    <Stack gap="lg">
      <Paper p="lg" withBorder>
        <Group justify="space-between" mb="lg">
          <Title order={3}>Live Metrics Dashboard</Title>
          <Group gap="xs">
            <Badge
              color="green"
              variant="light"
              leftSection={<IconRefresh size={12} />}
            >
              HTTP Polling Active
            </Badge>
            <Badge
              color="blue"
              variant="light"
              size="sm"
            >
              {stats.miningActive ? 'Mining Active' : 'Mining Stopped'}
            </Badge>
          </Group>
        </Group>

        {/* Key Metrics Cards */}
        <Grid>
          <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
            <Paper p="md" withBorder>
              <Group justify="apart">
                <div>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={700}>
                    Current Difficulty
                  </Text>
                  <Text fw={700} size="xl">
                    {currentMetrics.currentDifficulty}
                  </Text>
                </div>
                <ThemeIcon color={difficultyColor} variant="light" size={38} radius="md">
                  <IconShield size={20} />
                </ThemeIcon>
              </Group>
              <Badge color={difficultyColor} variant="light" size="sm" mt="sm">
                {currentMetrics.currentDifficulty >= 4 ? 'High Security' :
                  currentMetrics.currentDifficulty >= 3 ? 'Medium Security' : 'Normal'}
              </Badge>
            </Paper>
          </Grid.Col>

          <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
            <Paper p="md" withBorder>
              <Group justify="apart">
                <div>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={700}>
                    Active Connections
                  </Text>
                  <Text fw={700} size="xl">
                    {currentMetrics.activeConnections}
                  </Text>
                </div>
                <ThemeIcon color="blue" variant="light" size={38} radius="md">
                  <IconNetwork size={20} />
                </ThemeIcon>
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                Total: {currentMetrics.connectionsTotal}
              </Text>
            </Paper>
          </Grid.Col>

          <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
            <Paper p="md" withBorder>
              <Group justify="apart">
                <div>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={700}>
                    Success Rate
                  </Text>
                  <Text fw={700} size="xl">
                    {successRate.toFixed(1)}%
                  </Text>
                </div>
                <RingProgress
                  size={38}
                  thickness={4}
                  sections={[{ value: successRate, color: successRate > 90 ? 'green' : 'orange' }]}
                />
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                {currentMetrics.puzzlesSolvedTotal} solved / {currentMetrics.puzzlesFailedTotal} failed
              </Text>
            </Paper>
          </Grid.Col>

          <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
            <Paper p="md" withBorder>
              <Group justify="apart">
                <div>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={700}>
                    Avg Solve Time
                  </Text>
                  <Text fw={700} size="xl">
                    {formatSolveTime(currentMetrics.averageSolveTime)}
                  </Text>
                </div>
                <ThemeIcon color="teal" variant="light" size={38} radius="md">
                  <IconClock size={20} />
                </ThemeIcon>
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                Connection rate: {currentMetrics.connectionRate.toFixed(1)}/min
              </Text>
            </Paper>
          </Grid.Col>
        </Grid>
      </Paper>

      {/* Charts */}
      <Grid>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Paper p="md" withBorder>
            <Title order={4} mb="md">Difficulty Trend</Title>
            <div ref={difficultyChartRef} style={{ maxWidth: '100%' }}>
              {metricsHistory.length > 1 ? (
                <LineChart
                  h={200}
                  w="100%"
                  data={lineChartData}
                  dataKey="time"
                  xAxisLabel=""
                  series={[
                    { name: 'difficulty', color: 'blue.6', label: 'Difficulty Level' }
                  ]}
                  curveType="linear"
                  withLegend
                  withDots={false}
                  yAxisProps={{ domain: [1, 8] }}
                />
              ) : (
                <Text c="dimmed" ta="center" py="xl">Collecting data...</Text>
              )}
            </div>
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: 4 }}>
          <Paper p="md" withBorder>
            <Title order={4} mb="md">Solve Time Trend</Title>
            <div ref={solveTimeChartRef} style={{ maxWidth: '100%' }}>
              {metricsHistory.length > 1 ? (
                <LineChart
                  h={200}
                  w="100%"
                  data={lineChartData}
                  dataKey="time"
                  xAxisLabel=""
                  series={[
                    {
                      name: 'solveTime',
                      color: 'red.6',
                      label: 'Solve Time (ms)'
                    }
                  ]}
                  curveType="linear"
                  withLegend
                  withDots={false}
                  yAxisProps={{
                    domain: ['dataMin', 'dataMax'],
                    tickFormatter: (value: number) => `${value.toFixed(1)} ms`,
                  }}
                />
              ) : (
                <Text c="dimmed" ta="center" py="xl">Collecting data...</Text>
              )}
            </div>
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: 4 }}>
          <Paper p="md" withBorder>
            <Title order={4} mb="md">Connection Activity</Title>
            <div ref={connectionChartRef} style={{ maxWidth: '100%' }}>
              {metricsHistory.length > 1 ? (
                <AreaChart
                  h={200}
                  w="100%"
                  data={lineChartData}
                  dataKey="time"
                  xAxisLabel=""
                  series={[
                    { name: 'activeConnections', color: 'green.6', label: 'Active Connections' }
                  ]}
                  withLegend
                  withDots={false}
                />
              ) : (
                <Text c="dimmed" ta="center" py="xl">Collecting data...</Text>
              )}
            </div>
          </Paper>
        </Grid.Col>
      </Grid>

      {/* Client Behaviors */}
      <Paper p="md" withBorder>
        <Group justify="space-between" mb="md">
          <Title order={4}>Active Clients - Per-IP Difficulty</Title>
          <Group gap="xs">
            <Badge color="green" variant="light" size="sm">
              <IconUser size={12} style={{ marginRight: 4 }} />
              Normal Clients
            </Badge>
            <Badge color="red" variant="light" size="sm">
              <IconAlertTriangle size={12} style={{ marginRight: 4 }} />
              Aggressive Clients
            </Badge>
          </Group>
        </Group>
        
        {clientsResponse?.data?.clients && clientsResponse.data.clients.length > 0 ? (
          <ScrollArea h={400}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>IP Address</Table.Th>
                  <Table.Th>Difficulty</Table.Th>
                  <Table.Th>Connections</Table.Th>
                  <Table.Th>Success / Failure</Table.Th>
                  <Table.Th>Avg Solve Time</Table.Th>
                  <Table.Th>Reputation</Table.Th>
                  <Table.Th>Status</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {clientsResponse.data.clients.map((client: ClientBehavior) => {
                  const timeSinceConnection = new Date(client.lastConnection);
                  const minutesAgo = Math.floor((Date.now() - timeSinceConnection.getTime()) / 60000);
                  
                  return (
                    <Table.Tr key={client.ip}>
                      <Table.Td>
                        <Group gap="xs">
                          {client.isAggressive && <Text fw={700} c="red">!</Text>}
                          <Text fw={500}>{client.ip}</Text>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Badge 
                          color={client.difficulty >= 5 ? 'red' : client.difficulty >= 3 ? 'orange' : 'green'}
                          variant="filled"
                          size="sm"
                        >
                          {client.difficulty}
                        </Badge>
                      </Table.Td>
                      <Table.Td>{client.connectionCount}</Table.Td>
                      <Table.Td>
                        <Group gap="xs">
                          <Badge color="green" variant="light" size="xs">
                            {client.successfulChallenges}
                          </Badge>
                          <Text size="xs" c="dimmed">/</Text>
                          <Badge color="red" variant="light" size="xs">
                            {client.failedChallenges}
                          </Badge>
                          {client.totalChallenges > 0 && (
                            <Text size="xs" c={client.failureRate > 0.5 ? 'red' : 'green'}>
                              ({((client.successfulChallenges / client.totalChallenges) * 100).toFixed(1)}% success)
                            </Text>
                          )}
                          {client.totalChallenges === 0 && (
                            <Text size="xs" c="dimmed">
                              (no attempts)
                            </Text>
                          )}
                        </Group>
                      </Table.Td>
                      <Table.Td>{formatSolveTime(client.avgSolveTime)}</Table.Td>
                      <Table.Td>
                        <Progress 
                          value={client.reputation} 
                          color={client.reputation < 30 ? 'red' : client.reputation < 60 ? 'orange' : 'green'}
                          size="sm"
                          w={60}
                        />
                      </Table.Td>
                      <Table.Td>
                        <Text size="xs" c="dimmed">
                          {minutesAgo === 0 ? 'Just now' : `${minutesAgo}m ago`}
                        </Text>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        ) : (
          <Text c="dimmed" ta="center" py="xl">
            No active clients detected
          </Text>
        )}
        
        {clientsResponse?.data?.clients && (
          <Group justify="space-between" mt="md">
            <Text size="sm" c="dimmed">
              Total clients: {clientsResponse.data.clients.length}
            </Text>
            <Group gap="xs">
              <Text size="sm" c="dimmed">
                High difficulty: {clientsResponse.data.clients.filter((c: ClientBehavior) => c.difficulty >= 5).length}
              </Text>
              <Text size="sm" c="dimmed">•</Text>
              <Text size="sm" c="dimmed">
                Aggressive: {clientsResponse.data.clients.filter((c: ClientBehavior) => c.isAggressive).length}
              </Text>
            </Group>
          </Group>
        )}
      </Paper>
    </Stack>
  )
}