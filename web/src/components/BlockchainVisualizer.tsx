import { Group, Card, Text, Badge, Stack, ScrollArea, Tooltip } from '@mantine/core'
import { IconLink, IconHash, IconClock } from '@tabler/icons-react'
import { Block } from '../types'

interface Props {
  blocks: Block[]
}

export function BlockchainVisualizer({ blocks }: Props) {
  if (blocks.length === 0) {
    return <Text c="dimmed">No blocks mined yet...</Text>
  }

  return (
    <ScrollArea h={200}>
      <Group align="stretch" wrap="nowrap">
        {blocks.map((block, index) => (
          <Group key={block.index} wrap="nowrap">
            <Card 
              withBorder 
              w={300} 
              padding="sm"
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
                        <Text size="xs" lineClamp={2} c="blue">
                          "{block.quote.substring(0, 50)}..."
                        </Text>
                      </Tooltip>
                    )}
                  </>
                )}
              </Stack>
            </Card>

            {index < blocks.length - 1 && (
              <IconLink size={30} style={{ color: '#868e96' }} />
            )}
          </Group>
        ))}
      </Group>
    </ScrollArea>
  )
}