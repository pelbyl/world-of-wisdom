import { Stack, Text, Group, RingProgress, ThemeIcon, Paper, Alert, Badge } from '@mantine/core'
import { IconActivity, IconClock, IconHash, IconTrendingUp, IconUsers, IconUserPlus, IconShieldCheck, IconNetwork } from '@tabler/icons-react'
import { MiningStats } from '../types'

interface Props {
  stats: MiningStats
}

export function StatsPanel({ stats }: Props) {
  const successRate = stats.totalChallenges > 0 
    ? (stats.completedChallenges / stats.totalChallenges) * 100 
    : 0

  const getIntensityColor = (intensity: number) => {
    switch (intensity) {
      case 1: return 'green'
      case 2: return 'yellow' 
      case 3: return 'red'
      default: return 'gray'
    }
  }

  const getIntensityLabel = (intensity: number) => {
    switch (intensity) {
      case 1: return 'Low'
      case 2: return 'Medium'
      case 3: return 'High'
      default: return 'Unknown'
    }
  }

  return (
    <Stack gap="md">
      {stats.ddosProtectionActive && (
        <Alert color="red" icon={<IconShieldCheck size={16} />}>
          <Text size="sm" fw={500}>🔒 DDoS Protection Active</Text>
          <Text size="xs">High network load detected - adaptive security engaged</Text>
        </Alert>
      )}
      
      <Text size="xs" c="dimmed">
        Live simulation statistics from the WebSocket server
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
          <ThemeIcon size="sm" variant="light" color="cyan">
            <IconUsers size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Live Connections</Text>
            <Text fw={500}>{stats.liveConnections || 0}</Text>
          </div>
        </Group>
      </Paper>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="indigo">
            <IconUserPlus size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Total Connections</Text>
            <Text fw={500}>{stats.totalConnections || 0}</Text>
          </div>
        </Group>
      </Paper>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon 
            size="sm" 
            variant="light" 
            color={getIntensityColor(stats.networkIntensity || 1)}
          >
            <IconNetwork size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Network Intensity</Text>
            <Group gap="xs">
              <Text fw={500}>{getIntensityLabel(stats.networkIntensity || 1)}</Text>
              <Badge 
                size="xs" 
                color={getIntensityColor(stats.networkIntensity || 1)}
              >
                Level {stats.networkIntensity || 1}
              </Badge>
            </Group>
          </div>
        </Group>
      </Paper>

      <Paper withBorder p="xs">
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="blue">
            <IconUsers size={16} />
          </ThemeIcon>
          <div>
            <Text size="xs" c="dimmed">Active Miners</Text>
            <Text fw={500}>{stats.activeMinerCount || 0}</Text>
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