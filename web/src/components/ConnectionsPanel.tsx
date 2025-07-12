import { Stack, Text, Group, Badge, Indicator } from '@mantine/core'
import { ClientConnection } from '../types'

interface Props {
  connections: ClientConnection[]
}

export function ConnectionsPanel({ connections }: Props) {
  if (connections.length === 0) {
    return <Text c="dimmed" size="sm">No active connections</Text>
  }

  return (
    <Stack gap="xs">
      {connections.slice(-8).map(connection => (
        <Group key={connection.id} justify="space-between" wrap="nowrap">
          <Group gap="xs" style={{ minWidth: 0 }}>
            <Indicator 
              size={8} 
              color={connection.status === 'connected' ? 'green' : 
                     connection.status === 'solving' ? 'blue' : 'gray'}
            >
              <div />
            </Indicator>
            <Text size="xs" style={{ fontFamily: 'monospace' }} truncate>
              {connection.id.substring(0, 8)}
            </Text>
          </Group>
          
          <Group gap="xs">
            <Badge 
              size="xs" 
              color={connection.status === 'connected' ? 'green' : 
                     connection.status === 'solving' ? 'blue' : 'gray'}
            >
              {connection.status}
            </Badge>
            <Text size="xs" c="dimmed">
              {connection.challengesCompleted}
            </Text>
          </Group>
        </Group>
      ))}
      
      {connections.length > 8 && (
        <Text size="xs" c="dimmed" ta="center">
          +{connections.length - 8} more connections
        </Text>
      )}
    </Stack>
  )
}