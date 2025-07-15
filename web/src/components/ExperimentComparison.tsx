import React, { useState, useEffect } from 'react';
import {
  Card,
  Title,
  Text,
  Grid,
  Table,
  Badge,
  Group,
  Stack,
  Button,
  MultiSelect,
  Paper,
  ThemeIcon,
  Progress,
  Tooltip,
  ActionIcon,
  Alert,
} from '@mantine/core';
import {
  LineChart,
  BarChart,
  RadarChart,
} from '@mantine/charts';
import {
  IconCheck,
  IconX,
  IconTrendingUp,
  IconTrendingDown,
  IconInfoCircle,
  IconDownload,
  IconRefresh,
} from '@tabler/icons-react';

interface ScenarioResult {
  name: string;
  totalClients: number;
  normalClients: number;
  attackers: number;
  avgDifficulty: number;
  falsePositives: number;
  avgNormalSolveTime: number;
  detectionTime: number;
  successRate: number;
}

// Experiment Comparison Dashboard
export const ExperimentComparison: React.FC = () => {
  const [selectedScenarios, setSelectedScenarios] = useState<string[]>([
    'morning-rush',
    'script-kiddie',
    'ddos',
  ]);
  const [scenarioResults, setScenarioResults] = useState<Record<string, ScenarioResult>>({});
  const [isLoading, setIsLoading] = useState(false);

  const scenarios = [
    { value: 'morning-rush', label: 'Morning Rush' },
    { value: 'script-kiddie', label: 'Script Kiddie Attack' },
    { value: 'ddos', label: 'Sophisticated DDoS' },
    { value: 'botnet', label: 'Botnet Simulation' },
    { value: 'mixed', label: 'Mixed Reality' },
  ];

  const loadScenarioData = async () => {
    setIsLoading(true);
    try {
      // In a real implementation, this would load historical data for each scenario
      // For now, we'll simulate with current data

      // Simulate different results for each scenario
      const mockResults: Record<string, ScenarioResult> = {
        'morning-rush': {
          name: 'Morning Rush',
          totalClients: 15,
          normalClients: 15,
          attackers: 0,
          avgDifficulty: 1.8,
          falsePositives: 0,
          avgNormalSolveTime: 1500,
          detectionTime: 0,
          successRate: 100,
        },
        'script-kiddie': {
          name: 'Script Kiddie',
          totalClients: 6,
          normalClients: 5,
          attackers: 1,
          avgDifficulty: 2.3,
          falsePositives: 0,
          avgNormalSolveTime: 1800,
          detectionTime: 45,
          successRate: 95,
        },
        'ddos': {
          name: 'DDoS Attack',
          totalClients: 13,
          normalClients: 10,
          attackers: 3,
          avgDifficulty: 3.1,
          falsePositives: 1,
          avgNormalSolveTime: 2100,
          detectionTime: 30,
          successRate: 88,
        },
        'botnet': {
          name: 'Botnet',
          totalClients: 28,
          normalClients: 8,
          attackers: 20,
          avgDifficulty: 4.2,
          falsePositives: 2,
          avgNormalSolveTime: 2500,
          detectionTime: 25,
          successRate: 82,
        },
        'mixed': {
          name: 'Mixed Reality',
          totalClients: 20,
          normalClients: 12,
          attackers: 8,
          avgDifficulty: 3.5,
          falsePositives: 1,
          avgNormalSolveTime: 2300,
          detectionTime: 35,
          successRate: 90,
        },
      };

      setScenarioResults(mockResults);
    } catch (error) {
      console.error('Failed to load scenario data:', error);
    }
    setIsLoading(false);
  };

  useEffect(() => {
    loadScenarioData();
  }, []);

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Experiment Comparison</Title>
        <Group>
          <Button
            leftSection={<IconRefresh size={16} />}
            onClick={loadScenarioData}
            loading={isLoading}
            variant="light"
          >
            Refresh Data
          </Button>
          <Button leftSection={<IconDownload size={16} />} variant="subtle">
            Export Comparison
          </Button>
        </Group>
      </Group>

      <Paper p="md" withBorder>
        <MultiSelect
          label="Select scenarios to compare"
          placeholder="Choose scenarios"
          data={scenarios}
          value={selectedScenarios}
          onChange={setSelectedScenarios}
          maxValues={5}
        />
      </Paper>

      <Grid>
        <Grid.Col span={{ base: 12 }}>
          <ComparisonTable 
            scenarios={selectedScenarios.map(s => scenarioResults[s]).filter(Boolean)} 
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <PerformanceRadarChart 
            scenarios={selectedScenarios.map(s => scenarioResults[s]).filter(Boolean)} 
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <DetectionTimeComparison 
            scenarios={selectedScenarios.map(s => scenarioResults[s]).filter(Boolean)} 
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12 }}>
          <SuccessRateComparison 
            scenarios={selectedScenarios.map(s => scenarioResults[s]).filter(Boolean)} 
          />
        </Grid.Col>
      </Grid>
    </Stack>
  );
};

