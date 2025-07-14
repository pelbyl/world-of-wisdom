import { Paper, Title, Grid, Group, Badge, Text, Stack, RingProgress, ThemeIcon, Loader, Alert, Progress } from '@mantine/core'
import { LineChart, AreaChart } from '@mantine/charts'
import { IconShield, IconClock, IconNetwork, IconInfoCircle, IconRefresh } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
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

export function MetricsDashboard() {
  const { data: statsResponse, loading, error } = useStats({ interval: 1000 }) // 1s polling for stats
  const [metricsHistory, setMetricsHistory] = useState<MetricsData[]>([])

  const stats: StatsData | null = statsResponse?.status === 'success' ? statsResponse.data ?? null : null;

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
        // Keep last 50 data points
        return newHistory.slice(-50);
      });
    }
  }, [stats]);

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

  const lineChartData = metricsHistory.map((m, index) => ({
    time: index,
    difficulty: m.currentDifficulty,
    solveTime: m.averageSolveTime,
    connectionRate: m.connectionRate,
    connections: m.connectionsTotal
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
                    Total Connections
                  </Text>
                  <Text fw={700} size="xl">
                    {currentMetrics.connectionsTotal}
                  </Text>
                </div>
                <ThemeIcon color="blue" variant="light" size={38} radius="md">
                  <IconNetwork size={20} />
                </ThemeIcon>
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                Active: {currentMetrics.activeConnections}
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
            {metricsHistory.length > 1 ? (
              <LineChart
                h={200}
                data={lineChartData}
                dataKey="time"
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
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: 4 }}>
          <Paper p="md" withBorder>
            <Title order={4} mb="md">Solve Time Trend</Title>
            {metricsHistory.length > 1 ? (
              <LineChart
                h={200}
                data={lineChartData}
                dataKey="time"
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
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: 4 }}>
          <Paper p="md" withBorder>
            <Title order={4} mb="md">Connection Activity</Title>
            {metricsHistory.length > 1 ? (
              <AreaChart
                h={200}
                data={lineChartData}
                dataKey="time"
                series={[
                  { name: 'connectionRate', color: 'green.6', label: 'Rate/min' },
                  { name: 'connections', color: 'blue.6', label: 'Total' }
                ]}
                withLegend
                withDots={false}
              />
            ) : (
              <Text c="dimmed" ta="center" py="xl">Collecting data...</Text>
            )}
          </Paper>
        </Grid.Col>
      </Grid>

      {/* DDoS Protection Status */}
      <Paper p="md" withBorder>
        <Group justify="space-between">
          <div>
            <Title order={4}>DDoS Protection Status</Title>
            <Text size="sm" c="dimmed">
              Adaptive difficulty adjustments: {currentMetrics.difficultyAdjustments}
            </Text>
          </div>
          <Badge
            color={currentMetrics.connectionRate > 20 ? 'red' : currentMetrics.connectionRate > 10 ? 'orange' : 'green'}
            variant="light"
            size="lg"
          >
            {currentMetrics.connectionRate > 20 ? 'High Load Detected' :
              currentMetrics.connectionRate > 10 ? 'Moderate Load' : 'Normal Operation'}
          </Badge>
        </Group>

        {currentMetrics.connectionRate > 20 && (
          <Text size="sm" c="red" mt="sm">
            ⚠️ High connection rate detected - difficulty automatically increased to {currentMetrics.currentDifficulty}
          </Text>
        )}
      </Paper>
    </Stack>
  )
}