import { Stack, Text, ScrollArea, Group, Badge, Card, Tabs, Indicator } from '@mantine/core'
import { LogMessage } from '../types'
import { useEffect, useRef, useState, useMemo } from 'react'

interface Props {
  logs: LogMessage[]
}

export function CategorizedLogs({ logs }: Props) {
  const [activeTab, setActiveTab] = useState<string>('all')
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollAreaRefs = {
    all: useRef<HTMLDivElement>(null),
    connections: useRef<HTMLDivElement>(null),
    warnings: useRef<HTMLDivElement>(null),
    successes: useRef<HTMLDivElement>(null),
    errors: useRef<HTMLDivElement>(null),
  }

  // Categorize logs and limit to 100 per category
  const categorizedLogs = useMemo(() => {
    const sorted = [...logs].sort((a, b) => b.timestamp - a.timestamp)
    
    return {
      all: sorted.slice(0, 100),
      connections: sorted.filter(log => 
        log.message.toLowerCase().includes('connection') ||
        log.message.toLowerCase().includes('connected') ||
        log.message.toLowerCase().includes('disconnected')
      ).slice(0, 100),
      warnings: sorted.filter(log => log.level === 'warning').slice(0, 100),
      successes: sorted.filter(log => 
        log.level === 'success' ||
        log.message.toLowerCase().includes('solved') ||
        log.message.toLowerCase().includes('verified') ||
        log.message.toLowerCase().includes('quote')
      ).slice(0, 100),
      errors: sorted.filter(log => log.level === 'error').slice(0, 100),
    }
  }, [logs])

  // Auto-scroll to top when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollAreaRefs[activeTab as keyof typeof scrollAreaRefs]?.current) {
      scrollAreaRefs[activeTab as keyof typeof scrollAreaRefs].current!.scrollTop = 0
    }
  }, [logs.length, activeTab, autoScroll])

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

  const renderLogs = (logsToRender: LogMessage[], category: string) => {
    if (logsToRender.length === 0) {
      return (
        <Text c="dimmed" ta="center" py="xl">
          No {category === 'all' ? 'logs' : category} yet
        </Text>
      )
    }

    return (
      <ScrollArea h={600} ref={scrollAreaRefs[category as keyof typeof scrollAreaRefs]}>
        <Stack gap="xs">
          {logsToRender.map((log, index) => {
            const uniqueKey = `${log.timestamp}-${index}`
            return (
              <Card key={uniqueKey} withBorder p="xs" style={{ backgroundColor: 'var(--mantine-color-dark-8)' }}>
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
                    {log.icon && <span style={{ marginRight: '4px' }}>{log.icon}</span>}
                    {log.message}
                  </Text>
                </Group>
              </Card>
            )
          })}
        </Stack>
      </ScrollArea>
    )
  }

  return (
    <Stack gap="md">
      <Group justify="space-between" align="center">
        <Group gap="xs">
          <Text size="sm" c="dimmed">
            {categorizedLogs[activeTab as keyof typeof categorizedLogs].length} logs
            {categorizedLogs[activeTab as keyof typeof categorizedLogs].length >= 100 && 
              <Text span c="yellow" size="xs"> (limited to latest 100)</Text>
            }
          </Text>
          <Indicator processing disabled={!autoScroll} color="green">
            <Badge 
              size="sm" 
              color={autoScroll ? "green" : "gray"}
              variant="filled"
              style={{ cursor: 'pointer' }}
              onClick={() => setAutoScroll(!autoScroll)}
            >
              {autoScroll ? "🔄 Live" : "⏸️ Paused"}
            </Badge>
          </Indicator>
        </Group>
      </Group>

      <Tabs value={activeTab} onChange={(value) => value && setActiveTab(value)}>
        <Tabs.List>
          <Tabs.Tab value="all" rightSection={<Badge size="xs" variant="light">{categorizedLogs.all.length}</Badge>}>
            All Logs
          </Tabs.Tab>
          <Tabs.Tab value="connections" rightSection={<Badge size="xs" variant="light" color="blue">{categorizedLogs.connections.length}</Badge>}>
            Connections
          </Tabs.Tab>
          <Tabs.Tab value="successes" rightSection={<Badge size="xs" variant="light" color="green">{categorizedLogs.successes.length}</Badge>}>
            Successes
          </Tabs.Tab>
          <Tabs.Tab value="warnings" rightSection={<Badge size="xs" variant="light" color="yellow">{categorizedLogs.warnings.length}</Badge>}>
            Warnings
          </Tabs.Tab>
          <Tabs.Tab value="errors" rightSection={<Badge size="xs" variant="light" color="red">{categorizedLogs.errors.length}</Badge>}>
            Errors
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="all" pt="md">
          {renderLogs(categorizedLogs.all, 'all')}
        </Tabs.Panel>

        <Tabs.Panel value="connections" pt="md">
          {renderLogs(categorizedLogs.connections, 'connections')}
        </Tabs.Panel>

        <Tabs.Panel value="successes" pt="md">
          {renderLogs(categorizedLogs.successes, 'successes')}
        </Tabs.Panel>

        <Tabs.Panel value="warnings" pt="md">
          {renderLogs(categorizedLogs.warnings, 'warnings')}
        </Tabs.Panel>

        <Tabs.Panel value="errors" pt="md">
          {renderLogs(categorizedLogs.errors, 'errors')}
        </Tabs.Panel>
      </Tabs>
    </Stack>
  )
}