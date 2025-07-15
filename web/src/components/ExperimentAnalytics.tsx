import React, { useState, useEffect } from 'react';
import {
  Card,
  Title,
  Text,
  Grid,
  Progress,
  Badge,
  Group,
  Stack,
  RingProgress,
  ThemeIcon,
  Paper,
  Alert,
  Timeline,
  Button,
  Tabs,
  Center,
  Divider,
  Indicator,
} from '@mantine/core';
import {
  BarChart,
  DonutChart,
} from '@mantine/charts';
import {
  IconCheck,
  IconX,
  IconAlertTriangle,
  IconShield,
  IconUsers,
  IconActivity,
  IconTrendingUp,
  IconRocket,
  IconBug,
  IconWorld,
  IconBrain,
  IconMoodSmile,
  IconMoodSad,
  IconPlayerPlay,
  IconPlayerPause,
  IconTrendingDown,
} from '@tabler/icons-react';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081';

// Icon mapping
const iconMap: Record<string, any> = {
  'users': IconUsers,
  'bug': IconBug,
  'rocket': IconRocket,
  'world': IconWorld,
  'brain': IconBrain,
  'activity': IconActivity,
  'trending-up': IconTrendingUp,
  'trending-down': IconTrendingDown,
  'shield': IconShield,
  'alert-triangle': IconAlertTriangle,
  'x': IconX,
};

// Main Experiment Analytics Dashboard
export const ExperimentAnalytics: React.FC = () => {
  const [activeScenario, setActiveScenario] = useState<string>('morning-rush');
  const [isLive, setIsLive] = useState(false);

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Experiment Analytics</Title>
        <Button
          leftSection={isLive ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
          onClick={() => setIsLive(!isLive)}
          color={isLive ? 'red' : 'green'}
        >
          {isLive ? 'Stop Live Analysis' : 'Start Live Analysis'}
        </Button>
      </Group>

      <Grid>
        <Grid.Col span={{ base: 12, lg: 6 }}>
          <ExperimentSummary isLive={isLive} />
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: 6 }}>
          <SuccessCriteria scenario={activeScenario} />
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: 8 }}>
          <TabbedScenarioTimeline activeScenario={activeScenario} setActiveScenario={setActiveScenario} />
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: 4 }}>
          <ClientDistributionAnalysis />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <FailureRateMetrics />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <SolveTimeMetrics />
        </Grid.Col>
        <Grid.Col span={{ base: 12 }}>
          <AttackMitigationAnalysis />
        </Grid.Col>
      </Grid>
    </Stack>
  );
};

