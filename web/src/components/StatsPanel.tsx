import { Stack, Text, Group, RingProgress, ThemeIcon, Paper } from '@mantine/core'
import { IconActivity, IconClock, IconHash, IconTrendingUp } from '@tabler/icons-react'
import { MiningStats } from '../types'

interface Props {
  stats: MiningStats
}

export function StatsPanel({ stats }: Props) {
  const successRate = stats.totalChallenges > 0 
    ? (stats.completedChallenges / stats.totalChallenges) * 100 
    : 0

  return (
    <Stack gap="md">
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
            <Text fw={500}>{stats.currentDifficulty}</Text>
          </div>
        </Group>
      </Paper>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="green">
            <IconHash size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Hash Rate</Text>
            <Text fw={500}>{(stats.hashRate / 1000).toFixed(2)} KH/s</Text>
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
            <Text fw={500}>{(stats.averageSolveTime / 1000).toFixed(2)}s</Text>
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