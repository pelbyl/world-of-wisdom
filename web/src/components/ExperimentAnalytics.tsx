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
  AreaChart,
  BarChart,
  DonutChart,
  LineChart,
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
  IconDownload,
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
        <Group>
          <Button
            leftSection={isLive ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
            onClick={() => setIsLive(!isLive)}
            color={isLive ? 'red' : 'green'}
          >
            {isLive ? 'Stop Live Analysis' : 'Start Live Analysis'}
          </Button>
          <Button variant="subtle" leftSection={<IconDownload size={16} />}>
            Export Results
          </Button>
        </Group>
      </Group>

      <Tabs value={activeScenario} onChange={setActiveScenario}>
        <Tabs.List>
          <Tabs.Tab value="morning-rush" leftSection={<IconUsers size={14} />}>
            Morning Rush
          </Tabs.Tab>
          <Tabs.Tab value="script-kiddie" leftSection={<IconBug size={14} />}>
            Script Kiddie
          </Tabs.Tab>
          <Tabs.Tab value="ddos" leftSection={<IconRocket size={14} />}>
            DDoS Attack
          </Tabs.Tab>
          <Tabs.Tab value="botnet" leftSection={<IconWorld size={14} />}>
            Botnet
          </Tabs.Tab>
          <Tabs.Tab value="mixed" leftSection={<IconBrain size={14} />}>
            Mixed Reality
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value={activeScenario} pt="lg">
          <Grid>
            <Grid.Col span={{ base: 12, lg: 8 }}>
              <ExperimentSummary scenario={activeScenario} isLive={isLive} />
            </Grid.Col>
            <Grid.Col span={{ base: 12, lg: 4 }}>
              <SuccessCriteria scenario={activeScenario} />
            </Grid.Col>
            <Grid.Col span={{ base: 12 }}>
              <ScenarioTimeline scenario={activeScenario} />
            </Grid.Col>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <ClientDistributionAnalysis />
            </Grid.Col>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <PerformanceMetrics />
            </Grid.Col>
            <Grid.Col span={{ base: 12 }}>
              <AttackMitigationAnalysis />
            </Grid.Col>
          </Grid>
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
};

// Experiment Summary Component
const ExperimentSummary: React.FC<{ scenario: string; isLive: boolean }> = ({ scenario, isLive }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/summary?scenario=${scenario}`);
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
  }, [scenario, isLive]);

  if (loading) return <Card><Center>Loading...</Center></Card>;
  if (!data) return <Card><Center>No data available</Center></Card>;

  const Icon = iconMap[data.icon] || IconActivity;

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Group justify="space-between" mb="md">
        <Group>
          <ThemeIcon size="lg" color={data.color} variant="light">
            <Icon size={24} />
          </ThemeIcon>
          <div>
            <Title order={3}>{data.title}</Title>
            <Text size="sm" c="dimmed">{data.description}</Text>
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
          <Text size="sm" fw={500} mb="xs">Experiment Overview</Text>
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

        <Alert icon={<IconActivity size={16} />} color={data.color} variant="light">
          <Text size="sm" fw={500}>Expected Behavior</Text>
          <Text size="xs">{data.expected_behavior}</Text>
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
    <Card shadow="sm" padding="lg" radius="md" h="100%">
      <Title order={3} mb="md">Success Criteria</Title>
      
      <Center mb="md">
        <RingProgress
          size={120}
          thickness={12}
          roundCaps
          sections={[{ value: score, color: score >= 80 ? 'green' : score >= 60 ? 'yellow' : 'red' }]}
          label={
            <Center>
              <ThemeIcon
                color={score >= 80 ? 'green' : score >= 60 ? 'yellow' : 'red'}
                variant="light"
                radius="xl"
                size="xl"
              >
                {score >= 80 ? <IconMoodSmile size={30} /> : <IconMoodSad size={30} />}
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

// Scenario Timeline Component
const ScenarioTimeline: React.FC<{ scenario: string }> = ({ scenario }) => {
  const [timeline, setTimeline] = useState<any>(null);
  const [currentPhase, setCurrentPhase] = useState(0);

  useEffect(() => {
    const fetchTimeline = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/experiment/timeline?scenario=${scenario}`);
        const result = await response.json();
        setTimeline(result);
      } catch (error) {
        console.error('Failed to fetch timeline:', error);
      }
    };

    fetchTimeline();
  }, [scenario]);

  useEffect(() => {
    if (timeline?.events) {
      const interval = setInterval(() => {
        setCurrentPhase((prev) => (prev + 1) % timeline.events.length);
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [timeline]);

  if (!timeline) return <Card><Center>Loading timeline...</Center></Card>;

  return (
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Scenario Timeline</Title>
      <Timeline active={currentPhase} bulletSize={24} lineWidth={2}>
        {timeline.events?.map((event: any, idx: number) => {
          const Icon = iconMap[event.icon] || IconActivity;
          return (
            <Timeline.Item
              key={idx}
              bullet={<Icon size={14} />}
              color={event.color}
              title={event.event}
              active={idx <= currentPhase}
            >
              <Text c="dimmed" size="sm">{event.time}</Text>
            </Timeline.Item>
          );
        })}
      </Timeline>
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
    <Card shadow="sm" padding="lg" radius="md">
      <Title order={3} mb="md">Client Distribution</Title>
      <DonutChart
        data={data}
        h={250}
        withLabels
        withTooltip
        chartLabel={`${data.reduce((sum, d) => sum + d.value, 0)} clients`}
        thickness={30}
      />
    </Card>
  );
};

// Performance Metrics
const PerformanceMetrics: React.FC = () => {
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
      <Title order={3} mb="md">Performance by Difficulty</Title>
      <BarChart
        h={250}
        data={data}
        dataKey="difficulty"
        series={[
          { name: 'avgSolveTime', color: 'indigo.6', label: 'Avg Solve (ms)' },
          { name: 'failureRate', color: 'red.6', label: 'Failure Rate (%)' },
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