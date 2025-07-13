import { Stack, Text, ScrollArea, Group, Badge, Card, Pagination, Center, Button, Select, TextInput, ActionIcon } from '@mantine/core'
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
  const [levelFilter, setLevelFilter] = useState<string | null>(null)
  const [searchFilter, setSearchFilter] = useState('')

  // Filter and sort logs
  const filteredLogs = useMemo(() => {
    let filtered = [...logs]
    
    // Filter by level
    if (levelFilter && levelFilter !== 'all') {
      filtered = filtered.filter(log => log.level === levelFilter)
    }
    
    // Filter by search term
    if (searchFilter.trim()) {
      const searchTerm = searchFilter.toLowerCase().trim()
      filtered = filtered.filter(log => 
        log.message.toLowerCase().includes(searchTerm) ||
        log.level.toLowerCase().includes(searchTerm)
      )
    }
    
    // Sort with latest first
    return filtered.sort((a, b) => b.timestamp - a.timestamp)
  }, [logs, levelFilter, searchFilter])

  // Paginated logs
  const paginatedLogs = useMemo(() => {
    const start = (currentPage - 1) * LOGS_PER_PAGE
    const end = start + LOGS_PER_PAGE
    return filteredLogs.slice(start, end)
  }, [filteredLogs, currentPage])

  const totalPages = Math.ceil(filteredLogs.length / LOGS_PER_PAGE)
  
  // Reset to first page when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [levelFilter, searchFilter])

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
      {/* Filters */}
      <Stack gap="sm">
        <Group gap="md" align="end">
          <Select
            label="Filter by Level"
            placeholder="All levels"
            size="xs"
            style={{ width: 120 }}
            data={[
              { value: 'all', label: 'All' },
              { value: 'info', label: '📘 Info' },
              { value: 'success', label: '✅ Success' },
              { value: 'warning', label: '⚠️ Warning' },
              { value: 'error', label: '❌ Error' },
            ]}
            value={levelFilter}
            onChange={setLevelFilter}
            clearable
          />
          <TextInput
            label="Search"
            placeholder="Search logs..."
            size="xs"
            style={{ flex: 1, maxWidth: 200 }}
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            rightSection={
              searchFilter && (
                <ActionIcon
                  size="xs"
                  variant="subtle"
                  onClick={() => setSearchFilter('')}
                >
                  ✕
                </ActionIcon>
              )
            }
          />
        </Group>
        
        {/* Controls */}
        <Group justify="space-between" align="center">
          <Text size="sm" c="dimmed">
            {filteredLogs.length} logs 
            {logs.length !== filteredLogs.length && ` (filtered from ${logs.length})`} 
            • Page {currentPage} of {totalPages}
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
            {(levelFilter || searchFilter) && (
              <Button
                size="xs"
                variant="light"
                color="red"
                onClick={() => {
                  setLevelFilter(null)
                  setSearchFilter('')
                }}
              >
                🗑️ Clear Filters
              </Button>
            )}
          </Group>
        </Group>
      </Stack>

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