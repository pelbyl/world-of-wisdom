import { Group, Card, Text, Badge, Stack, ScrollArea, Tooltip, Pagination, Center, ActionIcon } from '@mantine/core'
import { IconLink, IconHash, IconClock, IconChevronLeft, IconChevronRight } from '@tabler/icons-react'
import { useState, useMemo } from 'react'
import { Block } from '../types'

interface Props {
  blocks: Block[]
}

export function BlockchainVisualizer({ blocks }: Props) {
  const [activePage, setActivePage] = useState(1)
  const blocksPerPage = 6 // Show 6 blocks per page for better performance

  const paginatedBlocks = useMemo<{
    blocks: Block[];
    totalPages: number;
    totalBlocks: number;
  }>(() => {
    if (blocks.length === 0) {
      return {
        blocks: [],
        totalPages: 0,
        totalBlocks: 0,
      }
    }

    // Reverse array to show latest blocks first
    const reversedBlocks = [...blocks].reverse()
    const totalPages = Math.ceil(reversedBlocks.length / blocksPerPage)

    // Auto-navigate to latest page when new blocks arrive
    if (activePage === 1 && blocks.length > 0) {
      // Keep showing latest blocks (page 1)
    }

    const startIndex = (activePage - 1) * blocksPerPage
    const endIndex = startIndex + blocksPerPage

    return {
      blocks: reversedBlocks.slice(startIndex, endIndex),
      totalPages,
      totalBlocks: blocks.length
    }
  }, [blocks, activePage, blocksPerPage])

  if (blocks.length === 0) {
    return <Text c="dimmed">No blocks mined yet...</Text>
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Text size="sm" c="dimmed">
          Total blocks: {paginatedBlocks.totalBlocks} | Showing page {activePage} of {paginatedBlocks.totalPages}
        </Text>
        {paginatedBlocks.totalPages > 1 && (
          <Group gap="xs">
            <ActionIcon
              variant="light"
              disabled={activePage === 1}
              onClick={() => setActivePage(1)}
            >
              <IconChevronLeft size={16} />
            </ActionIcon>
            <Text size="sm">Latest</Text>
            <ActionIcon
              variant="light"
              disabled={activePage === paginatedBlocks.totalPages}
              onClick={() => setActivePage(paginatedBlocks.totalPages)}
            >
              <IconChevronRight size={16} />
            </ActionIcon>
            <Text size="sm">Oldest</Text>
          </Group>
        )}
      </Group>

      <ScrollArea h={400}>
        <Group align="stretch" wrap="nowrap">
          {paginatedBlocks.blocks.map((block, index) => (
            <Group key={block.index} wrap="nowrap">
              <Card
                withBorder
                w={350}
                h={350}
                padding="md"
                style={{
                  borderColor: block.solution ? '#40c057' : '#fd7e14',
                  borderWidth: 2
                }}
              >
                <Stack gap="xs">
                  <Group justify="space-between">
                    <Badge size="lg" color={block.solution ? 'green' : 'orange'}>
                      Block #{block.index}
                    </Badge>
                    <Badge color="gray" size="sm">
                      Difficulty {block.challenge.difficulty}
                    </Badge>
                  </Group>

                  <Group gap="xs">
                    <IconClock size={16} />
                    <Text size="xs" c="dimmed">
                      {new Date(block.timestamp).toLocaleTimeString()}
                    </Text>
                  </Group>

                  {block.solution && (
                    <>
                      <Group gap="xs">
                        <IconHash size={16} />
                        <Tooltip label={block.hash}>
                          <Text size="xs" style={{ fontFamily: 'monospace' }}>
                            {block.hash.substring(0, 16)}...
                          </Text>
                        </Tooltip>
                      </Group>

                      <Text size="xs" c="dimmed">
                        Nonce: {block.solution.nonce}
                      </Text>

                      <Text size="xs" c="dimmed">
                        Attempts: {block.solution.attempts.toLocaleString()}
                      </Text>

                      <Text size="xs" c="dimmed">
                        Time: {(block.solution.timeToSolve / 1000).toFixed(2)}s
                      </Text>

                      {block.quote && (
                        <Tooltip label={block.quote}>
                          <Text size="sm" lineClamp={4} c="blue" fw={500}>
                            "{block.quote.substring(0, 120)}..."
                          </Text>
                        </Tooltip>
                      )}
                    </>
                  )}
                </Stack>
              </Card>

              {index < paginatedBlocks.blocks.length - 1 && (
                <IconLink size={30} style={{ color: '#868e96' }} />
              )}
            </Group>
          ))}
        </Group>
      </ScrollArea>

      {paginatedBlocks.totalPages > 1 && (
        <Center>
          <Pagination
            value={activePage}
            onChange={setActivePage}
            total={paginatedBlocks.totalPages}
            size="sm"
            withEdges
          />
        </Center>
      )}
    </Stack>
  )
}