// Experiment Summary Component
const ExperimentSummary: React.FC<{ isLive: boolean }> = ({ isLive }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/summary`);
        const result = await response.json();
        setData(result);
        setLoading(false);
      } catch (error) {
        console.error('Failed to fetch experiment summary:', error);
        setLoading(false);
      }
    };

    fetchData();
    const interval = isLive ? setInterval(fetchData, 2000) : null;
    return () => { if (interval) clearInterval(interval); };
  }, [isLive]);

  if (loading) return <Card h="100%"><Center>Loading...</Center></Card>;
  if (!data) return <Card h="100%"><Center>No data available</Center></Card>;

  const Icon = iconMap[data.icon] || IconActivity;

  return (
    <Card shadow="sm" padding="lg" radius="md" h="100%">
      <Group justify="space-between" mb="md">
        <Group>
          <ThemeIcon size="lg" color={data.color} variant="light">
            <Icon size={24} />
          </ThemeIcon>
          <div>
            <Title order={3}>Experiment Overview</Title>
            <Text size="sm" c="dimmed">General system performance metrics</Text>
          </div>
        </Group>
        {isLive && (
          <Indicator processing color="green">
            <Badge color="green" variant="light">LIVE</Badge>
          </Indicator>
        )}
      </Group>

      <Stack gap="md">
        <Paper p="md" withBorder>
          <Text size="sm" fw={500} mb="xs">System Overview</Text>
          <Grid gutter="xs">
            <Grid.Col span={6}>
              <Text size="xs" c="dimmed">Total Clients</Text>
              <Text size="lg" fw={700}>{data.total_clients}</Text>
            </Grid.Col>
            <Grid.Col span={6}>
              <Text size="xs" c="dimmed">Average Difficulty</Text>
              <Text size="lg" fw={700}>{data.avg_difficulty?.toFixed(2) || '0.00'}</Text>
            </Grid.Col>
          </Grid>
        </Paper>

        <Paper p="md" withBorder>
          <Text size="sm" fw={500} mb="xs">Client Distribution</Text>
          <Stack gap="xs">
            <Group justify="space-between">
              <Text size="sm">Normal (1-2)</Text>
              <Group gap="xs">
                <Progress 
                  value={(data.client_distribution?.normal || 0) / data.total_clients * 100} 
                  w={100} 
                  size="sm" 
                  color="green" 
                />
                <Badge color="green" variant="light">{data.client_distribution?.normal || 0}</Badge>
              </Group>
            </Group>
            <Group justify="space-between">
              <Text size="sm">Power Users (3)</Text>
              <Group gap="xs">
                <Progress 
                  value={(data.client_distribution?.power_user || 0) / data.total_clients * 100} 
                  w={100} 
                  size="sm" 
                  color="yellow" 
                />
                <Badge color="yellow" variant="light">{data.client_distribution?.power_user || 0}</Badge>
              </Group>
            </Group>
            <Group justify="space-between">
              <Text size="sm">Suspicious (4)</Text>
              <Group gap="xs">
                <Progress 
                  value={(data.client_distribution?.suspicious || 0) / data.total_clients * 100} 
                  w={100} 
                  size="sm" 
                  color="orange" 
                />
                <Badge color="orange" variant="light">{data.client_distribution?.suspicious || 0}</Badge>
              </Group>
            </Group>
            <Group justify="space-between">
              <Text size="sm">Attackers (5-6)</Text>
              <Group gap="xs">
                <Progress 
                  value={(data.client_distribution?.attacker || 0) / data.total_clients * 100} 
                  w={100} 
                  size="sm" 
                  color="red" 
                />
                <Badge color="red" variant="light">{data.client_distribution?.attacker || 0}</Badge>
              </Group>
            </Group>
          </Stack>
        </Paper>

        <Alert icon={<IconActivity size={16} />} color="blue" variant="light">
          <Text size="sm" fw={500}>Current System Status</Text>
          <Text size="xs">Monitoring all active experiments and client behaviors</Text>
        </Alert>
      </Stack>
    </Card>
  );
};

// Success Criteria Component
const SuccessCriteria: React.FC<{ scenario: string }> = ({ scenario }) => {
  const [criteria, setCriteria] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchCriteria = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/success-criteria?scenario=${scenario}`);
        const result = await response.json();
        setCriteria(result);
        setLoading(false);
      } catch (error) {
        console.error('Failed to fetch success criteria:', error);
        setLoading(false);
      }
    };

    fetchCriteria();
    const interval = setInterval(fetchCriteria, 5000);
    return () => clearInterval(interval);
  }, [scenario]);

  if (loading) return <Card><Center>Loading...</Center></Card>;
  if (!criteria) return <Card><Center>No criteria available</Center></Card>;

  const score = criteria.score || 0;

  return (
    <Card shadow="sm" padding="lg" radius="md" h="100%" style={{ display: 'flex', flexDirection: 'column' }}>
      <Title order={3} mb="md">Success Criteria</Title>
      
      <Center mb="md">
        <RingProgress
          size={140}
          thickness={16}
          roundCaps
          sections={[{ value: score, color: score >= 80 ? 'green' : score >= 60 ? 'yellow' : 'red' }]}
          label={
            <Center>
              <ThemeIcon
                color={score >= 80 ? 'green' : score >= 60 ? 'yellow' : 'red'}
                variant="light"
                radius="xl"
                size={48}
              >
                {score >= 80 ? <IconMoodSmile size={24} /> : <IconMoodSad size={24} />}
              </ThemeIcon>
            </Center>
          }
        />
      </Center>

      <Text ta="center" size="lg" fw={700} mb="md">
        {score.toFixed(0)}% Success Rate
      </Text>

      <Stack gap="md">
        {criteria.categories?.map((category: any, idx: number) => (
          <div key={idx}>
            <Text size="sm" fw={600} mb="xs">{category.name}</Text>
            <Stack gap="xs">
              {category.items?.map((item: any, itemIdx: number) => (
                <Group key={itemIdx} justify="space-between" wrap="nowrap">
                  <Group gap="xs" style={{ flex: 1 }}>
                    <ThemeIcon
                      size="xs"
                      color={item.pass ? 'green' : 'red'}
                      variant="light"
                    >
                      {item.pass ? <IconCheck size={14} /> : <IconX size={14} />}
                    </ThemeIcon>
                    <Text size="xs" style={{ flex: 1 }}>{item.label}</Text>
                  </Group>
                  <Badge size="xs" variant="light" color={item.pass ? 'green' : 'red'}>
                    {item.value}
                  </Badge>
                </Group>
              ))}
            </Stack>
            {idx < criteria.categories.length - 1 && <Divider my="sm" />}
          </div>
        ))}
      </Stack>
    </Card>
  );
};

