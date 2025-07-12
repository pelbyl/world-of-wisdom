import { Stack, Text, Group, Badge, Button, ScrollArea, Card, Alert, Progress } from '@mantine/core'
import { IconPlayerPlay, IconPlayerStop, IconHash, IconInfoCircle, IconCpu } from '@tabler/icons-react'
import { Challenge } from '../types'
import { useState } from 'react'

interface Props {
  challenges: Challenge[]
  onSimulateClient: () => void
  onStartMining: () => void
  onStopMining: () => void
  miningActive: boolean
}

export function ChallengePanel({ challenges, onSimulateClient, onStartMining, onStopMining, miningActive }: Props) {
  const [isSimulating, setIsSimulating] = useState(false)
  const recentChallenges = challenges.slice(-10).reverse()

  const handleSimulateClient = () => {
    setIsSimulating(true)
    onSimulateClient()
    // Reset after 5 seconds (typical mining time)
    setTimeout(() => setIsSimulating(false), 5000)
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Text size="lg" fw={500}>Mining Simulation</Text>
        <Group gap="sm">
          <Button
            leftSection={isSimulating ? <IconCpu size={16} /> : <IconPlayerPlay size={16} />}
            size="sm"
            onClick={handleSimulateClient}
            loading={isSimulating}
            disabled={isSimulating}
            variant="light"
          >
            {isSimulating ? 'Mining...' : 'Single Client'}
          </Button>
          
          {!miningActive ? (
            <Button
              leftSection={<IconPlayerPlay size={16} />}
              size="sm"
              onClick={onStartMining}
              color="green"
            >
              Start Auto Mining
            </Button>
          ) : (
            <Button
              leftSection={<IconPlayerStop size={16} />}
              size="sm"
              onClick={onStopMining}
              color="red"
            >
              Stop Auto Mining
            </Button>
          )}
        </Group>
      </Group>

      {miningActive && (
        <Alert icon={<IconCpu size={16} />} color="green">
          <Text size="sm">
            <strong>Continuous Mining Active!</strong>
            <br />
            • Automatically spawning new clients every 2-6 seconds
            <br />
            • Simulating real blockchain network with multiple miners
            <br />
            • Watch the blockchain grow continuously!
          </Text>
          <Progress size="sm" animated color="green" mt="sm" value={100} />
        </Alert>
      )}

      {isSimulating && (
        <Alert icon={<IconInfoCircle size={16} />} color="blue">
          <Text size="sm">
            <strong>Single Mining Process Started!</strong>
            <br />
            • Client connecting to TCP server
            <br />
            • Receiving PoW challenge (difficulty {challenges.length > 0 ? challenges[challenges.length - 1]?.difficulty || 'unknown' : 'unknown'})
            <br />
            • Computing SHA-256 hash solutions...
          </Text>
          <Progress size="sm" animated color="blue" mt="sm" value={100} />
        </Alert>
      )}

      <Text size="sm" c="dimmed">
        Use "Single Client" for one-time mining or "Start Auto Mining" for continuous blockchain simulation like a real network!
      </Text>

      <Text size="sm" fw={500}>Recent Mining Activity</Text>
      <ScrollArea h={300}>
        <Stack gap="xs">
          {recentChallenges.length === 0 ? (
            <Text c="dimmed" ta="center">No mining activity yet - click "Simulate Client" to start!</Text>
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