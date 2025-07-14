import { Stack, Text, Group, RingProgress, ThemeIcon, Paper } from '@mantine/core'
import { IconActivity, IconClock, IconTrendingUp } from '@tabler/icons-react'
import { MiningStats, MetricsData } from '../types'

interface Props {
  stats: MiningStats
  metrics?: MetricsData | null
}

export function StatsPanel({ stats, metrics }: Props) {
  const successRate = stats.totalChallenges > 0
    ? (stats.completedChallenges / stats.totalChallenges) * 100
    : 0

  // Use live difficulty from metrics if available, otherwise fall back to stats
  const currentDifficulty = metrics?.currentDifficulty || stats.currentDifficulty

  // Format average solve time: show in seconds if > 1000ms, otherwise milliseconds
  const formatSolveTime = (timeMs: number) => {
    if (timeMs > 1000) {
      return `${(timeMs / 1000).toFixed(2)}s`
    }
    return `${timeMs.toFixed(1)}ms`
  }

  return (
    <Stack gap="md">
      <Text size="xs" c="dimmed">
        Live system statistics from the API server
      </Text>
      
      <Group justify="space-between">
        <div>
          <Text size="xs" c="dimmed">Success Rate</Text>
          <Text size="xl" fw={700}>{successRate.toFixed(1)}%</Text>
        </div>
        <RingProgress
          size={60}
          thickness={4}
          sections={[{ value: successRate, color: 'green' }]}
        />
      </Group>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="blue">
            <IconTrendingUp size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Current Difficulty</Text>
            <Text fw={500}>{currentDifficulty}</Text>
          </div>
        </Group>
      </Paper>


      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="orange">
            <IconClock size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Avg Solve Time</Text>
            <Text fw={500}>{formatSolveTime(stats.averageSolveTime)}</Text>
          </div>
        </Group>
      </Paper>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="purple">
            <IconActivity size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Total Challenges</Text>
            <Text fw={500}>{stats.totalChallenges}</Text>
          </div>
        </Group>
      </Paper>
    </Stack>
  )
}