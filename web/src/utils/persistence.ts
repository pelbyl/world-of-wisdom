export interface PersistedState {
  stats: {
    totalChallenges: number;
    completedChallenges: number;
    averageSolveTime: number;
    currentDifficulty: number;
    hashRate: number;
    liveConnections?: number;
    totalConnections?: number;
    networkIntensity?: number;
    ddosProtectionActive?: boolean;
    activeMinerCount?: number;
  };
  blocks: any[];
  connections: any[];
  logs: Array<{
    timestamp: number;
    level: 'info' | 'success' | 'warning' | 'error';
    message: string;
    icon?: string;
  }>;
  metrics: any;
  persistedAt: string;
}

const STORAGE_KEY = 'wisdom_app_state';
const MAX_STORAGE_AGE = 24 * 60 * 60 * 1000; // 24 hours

export const saveState = (state: Partial<PersistedState>): void => {
  try {
    const persistedState: PersistedState = {
      stats: state.stats || {
        totalChallenges: 0,
        completedChallenges: 0,
        averageSolveTime: 0,
        currentDifficulty: 2,
        hashRate: 0,
        liveConnections: 0,
        totalConnections: 0,
        networkIntensity: 1,
        ddosProtectionActive: false,
        activeMinerCount: 0,
      },
      blocks: state.blocks || [],
      connections: state.connections || [],
      logs: state.logs || [],
      metrics: state.metrics || null,
      persistedAt: new Date().toISOString(),
    };

    // Limit data size to prevent localStorage quota issues
    const limitedState = {
      ...persistedState,
      blocks: persistedState.blocks.slice(-50), // Keep last 50 blocks
      logs: persistedState.logs.slice(-200), // Keep last 200 logs
      connections: persistedState.connections.slice(-100), // Keep last 100 connections
    };

    localStorage.setItem(STORAGE_KEY, JSON.stringify(limitedState));
  } catch (error) {
    console.warn('Failed to save state to localStorage:', error);
  }
};

export const loadState = (): Partial<PersistedState> | null => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return null;

    const parsed: PersistedState = JSON.parse(stored);
    
    // Check if data is too old
    const persistedTime = new Date(parsed.persistedAt).getTime();
    const now = Date.now();
    
    if (now - persistedTime > MAX_STORAGE_AGE) {
      localStorage.removeItem(STORAGE_KEY);
      return null;
    }

    return parsed;
  } catch (error) {
    console.warn('Failed to load state from localStorage:', error);
    localStorage.removeItem(STORAGE_KEY);
    return null;
  }
};

export const clearState = (): void => {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch (error) {
    console.warn('Failed to clear state from localStorage:', error);
  }
};

// Auto-save state periodically
export const startAutoSave = (getState: () => Partial<PersistedState>) => {
  const interval = setInterval(() => {
    saveState(getState());
  }, 30000); // Save every 30 seconds

  return () => clearInterval(interval);
};

// Save state on page unload
export const setupBeforeUnloadSave = (getState: () => Partial<PersistedState>) => {
  const handler = () => {
    saveState(getState());
  };

  window.addEventListener('beforeunload', handler);
  return () => window.removeEventListener('beforeunload', handler);
};