// Comparison Table Component
const ComparisonTable: React.FC<{ scenarios: ScenarioResult[] }> = ({ scenarios }) => {
  const getBadgeColor = (value: number, type: 'difficulty' | 'time' | 'rate') => {
    if (type === 'difficulty') {
      return value <= 2 ? 'green' : value <= 3.5 ? 'yellow' : 'red';
    } else if (type === 'time') {
      return value <= 30 ? 'green' : value <= 60 ? 'yellow' : 'red';
    } else {
      return value >= 90 ? 'green' : value >= 75 ? 'yellow' : 'red';
    }
  };

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Scenario Comparison</Title>
      <Table.ScrollContainer minWidth={800}>
        <Table highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Scenario</Table.Th>
              <Table.Th>Total Clients</Table.Th>
              <Table.Th>Attackers</Table.Th>
              <Table.Th>Avg Difficulty</Table.Th>
              <Table.Th>Detection Time</Table.Th>
              <Table.Th>False Positives</Table.Th>
              <Table.Th>Normal User Impact</Table.Th>
              <Table.Th>Success Rate</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {scenarios.map((scenario) => (
              <Table.Tr key={scenario.name}>
                <Table.Td fw={500}>{scenario.name}</Table.Td>
                <Table.Td>{scenario.totalClients}</Table.Td>
                <Table.Td>
                  <Badge color={scenario.attackers > 0 ? 'red' : 'green'} variant="light">
                    {scenario.attackers}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Badge color={getBadgeColor(scenario.avgDifficulty, 'difficulty')} variant="light">
                    {scenario.avgDifficulty.toFixed(1)}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  {scenario.detectionTime > 0 ? (
                    <Badge color={getBadgeColor(scenario.detectionTime, 'time')} variant="light">
                      {scenario.detectionTime}s
                    </Badge>
                  ) : (
                    <Text size="sm" c="dimmed">N/A</Text>
                  )}
                </Table.Td>
                <Table.Td>
                  <Badge color={scenario.falsePositives > 0 ? 'orange' : 'green'} variant="light">
                    {scenario.falsePositives}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <Text size="sm">{(scenario.avgNormalSolveTime / 1000).toFixed(1)}s</Text>
                    {scenario.avgNormalSolveTime < 2000 ? (
                      <IconCheck size={16} color="green" />
                    ) : (
                      <IconX size={16} color="red" />
                    )}
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <Progress 
                      value={scenario.successRate} 
                      color={getBadgeColor(scenario.successRate, 'rate')}
                      size="sm"
                      w={60}
                    />
                    <Text size="sm" fw={500}>{scenario.successRate}%</Text>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Table.ScrollContainer>
    </Card>
  );
};

// Performance Radar Chart
const PerformanceRadarChart: React.FC<{ scenarios: ScenarioResult[] }> = ({ scenarios }) => {
  const radarData = scenarios.map(scenario => ({
    scenario: scenario.name,
    'Detection Speed': Math.max(0, 100 - scenario.detectionTime),
    'User Experience': Math.max(0, 100 - (scenario.avgNormalSolveTime / 30)),
    'Accuracy': 100 - (scenario.falsePositives * 10),
    'Effectiveness': scenario.successRate,
    'Scalability': Math.max(0, 100 - (scenario.avgDifficulty * 15)),
  }));

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Performance Comparison</Title>
      <RadarChart
        h={300}
        data={radarData}
        dataKey="scenario"
        series={Object.keys(radarData[0] || {}).filter(k => k !== 'scenario').map(key => ({
          name: key,
          color: `var(--mantine-color-${['blue', 'green', 'orange', 'red', 'purple'][
            Object.keys(radarData[0] || {}).indexOf(key) % 5
          ]}-6)`,
          opacity: 0.2,
        }))}
        withPolarGrid
        withPolarAngleAxis
        withPolarRadiusAxis
      />
    </Card>
  );
};

// Detection Time Comparison
const DetectionTimeComparison: React.FC<{ scenarios: ScenarioResult[] }> = ({ scenarios }) => {
  const chartData = scenarios.map(scenario => ({
    scenario: scenario.name,
    detectionTime: scenario.detectionTime,
    targetTime: 30, // Target detection time
  }));

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Group justify="space-between" mb="md">
        <Title order={3}>Detection Time Analysis</Title>
        <Tooltip label="Time taken to identify and respond to attacks">
          <ActionIcon variant="subtle" size="sm">
            <IconInfoCircle size={16} />
          </ActionIcon>
        </Tooltip>
      </Group>
      <BarChart
        h={300}
        data={chartData}
        dataKey="scenario"
        series={[
          { name: 'detectionTime', color: 'indigo.6', label: 'Detection Time (s)' },
          { name: 'targetTime', color: 'gray.4', label: 'Target Time (s)' },
        ]}
        tickLine="y"
        withLegend
        legendProps={{ verticalAlign: 'bottom' }}
      />
      <Alert mt="md" color="blue" variant="light">
        <Text size="sm">
          Target: Detect and respond to attacks within 30 seconds
        </Text>
      </Alert>
    </Card>
  );
};