// Tabbed Scenario Timeline Component
const TabbedScenarioTimeline: React.FC<{ activeScenario: string; setActiveScenario: (scenario: string) => void }> = ({ activeScenario, setActiveScenario }) => {
  const [timelines, setTimelines] = useState<Record<string, any>>({});
  const [currentPhases, setCurrentPhases] = useState<Record<string, number>>({});

  const scenarios = [
    { value: 'morning-rush', label: 'Morning Rush', icon: IconUsers },
    { value: 'script-kiddie', label: 'Script Kiddie', icon: IconBug },
    { value: 'ddos', label: 'DDoS Attack', icon: IconRocket },
    { value: 'botnet', label: 'Botnet', icon: IconWorld },
    { value: 'mixed', label: 'Mixed Reality', icon: IconBrain },
  ];

  useEffect(() => {
    const fetchTimeline = async (scenario: string) => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/timeline?scenario=${scenario}`);
        const result = await response.json();
        setTimelines(prev => ({ ...prev, [scenario]: result }));
        setCurrentPhases(prev => ({ ...prev, [scenario]: 0 }));
      } catch (error) {
        console.error(`Failed to fetch timeline for ${scenario}:`, error);
      }
    };

    // Fetch timeline for active scenario if not already loaded
    if (!timelines[activeScenario]) {
      fetchTimeline(activeScenario);
    }
  }, [activeScenario]);

  useEffect(() => {
    const interval = setInterval(() => {
      setCurrentPhases(prev => {
        const updated = { ...prev };
        Object.keys(timelines).forEach(scenario => {
          if (timelines[scenario]?.events) {
            updated[scenario] = (prev[scenario] + 1) % timelines[scenario].events.length;
          }
        });
        return updated;
      });
    }, 5000);
    return () => clearInterval(interval);
  }, [timelines]);

  const timeline = timelines[activeScenario];
  const currentPhase = currentPhases[activeScenario] || 0;

  return (
    <Card shadow="sm" padding="lg" radius="md" h="100%">
      <Title order={3} mb="md">Scenario Timeline</Title>
      
      <Tabs value={activeScenario} onChange={(value) => value && setActiveScenario(value)}>
        <Tabs.List>
          {scenarios.map((scenario) => {
            const Icon = scenario.icon;
            return (
              <Tabs.Tab key={scenario.value} value={scenario.value} leftSection={<Icon size={14} />}>
                {scenario.label}
              </Tabs.Tab>
            );
          })}
        </Tabs.List>

        <Tabs.Panel value={activeScenario} pt="md">
          {!timeline ? (
            <Center>Loading timeline...</Center>
          ) : (
            <Timeline active={currentPhase} bulletSize={24} lineWidth={2}>
              {timeline.events?.map((event: any, idx: number) => {
                const Icon = iconMap[event.icon] || IconActivity;
                return (
                  <Timeline.Item
                    key={idx}
                    bullet={<Icon size={14} />}
                    color={event.color}
                    title={event.event}
                  >
                    <Text c="dimmed" size="sm">{event.time}</Text>
                  </Timeline.Item>
                );
              })}
            </Timeline>
          )}
        </Tabs.Panel>
      </Tabs>
    </Card>
  );
};

// Client Distribution Analysis
const ClientDistributionAnalysis: React.FC = () => {
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/client-behaviors`);
        const result = await response.json();
        
        // Process data for distribution
        const behaviors = result.data?.clients || [];
        const distribution = [
          { difficulty: 'Normal (1-2)', value: behaviors.filter((c: any) => c.difficulty <= 2).length, color: 'green.6' },
          { difficulty: 'Power (3)', value: behaviors.filter((c: any) => c.difficulty === 3).length, color: 'yellow.6' },
          { difficulty: 'Suspicious (4)', value: behaviors.filter((c: any) => c.difficulty === 4).length, color: 'orange.6' },
          { difficulty: 'Attacker (5-6)', value: behaviors.filter((c: any) => c.difficulty >= 5).length, color: 'red.6' },
        ];
        
        setData(distribution);
      } catch (error) {
        console.error('Failed to fetch distribution:', error);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 3000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Card shadow="sm" padding="lg" radius="md" h="100%">
      <Title order={3} mb="md">Client Distribution</Title>
      <DonutChart
        data={data}
        h={350}
        withLabels
        withTooltip
        chartLabel={`${data.reduce((sum, d) => sum + d.value, 0)} clients`}
        thickness={30}
      />
    </Card>
  );
};

