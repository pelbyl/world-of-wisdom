import { Paper, Title, Grid, Group, Badge, Text, Stack, RingProgress, ThemeIcon, Loader, Alert, Progress } from '@mantine/core'
import { LineChart, AreaChart } from '@mantine/charts'
import { IconShield, IconClock, IconNetwork, IconWifi, IconInfoCircle } from '@tabler/icons-react'
import { useEffect, useState } from 'react'

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

interface MetricsDashboardProps {
  metrics: MetricsData | null
}

export function MetricsDashboard({ metrics }: MetricsDashboardProps) {
  const [metricsHistory, setMetricsHistory] = useState<MetricsData[]>([])

  useEffect(() => {
    if (metrics) {
      setMetricsHistory(prev => {
        const newHistory = [...prev, metrics]
        // Keep last 50 data points
        return newHistory.slice(-50)
      })
    }
  }, [metrics])

  if (!metrics) {
    return (
      <Paper p="lg" withBorder>
        <Stack gap="md">
          <Group justify="space-between">
            <Title order={3}>Live Metrics Dashboard</Title>
            <Badge color="orange" variant="light" leftSection={<Loader size={12} />}>
              Connecting
            </Badge>
          </Group>

          <Alert icon={<IconInfoCircle size={16} />} color="blue">
            <Text size="sm">
              <strong>Setting up real-time metrics...</strong>
              <br />
              • Connecting to Prometheus server
              <br />
              • Establishing WebSocket connection
              <br />
              • Loading current system status
            </Text>
          </Alert>

          <Progress size="sm" animated color="blue" value={100} />

          <Text size="sm" c="dimmed" ta="center">
            This may take a few seconds on first load
          </Text>
        </Stack>
      </Paper>
    )
  }

  const successRate = metrics.puzzlesSolvedTotal + metrics.puzzlesFailedTotal > 0
    ? (metrics.puzzlesSolvedTotal / (metrics.puzzlesSolvedTotal + metrics.puzzlesFailedTotal)) * 100
    : 0

  const lineChartData = metricsHistory.map((m, index) => ({
    time: index,
    difficulty: m.currentDifficulty,
    solveTime: m.averageSolveTime,
    connectionRate: m.connectionRate,
    connections: m.connectionsTotal
  }))

  const difficultyColor = metrics.currentDifficulty >= 4 ? 'red' : metrics.currentDifficulty >= 3 ? 'orange' : 'green'

  return (
    <Stack gap="lg">
      <Paper p="lg" withBorder>
        <Group justify="space-between" mb="lg">
          <Title order={3}>Live Metrics Dashboard</Title>
          <Badge
            color="green"
            variant="light"
            leftSection={<IconWifi size={12} />}
          >
            Live Data Connected
          </Badge>
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
                    {metrics.currentDifficulty}
                  </Text>
                </div>
                <ThemeIcon color={difficultyColor} variant="light" size={38} radius="md">
                  <IconShield size={20} />
                </ThemeIcon>
              </Group>
              <Badge color={difficultyColor} variant="light" size="sm" mt="sm">
                {metrics.currentDifficulty >= 4 ? 'High Security' :
                  metrics.currentDifficulty >= 3 ? 'Medium Security' : 'Normal'}
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
                    {metrics.connectionsTotal}
                  </Text>
                </div>
                <ThemeIcon color="blue" variant="light" size={38} radius="md">
                  <IconNetwork size={20} />
                </ThemeIcon>
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                Active: {metrics.activeConnections}
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
                {metrics.puzzlesSolvedTotal} solved / {metrics.puzzlesFailedTotal} failed
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
                    {metrics.averageSolveTime.toFixed(1)}ms
                  </Text>
                </div>
                <ThemeIcon color="teal" variant="light" size={38} radius="md">
                  <IconClock size={20} />
                </ThemeIcon>
              </Group>
              <Text size="xs" c="dimmed" mt="sm">
                Connection rate: {metrics.connectionRate.toFixed(1)}/min
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
              Adaptive difficulty adjustments: {metrics.difficultyAdjustments}
            </Text>
          </div>
          <Badge
            color={metrics.connectionRate > 20 ? 'red' : metrics.connectionRate > 10 ? 'orange' : 'green'}
            variant="light"
            size="lg"
          >
            {metrics.connectionRate > 20 ? 'High Load Detected' :
              metrics.connectionRate > 10 ? 'Moderate Load' : 'Normal Operation'}
          </Badge>
        </Group>

        {metrics.connectionRate > 20 && (
          <Text size="sm" c="red" mt="sm">
            ⚠️ High connection rate detected - difficulty automatically increased to {metrics.currentDifficulty}
          </Text>
        )}
      </Paper>
    </Stack>
  )
}