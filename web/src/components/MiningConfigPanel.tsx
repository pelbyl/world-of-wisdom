import { 
  Stack, 
  Text, 
  Group, 
  Button, 
  NumberInput, 
  Switch, 
  Divider,
  Card,
  Grid,
  Badge,
  Tooltip
} from '@mantine/core'
import { 
  IconPlayerPlay, 
  IconPlayerStop, 
  IconSettings,
  IconCpu,
  IconUsers,
  IconClock,
  IconBolt
} from '@tabler/icons-react'
import { useState } from 'react'

interface MiningConfig {
  initialIntensity: number
  maxIntensity: number
  intensityStep: number
  autoScale: boolean
  minMiners: number
  maxMiners: number
  duration: number
}

interface Props {
  onStartMining: (config?: MiningConfig) => void
  onStopMining: () => void
  miningActive: boolean
}

export function MiningConfigPanel({ onStartMining, onStopMining, miningActive }: Props) {
  const [config, setConfig] = useState<MiningConfig>({
    initialIntensity: 1,
    maxIntensity: 3,
    intensityStep: 30,
    autoScale: true,
    minMiners: 2,
    maxMiners: 25,
    duration: 0
  })

  const [showAdvanced, setShowAdvanced] = useState(false)

  const handleQuickStart = (preset: 'demo' | 'stress' | 'ddos') => {
    let quickConfig: MiningConfig
    
    switch (preset) {
      case 'demo':
        quickConfig = {
          initialIntensity: 1,
          maxIntensity: 2,
          intensityStep: 20,
          autoScale: true,
          minMiners: 2,
          maxMiners: 8,
          duration: 60
        }
        break
      case 'stress':
        quickConfig = {
          initialIntensity: 2,
          maxIntensity: 3,
          intensityStep: 15,
          autoScale: true,
          minMiners: 5,
          maxMiners: 20,
          duration: 120
        }
        break
      case 'ddos':
        quickConfig = {
          initialIntensity: 3,
          maxIntensity: 3,
          intensityStep: 10,
          autoScale: false,
          minMiners: 15,
          maxMiners: 30,
          duration: 180
        }
        break
    }
    
    setConfig(quickConfig)
    onStartMining(quickConfig)
  }

  const handleCustomStart = () => {
    onStartMining(config)
  }

  const getIntensityColor = (level: number) => {
    switch (level) {
      case 1: return 'green'
      case 2: return 'yellow'
      case 3: return 'red'
      default: return 'gray'
    }
  }

  return (
    <Stack gap="md">
      {/* Quick Start Presets */}
      <Card withBorder p="md">
        <Text size="sm" fw={500} mb="md">🚀 Quick Start Presets</Text>
        <Grid>
          <Grid.Col span={4}>
            <Tooltip label="Light demo - 2-8 miners, 60 seconds">
              <Button
                fullWidth
                variant="light"
                color="green"
                leftSection={<IconCpu size={16} />}
                onClick={() => handleQuickStart('demo')}
                disabled={miningActive}
              >
                Demo Mode
              </Button>
            </Tooltip>
          </Grid.Col>
          
          <Grid.Col span={4}>
            <Tooltip label="Medium load - 5-20 miners, 2 minutes">
              <Button
                fullWidth
                variant="light"
                color="yellow"
                leftSection={<IconBolt size={16} />}
                onClick={() => handleQuickStart('stress')}
                disabled={miningActive}
              >
                Stress Test
              </Button>
            </Tooltip>
          </Grid.Col>
          
          <Grid.Col span={4}>
            <Tooltip label="Heavy load - triggers DDoS protection, 3 minutes">
              <Button
                fullWidth
                variant="light"
                color="red"
                leftSection={<IconUsers size={16} />}
                onClick={() => handleQuickStart('ddos')}
                disabled={miningActive}
              >
                DDoS Demo
              </Button>
            </Tooltip>
          </Grid.Col>
        </Grid>
      </Card>

      <Divider label="Advanced Configuration" />

      {/* Advanced Configuration */}
      <Group justify="space-between">
        <Text size="sm" fw={500}>🛠️ Custom Configuration</Text>
        <Button
          size="xs"
          variant="subtle"
          leftSection={<IconSettings size={14} />}
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          {showAdvanced ? 'Hide' : 'Show'} Advanced
        </Button>
      </Group>

      {showAdvanced && (
        <Card withBorder p="md">
          <Stack gap="md">
            <Group grow>
              <Tooltip label="Starting network intensity level">
                <NumberInput
                  label="Initial Intensity"
                  min={1}
                  max={3}
                  value={config.initialIntensity}
                  onChange={(val) => setConfig({...config, initialIntensity: Number(val)})}
                  rightSection={
                    <Badge size="xs" color={getIntensityColor(config.initialIntensity)}>
                      {config.initialIntensity}
                    </Badge>
                  }
                />
              </Tooltip>
              
              <Tooltip label="Maximum network intensity level">
                <NumberInput
                  label="Max Intensity"
                  min={1}
                  max={3}
                  value={config.maxIntensity}
                  onChange={(val) => setConfig({...config, maxIntensity: Number(val)})}
                  rightSection={
                    <Badge size="xs" color={getIntensityColor(config.maxIntensity)}>
                      {config.maxIntensity}
                    </Badge>
                  }
                />
              </Tooltip>
            </Group>

            <Group grow>
              <Tooltip label="Minimum concurrent miners">
                <NumberInput
                  label="Min Miners"
                  min={1}
                  max={50}
                  value={config.minMiners}
                  onChange={(val) => setConfig({...config, minMiners: Number(val)})}
                />
              </Tooltip>
              
              <Tooltip label="Maximum concurrent miners">
                <NumberInput
                  label="Max Miners"
                  min={1}
                  max={50}
                  value={config.maxMiners}
                  onChange={(val) => setConfig({...config, maxMiners: Number(val)})}
                />
              </Tooltip>
            </Group>

            <Group grow>
              <Tooltip label="Seconds between intensity changes">
                <NumberInput
                  label="Intensity Step (sec)"
                  min={5}
                  max={300}
                  value={config.intensityStep}
                  onChange={(val) => setConfig({...config, intensityStep: Number(val)})}
                  leftSection={<IconClock size={16} />}
                />
              </Tooltip>
              
              <Tooltip label="Simulation duration (0 = infinite)">
                <NumberInput
                  label="Duration (sec)"
                  min={0}
                  max={3600}
                  value={config.duration}
                  onChange={(val) => setConfig({...config, duration: Number(val)})}
                  leftSection={<IconClock size={16} />}
                />
              </Tooltip>
            </Group>

            <Switch
              label="Auto-Scale Intensity"
              description="Automatically increase network intensity over time"
              checked={config.autoScale}
              onChange={(event) => setConfig({...config, autoScale: event.currentTarget.checked})}
            />
          </Stack>
        </Card>
      )}

      {/* Control Buttons */}
      <Group grow>
        {!miningActive ? (
          <>
            <Button
              leftSection={<IconPlayerPlay size={16} />}
              color="blue"
              onClick={() => onStartMining()}
              variant="light"
            >
              Simple Start
            </Button>
            <Button
              leftSection={<IconPlayerPlay size={16} />}
              color="green"
              onClick={handleCustomStart}
              disabled={!showAdvanced}
            >
              Custom Start
            </Button>
          </>
        ) : (
          <Button
            fullWidth
            leftSection={<IconPlayerStop size={16} />}
            color="red"
            onClick={onStopMining}
          >
            Stop Mining
          </Button>
        )}
      </Group>

      <Text size="xs" c="dimmed" ta="center">
        {miningActive 
          ? "🟢 Network simulation is active - monitor logs for real-time activity"
          : "⚡ Configure and start your blockchain network simulation"
        }
      </Text>
    </Stack>
  )
}