// Success Rate Comparison
const SuccessRateComparison: React.FC<{ scenarios: ScenarioResult[] }> = ({ scenarios }) => {
  const chartData = scenarios.map(scenario => ({
    scenario: scenario.name,
    successRate: scenario.successRate,
    normalUserExperience: 100 - (scenario.avgNormalSolveTime / 30),
    attackerMitigation: scenario.attackers > 0 ? 90 : 0,
  }));

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Success Metrics Overview</Title>
      <LineChart
        h={300}
        data={chartData}
        dataKey="scenario"
        series={[
          { name: 'successRate', color: 'green.6', label: 'Overall Success (%)' },
          { name: 'normalUserExperience', color: 'blue.6', label: 'User Experience (%)' },
          { name: 'attackerMitigation', color: 'red.6', label: 'Attack Mitigation (%)' },
        ]}
        curveType="linear"
        withLegend
        legendProps={{ verticalAlign: 'bottom' }}
        withDots
      />
      
      <Grid mt="md">
        {scenarios.map((scenario) => (
          <Grid.Col key={scenario.name} span={{ base: 12, sm: 6, md: 4 }}>
            <Paper p="sm" withBorder>
              <Text size="sm" fw={500} mb="xs">{scenario.name}</Text>
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text size="xs" c="dimmed">Overall</Text>
                  <Badge 
                    color={scenario.successRate >= 90 ? 'green' : scenario.successRate >= 75 ? 'yellow' : 'red'}
                    variant="light"
                  >
                    {scenario.successRate}%
                  </Badge>
                </Group>
                <Group justify="space-between">
                  <Text size="xs" c="dimmed">Attackers Blocked</Text>
                  <Text size="xs" fw={500}>
                    {scenario.attackers > 0 ? `${scenario.attackers}/${scenario.attackers}` : 'N/A'}
                  </Text>
                </Group>
                <Group justify="space-between">
                  <Text size="xs" c="dimmed">User Impact</Text>
                  <ThemeIcon 
                    size="xs" 
                    color={scenario.avgNormalSolveTime < 2000 ? 'green' : 'orange'}
                    variant="light"
                  >
                    {scenario.avgNormalSolveTime < 2000 ? <IconTrendingDown size={14} /> : <IconTrendingUp size={14} />}
                  </ThemeIcon>
                </Group>
              </Stack>
            </Paper>
          </Grid.Col>
        ))}
      </Grid>
    </Card>
  );
};