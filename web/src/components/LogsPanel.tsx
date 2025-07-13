import { Stack, Text, ScrollArea, Group, Badge, Card, Pagination, Center, Button } from '@mantine/core'
import { LogMessage } from '../types'
import { useEffect, useRef, useState, useMemo } from 'react'

interface Props {
  logs: LogMessage[]
}

const LOGS_PER_PAGE = 20

export function LogsPanel({ logs }: Props) {
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [autoScroll, setAutoScroll] = useState(true)

  // Sort logs with latest first
  const sortedLogs = useMemo(() => {
    return [...logs].sort((a, b) => b.timestamp - a.timestamp)
  }, [logs])

  // Paginated logs
  const paginatedLogs = useMemo(() => {
    const start = (currentPage - 1) * LOGS_PER_PAGE
    const end = start + LOGS_PER_PAGE
    return sortedLogs.slice(start, end)
  }, [sortedLogs, currentPage])

  const totalPages = Math.ceil(sortedLogs.length / LOGS_PER_PAGE)

  // Reset to first page when new logs arrive (if on first page)
  useEffect(() => {
    if (currentPage === 1 && autoScroll) {
      // Stay on first page to see latest logs
    }
  }, [logs.length, currentPage, autoScroll])

  // Auto-scroll to top when on first page and new logs arrive
  useEffect(() => {
    if (currentPage === 1 && autoScroll && scrollAreaRef.current) {
      scrollAreaRef.current.scrollTop = 0
    }
  }, [paginatedLogs, currentPage, autoScroll])

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
    <Stack gap="md">
      {/* Controls */}
      <Group justify="space-between" align="center">
        <Text size="sm" c="dimmed">
          {sortedLogs.length} logs total • Page {currentPage} of {totalPages}
        </Text>
        <Group gap="xs">
          <Button
            size="xs"
            variant={autoScroll ? "filled" : "light"}
            onClick={() => setAutoScroll(!autoScroll)}
            color={autoScroll ? "green" : "gray"}
          >
            {autoScroll ? "🔄 Live" : "⏸️ Paused"}
          </Button>
          {currentPage !== 1 && (
            <Button
              size="xs"
              variant="light"
              onClick={() => setCurrentPage(1)}
            >
              📄 Latest
            </Button>
          )}
        </Group>
      </Group>

      {/* Logs */}
      <ScrollArea h={600} ref={scrollAreaRef}>
        <Stack gap="xs">
          {paginatedLogs.map((log, index) => {
            // Create unique key using timestamp and index for better React reconciliation
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

      {/* Pagination */}
      {totalPages > 1 && (
        <Center>
          <Pagination
            total={totalPages}
            value={currentPage}
            onChange={setCurrentPage}
            size="sm"
            withEdges
          />
        </Center>
      )}
    </Stack>
  )
}