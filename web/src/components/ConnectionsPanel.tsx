import { Stack, Text, Group, Badge, Indicator, ScrollArea } from '@mantine/core'
import { ClientConnection } from '../types/api'
import { useConnections } from '../hooks/useV1API'

interface Props {}

export function ConnectionsPanel({}: Props) {
  const { data: connectionsResponse, loading, error } = useConnections()
  
  const connections = connectionsResponse?.status === 'success' && connectionsResponse.data ? connectionsResponse.data.connections : []
  
  if (loading) {
    return <Text c="dimmed" size="sm">Loading connections...</Text>
  }
  
  if (error) {
    return <Text c="red" size="sm">Error loading connections</Text>
  }
  
  if (connections.length === 0) {
    return <Text c="dimmed" size="sm">No active connections</Text>
  }

  return (
    <ScrollArea h={200}>
      <Stack gap="xs">
        {connections.map((connection: ClientConnection) => (
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
        
        {connections.length === 0 && (
          <Text size="xs" c="dimmed" ta="center">
            No connections yet
          </Text>
        )}
      </Stack>
    </ScrollArea>
  )
}