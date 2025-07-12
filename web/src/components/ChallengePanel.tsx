import { Stack, Text, Group, Badge, Button, ScrollArea, Card } from '@mantine/core'
import { IconPlayerPlay, IconHash } from '@tabler/icons-react'
import { Challenge } from '../types'

interface Props {
  challenges: Challenge[]
  onSimulateClient: () => void
}

export function ChallengePanel({ challenges, onSimulateClient }: Props) {
  const recentChallenges = challenges.slice(-10).reverse()

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Text size="lg" fw={500}>Recent Challenges</Text>
        <Button
          leftSection={<IconPlayerPlay size={16} />}
          size="sm"
          onClick={onSimulateClient}
        >
          Simulate Client
        </Button>
      </Group>

      <ScrollArea h={300}>
        <Stack gap="xs">
          {recentChallenges.length === 0 ? (
            <Text c="dimmed" ta="center">No challenges yet</Text>
          ) : (
            recentChallenges.map(challenge => (
              <Card key={challenge.id} withBorder padding="sm">
                <Group justify="space-between">
                  <Group gap="xs">
                    <IconHash size={16} />
                    <Text size="sm" style={{ fontFamily: 'monospace' }}>
                      {challenge.seed.substring(0, 12)}...
                    </Text>
                    <Badge size="sm" color="blue">
                      D{challenge.difficulty}
                    </Badge>
                  </Group>
                  
                  <Group gap="xs">
                    <Badge 
                      size="sm" 
                      color={
                        challenge.status === 'completed' ? 'green' :
                        challenge.status === 'solving' ? 'blue' :
                        challenge.status === 'failed' ? 'red' : 'gray'
                      }
                    >
                      {challenge.status}
                    </Badge>
                    <Text size="xs" c="dimmed">
                      {new Date(challenge.timestamp).toLocaleTimeString()}
                    </Text>
                  </Group>
                </Group>
              </Card>
            ))
          )}
        </Stack>
      </ScrollArea>
    </Stack>
  )
}