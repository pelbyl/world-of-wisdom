import { Stack, Text, Group, Badge, ScrollArea, Card } from '@mantine/core'
import { IconHash } from '@tabler/icons-react'
import { ChallengeDetail } from '../types/api'
import { useChallenges } from '../hooks/useV1API'

export function ChallengePanel() {
  const { data: challengesResponse } = useChallenges(20)
  
  const challenges = challengesResponse?.status === 'success' && challengesResponse.data ? challengesResponse.data.challenges : []
  const recentChallenges = challenges.slice(-10).reverse()

  return (
    <Stack gap="md" style={{ height: '100%' }}>
      <Text size="sm" c="dimmed">
        Recent challenges from connected TCP clients
      </Text>

      <ScrollArea style={{ flex: 1 }}>
        <Stack gap="xs">
          {recentChallenges.length === 0 ? (
            <Text c="dimmed" ta="center">No recent challenges</Text>
          ) : (
            recentChallenges.map((challenge: ChallengeDetail) => (
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
                      {new Date(challenge.createdAt).toLocaleTimeString()}
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