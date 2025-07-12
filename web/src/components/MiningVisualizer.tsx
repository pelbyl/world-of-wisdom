import { Stack, Progress, Text, Group, Badge, Card } from '@mantine/core'
import { Challenge } from '../types'
import { useEffect, useState } from 'react'

interface Props {
  challenges: Challenge[]
}

export function MiningVisualizer({ challenges }: Props) {
  const [progress, setProgress] = useState<Record<string, number>>({})

  useEffect(() => {
    const interval = setInterval(() => {
      setProgress(prev => {
        const updated = { ...prev }
        challenges.forEach(challenge => {
          if (challenge.status === 'solving') {
            updated[challenge.id] = Math.min((updated[challenge.id] || 0) + Math.random() * 10, 100)
          } else if (challenge.status === 'completed') {
            updated[challenge.id] = 100
          }
        })
        return updated
      })
    }, 100)

    return () => clearInterval(interval)
  }, [challenges])

  const activeChallenges = challenges.filter(c => 
    c.status === 'solving' || c.status === 'pending'
  ).slice(-5)

  if (activeChallenges.length === 0) {
    return <Text c="dimmed">No active mining operations...</Text>
  }

  return (
    <Stack gap="md">
      {activeChallenges.map(challenge => (
        <Card key={challenge.id} withBorder padding="sm">
          <Stack gap="xs">
            <Group justify="space-between">
              <Group gap="xs">
                <Text size="sm" fw={500}>Client {challenge.clientId.substring(0, 8)}</Text>
                <Badge size="sm" color={challenge.status === 'solving' ? 'blue' : 'gray'}>
                  {challenge.status}
                </Badge>
              </Group>
              <Badge size="sm" color="orange">
                Difficulty {challenge.difficulty}
              </Badge>
            </Group>

            <Text size="xs" c="dimmed" style={{ fontFamily: 'monospace' }}>
              Seed: {challenge.seed.substring(0, 16)}...
            </Text>

            {challenge.status === 'solving' && (
              <Progress 
                value={progress[challenge.id] || 0} 
                animated 
                color="blue"
                size="sm"
              />
            )}

            <Text size="xs" c="dimmed">
              Started: {new Date(challenge.timestamp).toLocaleTimeString()}
            </Text>
          </Stack>
        </Card>
      ))}
    </Stack>
  )
}