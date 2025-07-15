import { Stack, Text, ScrollArea, Group, Badge, Card, Paper, Title, Indicator, Box } from '@mantine/core'
import { useEffect, useRef, useState } from 'react'
import { useLogs } from '../hooks/useAPI'

export function LiveLogs() {
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const [isLive, setIsLive] = useState(true)
  
  // Fetch logs with higher frequency for live view
  const { data: logsResponse } = useLogs(50, { interval: 1000 })
  const logs = logsResponse?.status === 'success' && logsResponse.data ? logsResponse.data : []
  
  // Sort logs by timestamp (newest first)
  const sortedLogs = [...logs].sort((a, b) => b.timestamp - a.timestamp)
  
  // Auto-scroll to top when new logs arrive
  useEffect(() => {
    if (isLive && scrollAreaRef.current) {
      scrollAreaRef.current.scrollTop = 0
    }
  }, [logs.length, isLive])
  
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
    const date = new Date(timestamp)
    const time = date.toLocaleTimeString([], {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${time}.${ms}`
  }
  
  const getLogIcon = (message: string): string => {
    // Enhanced icon detection based on message content
    if (message.includes('New connection')) return '🔌'
    if (message.includes('Sending') && message.includes('challenge')) return '📤'
    if (message.includes('Received') && message.includes('solution')) return '📥'
    if (message.includes('Solution verified')) return '✅'
    if (message.includes('Invalid solution')) return '❌'
    if (message.includes('Client disconnected')) return '🔌'
    if (message.includes('Quote served')) return '💬'
    if (message.includes('Failed')) return '⚠️'
    if (message.includes('Error')) return '🚨'
    if (message.includes('binary')) return '📦'
    if (message.includes('json')) return '📄'
    return '📝'
  }
  
  return (
    <Paper shadow="xs" p="md" withBorder h="100%">
      <Stack gap="md" h="100%">
        <Group justify="space-between" align="center">
          <Group gap="xs">
            <Title order={3}>Live Activity Stream</Title>
            <Indicator processing disabled={!isLive} color="green">
              <Badge 
                size="sm" 
                color={isLive ? "green" : "gray"}
                variant="filled"
                style={{ cursor: 'pointer' }}
                onClick={() => setIsLive(!isLive)}
              >
                {isLive ? "🔴 LIVE" : "⏸️ PAUSED"}
              </Badge>
            </Indicator>
          </Group>
          <Text size="xs" c="dimmed">
            Last {sortedLogs.length} events
          </Text>
        </Group>
        
        <Box style={{ flex: 1, minHeight: 0 }}>
          <ScrollArea h="100%" ref={scrollAreaRef}>
            <Stack gap="xs">
              {sortedLogs.length === 0 ? (
                <Text c="dimmed" ta="center" py="xl">
                  Waiting for activity...
                </Text>
              ) : (
                sortedLogs.map((log, index) => {
                  const uniqueKey = `${log.timestamp}-${index}`
                  const icon = log.icon || getLogIcon(log.message)
                  
                  return (
                    <Card 
                      key={uniqueKey} 
                      p="xs" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-dark-8)',
                        borderLeft: `3px solid var(--mantine-color-${getLevelColor(log.level)}-6)`,
                        animation: index === 0 ? 'slideIn 0.3s ease-out' : undefined
                      }}
                    >
                      <Group gap="xs" wrap="nowrap">
                        <Text size="xl">{icon}</Text>
                        <Stack gap={0} style={{ flex: 1 }}>
                          <Group gap="xs">
                            <Badge size="xs" color={getLevelColor(log.level)} variant="light">
                              {log.level.toUpperCase()}
                            </Badge>
                            <Text size="xs" c="dimmed">
                              {formatTime(log.timestamp)}
                            </Text>
                          </Group>
                          <Text 
                            size="sm" 
                            style={{ 
                              fontFamily: 'monospace',
                              wordBreak: 'break-word'
                            }}
                            c={log.level === 'error' ? 'red.4' : 
                               log.level === 'warning' ? 'yellow.4' :
                               log.level === 'success' ? 'green.4' : 'gray.3'}
                          >
                            {log.message}
                          </Text>
                        </Stack>
                      </Group>
                    </Card>
                  )
                })
              )}
            </Stack>
            
            <style>{`
              @keyframes slideIn {
                from {
                  opacity: 0;
                  transform: translateY(-10px);
                }
                to {
                  opacity: 1;
                  transform: translateY(0);
                }
              }
            `}</style>
          </ScrollArea>
        </Box>
      </Stack>
    </Paper>
  )
}