// Failure Rate Metrics
const FailureRateMetrics: React.FC = () => {
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/performance`);
        const result = await response.json();
        setData(result.data || []);
      } catch (error) {
        console.error('Failed to fetch performance data:', error);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Failure Rate by Difficulty</Title>
      <BarChart
        h={250}
        data={data}
        dataKey="difficulty"
        series={[
          { name: 'failureRate', color: 'red.6', label: 'Failure Rate (%)' },
        ]}
        tickLine="y"
        withLegend
        legendProps={{ verticalAlign: 'bottom' }}
      />
    </Card>
  );
};

// Solve Time Metrics
const SolveTimeMetrics: React.FC = () => {
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/performance`);
        const result = await response.json();
        setData(result.data || []);
      } catch (error) {
        console.error('Failed to fetch performance data:', error);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Average Solve Time by Difficulty</Title>
      <BarChart
        h={250}
        data={data}
        dataKey="difficulty"
        series={[
          { name: 'avgSolveTime', color: 'indigo.6', label: 'Avg Solve Time (ms)' },
        ]}
        tickLine="y"
        withLegend
        legendProps={{ verticalAlign: 'bottom' }}
      />
    </Card>
  );
};

// Attack Mitigation Analysis
const AttackMitigationAnalysis: React.FC = () => {
  const [stats, setStats] = useState<any>(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/mitigation`);
        const result = await response.json();
        setStats(result);
      } catch (error) {
        console.error('Failed to fetch mitigation stats:', error);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 3000);
    return () => clearInterval(interval);
  }, []);

  if (!stats) return <Card><Center>Loading...</Center></Card>;

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Attack Mitigation Analysis</Title>
      
      <Grid gutter="md">
        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" wrap="nowrap">
              <div>
                <Text size="xs" c="dimmed">Detection Rate</Text>
                <Text size="xl" fw={700}>{stats.detection_rate?.toFixed(1)}%</Text>
              </div>
              <ThemeIcon size="lg" color="blue" variant="light">
                <IconActivity size={24} />
              </ThemeIcon>
            </Group>
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" wrap="nowrap">
              <div>
                <Text size="xs" c="dimmed">Time to Detect</Text>
                <Text size="xl" fw={700}>{stats.avg_time_to_detect}</Text>
              </div>
              <ThemeIcon size="lg" color="green" variant="light">
                <IconRocket size={24} />
              </ThemeIcon>
            </Group>
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" wrap="nowrap">
              <div>
                <Text size="xs" c="dimmed">False Positives</Text>
                <Text size="xl" fw={700} c={stats.false_positive_rate > 5 ? 'red' : 'green'}>
                  {stats.false_positive_rate?.toFixed(1)}%
                </Text>
              </div>
              <ThemeIcon size="lg" color={stats.false_positive_rate > 5 ? 'red' : 'green'} variant="light">
                <IconAlertTriangle size={24} />
              </ThemeIcon>
            </Group>
          </Paper>
        </Grid.Col>

        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" wrap="nowrap">
              <div>
                <Text size="xs" c="dimmed">Effectiveness</Text>
                <Text size="xl" fw={700}>{stats.effectiveness_score}%</Text>
              </div>
              <ThemeIcon size="lg" color="green" variant="light">
                <IconShield size={24} />
              </ThemeIcon>
            </Group>
          </Paper>
        </Grid.Col>
      </Grid>

      <Stack mt="md" gap="xs">
        <Alert 
          icon={<IconUsers size={16} />} 
          color={stats.normal_user_impact < 3000 ? 'green' : 'yellow'} 
          variant="light"
        >
          Normal user impact: {(stats.normal_user_impact / 1000).toFixed(1)}s average solve time
        </Alert>
        <Alert 
          icon={<IconShield size={16} />} 
          color="blue" 
          variant="light"
        >
          {stats.attackers_penalized} attackers successfully penalized
        </Alert>
      </Stack>
    </Card>
  );
};