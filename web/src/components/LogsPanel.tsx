import { Stack, Text, ScrollArea, Group, Badge, Card } from '@mantine/core'
import { LogMessage } from '../types'
import { useEffect, useRef } from 'react'

interface Props {
  logs: LogMessage[]
}

export function LogsPanel({ logs }: Props) {
  const scrollAreaRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollAreaRef.current) {
      scrollAreaRef.current.scrollTop = scrollAreaRef.current.scrollHeight
    }
  }, [logs])

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'success': return 'green'
      case 'warning': return 'yellow'
      case 'error': return 'red'
      case 'info': 
      default: return 'blue'
    }
  }

  const formatTime = (timestamp: number) => {
    return new Date(timestamp).toLocaleTimeString([], {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  if (logs.length === 0) {
    return (
      <Text c="dimmed" ta="center" py="xl">
        No simulation logs yet - start auto mining to see network activity!
      </Text>
    )
  }

  return (
    <ScrollArea h={300} ref={scrollAreaRef}>
      <Stack gap="xs">
        {logs.map((log, index) => (
          <Card key={index} withBorder p="xs" style={{ backgroundColor: 'var(--mantine-color-dark-8)' }}>
            <Group gap="xs" wrap="nowrap">
              <Badge size="xs" color={getLevelColor(log.level)} variant="light">
                {formatTime(log.timestamp)}
              </Badge>
              <Text 
                size="sm" 
                style={{ 
                  fontFamily: 'monospace',
                  flex: 1,
                  wordBreak: 'break-word'
                }}
                c={log.level === 'error' ? 'red' : 
                   log.level === 'warning' ? 'yellow' :
                   log.level === 'success' ? 'green' : 'white'}
              >
                {log.message}
              </Text>
            </Group>
          </Card>
        ))}
      </Stack>
    </ScrollArea>
  )
}