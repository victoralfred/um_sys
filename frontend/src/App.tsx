import { Router, Route } from '@solidjs/router';
import { createSignal, Show, For } from 'solid-js';
import { createStore } from 'solid-js/store';

// Enhanced user data with roles and activity tracking
const [users, setUsers] = createStore([
  { 
    id: 1, 
    name: 'John Doe', 
    email: 'john@example.com', 
    status: 'Active', 
    role: 'Admin',
    department: 'IT',
    lastLogin: '2024-01-15T10:30:00Z',
    createdAt: '2023-06-15T08:00:00Z',
    loginCount: 245,
    activityScore: 95,
    permissions: ['read', 'write', 'delete', 'admin'],
    lastActivity: '2024-01-15T11:45:00Z'
  },
  { 
    id: 2, 
    name: 'Jane Smith', 
    email: 'jane@example.com', 
    status: 'Active', 
    role: 'Manager',
    department: 'Sales',
    lastLogin: '2024-01-14T16:45:00Z',
    createdAt: '2023-08-20T09:15:00Z',
    loginCount: 189,
    activityScore: 87,
    permissions: ['read', 'write', 'manage'],
    lastActivity: '2024-01-14T17:30:00Z'
  },
  { 
    id: 3, 
    name: 'Bob Johnson', 
    email: 'bob@example.com', 
    status: 'Inactive', 
    role: 'User',
    department: 'Marketing',
    lastLogin: '2024-01-10T14:20:00Z',
    createdAt: '2023-09-10T11:30:00Z',
    loginCount: 67,
    activityScore: 23,
    permissions: ['read'],
    lastActivity: '2024-01-10T14:45:00Z'
  },
  { 
    id: 4, 
    name: 'Alice Cooper', 
    email: 'alice@example.com', 
    status: 'Active', 
    role: 'Manager',
    department: 'HR',
    lastLogin: '2024-01-15T09:15:00Z',
    createdAt: '2023-07-25T14:45:00Z',
    loginCount: 201,
    activityScore: 91,
    permissions: ['read', 'write', 'manage'],
    lastActivity: '2024-01-15T10:00:00Z'
  },
  { 
    id: 5, 
    name: 'Charlie Brown', 
    email: 'charlie@example.com', 
    status: 'Active', 
    role: 'User',
    department: 'Finance',
    lastLogin: '2024-01-13T13:30:00Z',
    createdAt: '2023-10-05T16:20:00Z',
    loginCount: 134,
    activityScore: 72,
    permissions: ['read', 'write'],
    lastActivity: '2024-01-13T14:15:00Z'
  }
]);

// Activity log store for tracking user actions
const [activityLog, setActivityLog] = createStore([
  { id: 1, userId: 1, action: 'User Created', timestamp: new Date().toISOString(), details: 'Created new user account' },
  { id: 2, userId: 2, action: 'Login', timestamp: new Date(Date.now() - 86400000).toISOString(), details: 'User logged in successfully' },
  { id: 3, userId: 1, action: 'User Updated', timestamp: new Date(Date.now() - 172800000).toISOString(), details: 'Updated user profile information' },
  { id: 4, userId: 3, action: 'Status Changed', timestamp: new Date(Date.now() - 259200000).toISOString(), details: 'User status changed to inactive' },
  { id: 5, userId: 4, action: 'Role Changed', timestamp: new Date(Date.now() - 345600000).toISOString(), details: 'User role updated to Manager' }
]);

// Modal states
const [showCreateModal, setShowCreateModal] = createSignal(false);
const [showEditModal, setShowEditModal] = createSignal(false);
const [editingUser, setEditingUser] = createSignal(null);
const [showProfileModal, setShowProfileModal] = createSignal(false);
const [selectedProfile, setSelectedProfile] = createSignal(null);
const [showAdminPanel, setShowAdminPanel] = createSignal(false);

// Notification system
const [notifications, setNotifications] = createStore([]);
const [notificationId, setNotificationId] = createSignal(1);

const addNotification = (message: string, type = 'info', duration = 5000) => {
  const id = notificationId();
  setNotificationId(id + 1);
  
  const notification = {
    id,
    message,
    type, // 'success', 'error', 'warning', 'info'
    timestamp: Date.now(),
    visible: true
  };
  
  setNotifications([...notifications, notification]);
  
  // Auto-remove notification
  if (typeof window !== 'undefined') {
    window.setTimeout(() => {
      removeNotification(id);
    }, duration);
  }
  
  return id;
};

const removeNotification = (id: number) => {
  setNotifications(notifications.filter(n => n.id !== id));
};

// Search and filtering state
const [searchQuery, setSearchQuery] = createSignal('');
const [statusFilter, setStatusFilter] = createSignal('all');
const [roleFilter, setRoleFilter] = createSignal('all');
const [departmentFilter, setDepartmentFilter] = createSignal('all');
const [sortBy, setSortBy] = createSignal('name');
const [sortOrder, setSortOrder] = createSignal('asc');

// Bulk selection state
const [selectedUsers, setSelectedUsers] = createSignal([]);

// Dark mode state - initialize from localStorage
const [darkMode, setDarkMode] = createSignal(() => {
  try {
    return window.localStorage.getItem('umanager-dark-mode') === 'true';
  } catch {
    return false;
  }
});

// Real-time simulation state
const [realtimeActive, setRealtimeActive] = createSignal(false);
const [simulationInterval, setSimulationInterval] = createSignal(null);

// Keyboard shortcuts state
const [showShortcuts, setShowShortcuts] = createSignal(false);

// Performance monitoring
const [performanceMetrics, setPerformanceMetrics] = createStore({
  renderTime: 0,
  memoryUsage: 0,
  componentCount: 0,
  lastUpdate: Date.now(),
  history: [],
  alerts: []
});
const [showPerformanceMonitor, setShowPerformanceMonitor] = createSignal(false);

// Feature Flags
const [featureFlags, setFeatureFlags] = createStore({
  advancedAnalytics: true,
  realTimeNotifications: true,
  bulkOperations: true,
  advancedFiltering: true,
  exportFeatures: true,
  betaFeatures: false,
  experimentalUI: false,
  aiInsights: false
});

// Background Jobs Configuration
const [backgroundJobs, setBackgroundJobs] = createStore({
  dataSync: {
    enabled: true,
    interval: 300, // 5 minutes
    lastRun: Date.now() - 180000, // 3 minutes ago
    status: 'running'
  },
  reportGeneration: {
    enabled: true,
    interval: 3600, // 1 hour
    lastRun: Date.now() - 1200000, // 20 minutes ago
    status: 'idle'
  },
  userActivitySync: {
    enabled: true,
    interval: 60, // 1 minute
    lastRun: Date.now() - 30000, // 30 seconds ago
    status: 'running'
  },
  backupTask: {
    enabled: false,
    interval: 86400, // 24 hours
    lastRun: Date.now() - 3600000, // 1 hour ago
    status: 'disabled'
  }
});

// Theme utilities
const getTheme = () => {
  const isDark = darkMode();
  return {
    // Background colors
    bg: {
      primary: isDark ? '#1A1B23' : '#F7F8F9',
      secondary: isDark ? '#2A2B35' : '#FFFFFF',
      tertiary: isDark ? '#3A3B45' : '#F4F5F7',
      accent: isDark ? '#4A4B55' : '#E6F3FF'
    },
    // Text colors
    text: {
      primary: isDark ? '#E6E6E6' : '#172B4D',
      secondary: isDark ? '#B3B3B3' : '#6B778C',
      tertiary: isDark ? '#808080' : '#8993A4',
      accent: isDark ? '#66B2FF' : '#0052CC'
    },
    // Border colors
    border: {
      primary: isDark ? '#404040' : '#DFE1E6',
      secondary: isDark ? '#505050' : '#F1F2F4',
      accent: isDark ? '#66B2FF' : '#0052CC'
    },
    // Status colors (remain consistent)
    status: {
      success: '#006644',
      error: '#DE350B',
      warning: '#FF8B00',
      info: '#0052CC'
    }
  };
};

// Activity tracking functions
const logActivity = (userId: number, action: string, details: string) => {
  const newActivity = {
    id: Math.max(...activityLog.map(a => a.id), 0) + 1,
    userId,
    action,
    timestamp: new Date().toISOString(),
    details
  };
  setActivityLog([newActivity, ...activityLog]);
};

const getUserName = (userId: number) => {
  const user = users.find(u => u.id === userId);
  return user ? user.name : 'Unknown User';
};

const formatTimeAgo = (timestamp: string) => {
  const now = new Date();
  const time = new Date(timestamp);
  const diffInHours = Math.floor((now - time) / (1000 * 60 * 60));
  
  if (diffInHours < 1) return 'Just now';
  if (diffInHours === 1) return '1 hour ago';
  if (diffInHours < 24) return `${diffInHours} hours ago`;
  
  const diffInDays = Math.floor(diffInHours / 24);
  if (diffInDays === 1) return '1 day ago';
  if (diffInDays < 7) return `${diffInDays} days ago`;
  
  return time.toLocaleDateString();
};

// Real-time simulation functions
const simulateUserActivity = () => {
  const actions = ['Login', 'Profile Updated', 'Password Changed', 'Settings Modified', 'Data Export'];
  const user = users[Math.floor(Math.random() * users.length)];
  const action = actions[Math.floor(Math.random() * actions.length)];
  const details = `Simulated ${action.toLowerCase()} for ${user.name}`;
  
  logActivity(user.id, action, details);
  
  // Update user's activity score and login count occasionally
  if (Math.random() > 0.7) {
    setUsers(users.map(u => 
      u.id === user.id 
        ? { 
            ...u, 
            activityScore: Math.min(100, u.activityScore + Math.floor(Math.random() * 5)),
            loginCount: u.loginCount + (action === 'Login' ? 1 : 0),
            lastActivity: new Date().toISOString()
          }
        : u
    ));
  }
  
  // Show notification occasionally
  if (Math.random() > 0.8) {
    addNotification(`🔄 ${user.name} ${action.toLowerCase()}`, 'info', 3000);
  }
};

const startRealtimeSimulation = () => {
  if (realtimeActive()) return;
  
  setRealtimeActive(true);
  const interval = window.setInterval(() => {
    simulateUserActivity();
  }, 3000 + Math.random() * 4000); // Random interval between 3-7 seconds
  
  setSimulationInterval(interval);
  addNotification('🚀 Real-time simulation started - watch the activity feed!', 'success', 5000);
};

const stopRealtimeSimulation = () => {
  if (!realtimeActive()) return;
  
  const interval = simulationInterval();
  if (interval) {
    if (typeof window !== 'undefined') {
      window.clearInterval(interval);
    }
    setSimulationInterval(null);
  }
  
  setRealtimeActive(false);
  addNotification('⏸️ Real-time simulation stopped', 'info', 3000);
};

// Keyboard shortcuts system
const handleKeyboardShortcuts = (event: KeyboardEvent) => {
  
  const { key, ctrlKey, metaKey, altKey, shiftKey } = event;
  const isCmd = ctrlKey || metaKey;
  
  // Prevent default browser shortcuts when needed
  const shortcuts = {
    // Global shortcuts
    '?': () => {
      setShowShortcuts(true);
      addNotification('Keyboard shortcuts panel opened', 'info');
    },
    'Escape': () => {
      setShowShortcuts(false);
      setShowAdminPanel(false);
      setShowProfileModal(false);
      setShowCreateModal(false);
      setShowEditModal(false);
    },
    
    // Theme and UI shortcuts
    'd': () => {
      if (isCmd) {
        event.preventDefault();
        const newMode = !darkMode();
        updateDarkMode(newMode);
        addNotification(`Switched to ${newMode ? 'dark' : 'light'} mode via keyboard`, 'success');
      }
    },
    
    // Navigation shortcuts
    '1': () => {
      if (altKey) {
        event.preventDefault();
        window.location.href = '/dashboard';
        addNotification('Navigated to Dashboard', 'info');
      }
    },
    '2': () => {
      if (altKey) {
        event.preventDefault();
        window.location.href = '/users';
        addNotification('Navigated to Users', 'info');
      }
    },
    '3': () => {
      if (altKey) {
        event.preventDefault();
        window.location.href = '/login';
        addNotification('Navigated to Login', 'info');
      }
    },
    
    // Quick actions
    'n': () => {
      if (isCmd) {
        event.preventDefault();
        setShowCreateModal(true);
        addNotification('Create user modal opened via keyboard', 'info');
      }
    },
    'a': () => {
      if (isCmd && shiftKey) {
        event.preventDefault();
        setShowAdminPanel(true);
        addNotification('Admin panel opened via keyboard', 'info');
      }
    },
    'p': () => {
      if (isCmd && shiftKey) {
        event.preventDefault();
        setShowPerformanceMonitor(true);
        addNotification('Performance monitor opened', 'info');
      }
    },
    'r': () => {
      if (isCmd && shiftKey) {
        event.preventDefault();
        realtimeActive() ? stopRealtimeSimulation() : startRealtimeSimulation();
      }
    }
  };
  
  if (shortcuts[key]) {
    shortcuts[key]();
  }
};

// Initialize keyboard shortcuts
if (typeof window !== 'undefined') {
  window.addEventListener('keydown', handleKeyboardShortcuts);
}

// LocalStorage persistence functions
const saveToLocalStorage = (key: string, data: unknown) => {
  try {
    window.localStorage.setItem(`umanager-${key}`, JSON.stringify(data));
  } catch (error) {
    console.warn('Failed to save to localStorage:', error);
  }
};

const loadFromLocalStorage = (key: string, defaultValue: unknown = null) => {
  try {
    const item = window.localStorage.getItem(`umanager-${key}`);
    return item ? JSON.parse(item) : defaultValue;
  } catch (error) {
    console.warn('Failed to load from localStorage:', error);
    return defaultValue;
  }
};

// Save preferences whenever they change
const saveDarkMode = (mode: boolean) => {
  try {
    window.localStorage.setItem('umanager-dark-mode', mode.toString());
  } catch (error) {
    console.warn('Failed to save dark mode preference:', error);
  }
};

// Use the function to save dark mode changes
const updateDarkMode = (newMode: boolean) => {
  setDarkMode(newMode);
  saveDarkMode(newMode);
};

// Save user data periodically
const saveUserData = () => {
  saveToLocalStorage('users', users);
  saveToLocalStorage('activity-log', activityLog);
  addNotification('Data automatically saved', 'success', 2000);
};

// Load user data on initialization
const loadUserData = () => {
  try {
    const savedUsers = loadFromLocalStorage('users');
    const savedActivity = loadFromLocalStorage('activity-log');
    
    if (savedUsers && savedUsers.length > 0) {
      setUsers(savedUsers);
      addNotification('User data restored from previous session', 'info', 4000);
    }
    
    if (savedActivity && savedActivity.length > 0) {
      setActivityLog(savedActivity);
    }
  } catch (error) {
    console.warn('Failed to load user data:', error);
    addNotification('Using default data - previous session could not be restored', 'warning', 4000);
  }
};

// Auto-save every 30 seconds
if (typeof window !== 'undefined') {
  window.setInterval(() => {
    if (users.length > 0) {
      saveUserData();
    }
  }, 30000);
}

// Load data on app start
if (typeof window !== 'undefined') {
  window.setTimeout(() => {
    loadUserData();
  }, 1000);
}

// Performance Monitoring Functions
const updatePerformanceMetrics = () => {
  const now = Date.now();
  
  // Simulate memory usage (in MB)
  const memoryUsage = Math.floor(15 + Math.random() * 25);
  
  // Count DOM elements as component count proxy
  const componentCount = typeof document !== 'undefined' ? document.querySelectorAll('*').length : 100;
  
  // Simulate render time (in ms)
  const renderTime = Math.floor(8 + Math.random() * 12);
  
  const newMetrics = {
    renderTime,
    memoryUsage,
    componentCount,
    lastUpdate: now,
    history: [...performanceMetrics.history, {
      timestamp: now,
      renderTime,
      memoryUsage,
      componentCount
    }].slice(-20), // Keep last 20 entries
    alerts: [
      ...performanceMetrics.alerts,
      ...(memoryUsage > 35 ? [{
        id: Date.now(),
        type: 'warning',
        message: `High memory usage detected: ${memoryUsage}MB`,
        timestamp: now
      }] : []),
      ...(renderTime > 15 ? [{
        id: Date.now() + 1,
        type: 'warning', 
        message: `Slow render time detected: ${renderTime}ms`,
        timestamp: now
      }] : [])
    ].slice(-10) // Keep last 10 alerts
  };
  
  setPerformanceMetrics(newMetrics);
};

// Start performance monitoring
if (typeof window !== 'undefined') {
  window.setInterval(() => {
    updatePerformanceMetrics();
  }, 2000);
}

// Initial performance update
if (typeof window !== 'undefined') {
  window.setTimeout(() => {
    updatePerformanceMetrics();
  }, 500);
}

// Feature Flag Management Functions
const toggleFeatureFlag = (flagName: string) => {
  setFeatureFlags(flagName, !featureFlags[flagName]);
  saveToLocalStorage('feature-flags', featureFlags);
  addNotification(`Feature "${flagName}" ${featureFlags[flagName] ? 'enabled' : 'disabled'}`, 'info');
};

// Background Job Management Functions
const toggleBackgroundJob = (jobName: string) => {
  setBackgroundJobs(jobName, 'enabled', !backgroundJobs[jobName].enabled);
  setBackgroundJobs(jobName, 'status', backgroundJobs[jobName].enabled ? 'disabled' : 'idle');
  saveToLocalStorage('background-jobs', backgroundJobs);
  addNotification(`Background job "${jobName}" ${backgroundJobs[jobName].enabled ? 'enabled' : 'disabled'}`, backgroundJobs[jobName].enabled ? 'success' : 'warning');
};

const updateJobInterval = (jobName: string, newInterval: number) => {
  setBackgroundJobs(jobName, 'interval', newInterval);
  saveToLocalStorage('background-jobs', backgroundJobs);
  addNotification(`Updated ${jobName} interval to ${newInterval} seconds`, 'info');
};

const getJobStatusColor = (status: string) => {
  switch (status) {
    case 'running': return '#10B981';
    case 'idle': return '#F59E0B';
    case 'error': return '#EF4444';
    case 'disabled': return '#6B7280';
    default: return '#6B7280';
  }
};

const formatLastRun = (timestamp: number) => {
  const now = Date.now();
  const diff = now - timestamp;
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  
  if (minutes < 1) return 'Just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return new Date(timestamp).toLocaleDateString();
};

// Load feature flags and background jobs from localStorage
if (typeof window !== 'undefined') {
  window.setTimeout(() => {
  const savedFlags = loadFromLocalStorage('feature-flags');
  const savedJobs = loadFromLocalStorage('background-jobs');
  
  if (savedFlags) {
    Object.keys(savedFlags).forEach(key => {
      if (key in featureFlags) {
        setFeatureFlags(key, savedFlags[key]);
      }
    });
  }
  
  if (savedJobs) {
    Object.keys(savedJobs).forEach(key => {
      if (key in backgroundJobs) {
        setBackgroundJobs(key, savedJobs[key]);
      }
    });
  }
  }, 100);
}

// Keyboard Shortcuts Help Panel
const KeyboardShortcutsPanel = () => {
  return (
    <Show when={showShortcuts()}>
      <div 
        style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); display: flex; align-items: center; justify-content: center; z-index: 2000; backdrop-filter: blur(4px);"
        onclick={() => setShowShortcuts(false)}
      >
        <div 
          style={`background: ${getTheme().bg.secondary}; border-radius: 12px; width: 90%; max-width: 600px; max-height: 80vh; overflow-y: auto; box-shadow: 0 20px 60px rgba(0,0,0,0.4); border: 1px solid ${getTheme().border.primary};`}
          onclick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div style="background: linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%); padding: 24px; color: white; border-radius: 12px 12px 0 0;">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <div>
                <h2 style="margin: 0 0 4px 0; font-size: 24px; font-weight: 600;">⌨️ Keyboard Shortcuts</h2>
                <p style="margin: 0; opacity: 0.9; font-size: 14px;">Master UManager with these powerful shortcuts</p>
              </div>
              <button
                onclick={() => setShowShortcuts(false)}
                style="background: rgba(255,255,255,0.2); border: none; color: white; width: 32px; height: 32px; border-radius: 50%; cursor: pointer; font-size: 16px;"
              >
                ×
              </button>
            </div>
          </div>

          {/* Shortcuts Content */}
          <div style="padding: 24px;">
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 24px;">
              {/* Global Shortcuts */}
              <div>
                <h3 style={`color: ${getTheme().text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                  <span style="margin-right: 8px;">🌐</span> Global
                </h3>
                <div style="space-y: 12px;">
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Show shortcuts</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>?</kbd>
                  </div>
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Close modals</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>Esc</kbd>
                  </div>
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Toggle dark mode</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌘ D</kbd>
                  </div>
                </div>
              </div>

              {/* Navigation */}
              <div>
                <h3 style={`color: ${getTheme().text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                  <span style="margin-right: 8px;">🧭</span> Navigation
                </h3>
                <div style="space-y: 12px;">
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Dashboard</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌥ 1</kbd>
                  </div>
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Users page</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌥ 2</kbd>
                  </div>
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Login page</span>
                    <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌥ 3</kbd>
                  </div>
                </div>
              </div>
            </div>

            {/* Quick Actions */}
            <div style="margin-top: 24px;">
              <h3 style={`color: ${getTheme().text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                <span style="margin-right: 8px;">⚡</span> Quick Actions
              </h3>
              <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                  <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Create user</span>
                  <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌘ N</kbd>
                </div>
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                  <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Admin panel</span>
                  <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌘⇧ A</kbd>
                </div>
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                  <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Toggle live demo</span>
                  <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌘⇧ R</kbd>
                </div>
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                  <span style={`color: ${getTheme().text.secondary}; font-size: 14px;`}>Performance monitor</span>
                  <kbd style={`background: ${getTheme().bg.tertiary}; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: ${getTheme().text.primary}; border: 1px solid ${getTheme().border.primary};`}>⌘⇧ P</kbd>
                </div>
              </div>
            </div>

            {/* Footer */}
            <div style={`margin-top: 24px; padding-top: 16px; border-top: 1px solid ${getTheme().border.primary}; text-align: center;`}>
              <p style={`color: ${getTheme().text.tertiary}; font-size: 12px; margin: 0;`}>
                💡 Pro tip: Use keyboard shortcuts to navigate UManager like a power user!
              </p>
            </div>
          </div>
        </div>
      </div>
    </Show>
  );
};

// User Profile Modal
const UserProfileModal = () => {
  return (
    <Show when={showProfileModal() && selectedProfile()}>
      {(() => {
        const user = selectedProfile();
        const userActivities = activityLog.filter(a => a.userId === user.id).slice(0, 10);
        
        return (
          <div 
            style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 1000; backdrop-filter: blur(2px);"
            onclick={() => {
              setShowProfileModal(false);
              setSelectedProfile(null);
            }}
          >
            <div 
              style="background: white; border-radius: 12px; width: 90%; max-width: 600px; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 40px rgba(0,0,0,0.2);"
              onclick={(e) => e.stopPropagation()}
            >
              {/* Header */}
              <div style="background: linear-gradient(135deg, #0052CC 0%, #4A90E2 100%); padding: 30px; color: white; border-radius: 12px 12px 0 0; position: relative;">
                <button
                  onclick={() => {
                    setShowProfileModal(false);
                    setSelectedProfile(null);
                  }}
                  style="position: absolute; top: 20px; right: 20px; background: rgba(255,255,255,0.2); border: none; color: white; width: 36px; height: 36px; border-radius: 50%; cursor: pointer; font-size: 18px; display: flex; align-items: center; justify-content: center;"
                >
                  ×
                </button>
                
                <div style="display: flex; align-items: center; gap: 20px;">
                  <div style="width: 80px; height: 80px; background: rgba(255,255,255,0.2); border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 32px;">
                    {user.name.split(' ').map(n => n[0]).join('')}
                  </div>
                  <div>
                    <h2 style="margin: 0 0 8px 0; font-size: 28px; font-weight: 600;">{user.name}</h2>
                    <div style="opacity: 0.9; font-size: 16px; margin-bottom: 8px;">{user.email}</div>
                    <div style="display: flex; gap: 12px; align-items: center;">
                      <span style={`padding: 6px 12px; border-radius: 16px; font-size: 12px; font-weight: 500; ${
                        user.status === 'Active' 
                          ? 'background: rgba(255,255,255,0.2); color: white;' 
                          : 'background: rgba(255,255,255,0.1); color: rgba(255,255,255,0.8);'
                      }`}>
                        {user.status}
                      </span>
                      <span style="padding: 6px 12px; border-radius: 16px; font-size: 12px; font-weight: 500; background: rgba(255,255,255,0.2); color: white;">
                        {user.role}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Content */}
              <div style="padding: 30px;">
                {/* Stats Row */}
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 16px; margin-bottom: 30px;">
                  <div style="text-align: center; padding: 16px; background: #F7F8F9; border-radius: 8px;">
                    <div style="font-size: 24px; font-weight: 600; color: #0052CC; margin-bottom: 4px;">{user.loginCount}</div>
                    <div style="font-size: 12px; color: #6B778C;">Total Logins</div>
                  </div>
                  <div style="text-align: center; padding: 16px; background: #F7F8F9; border-radius: 8px;">
                    <div style="font-size: 24px; font-weight: 600; color: #006644; margin-bottom: 4px;">{user.activityScore}%</div>
                    <div style="font-size: 12px; color: #6B778C;">Activity Score</div>
                  </div>
                  <div style="text-align: center; padding: 16px; background: #F7F8F9; border-radius: 8px;">
                    <div style="font-size: 24px; font-weight: 600; color: #FF8B00; margin-bottom: 4px;">{user.permissions.length}</div>
                    <div style="font-size: 12px; color: #6B778C;">Permissions</div>
                  </div>
                </div>

                {/* Two Column Layout */}
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px;">
                  {/* User Details */}
                  <div>
                    <h4 style="color: #172B4D; font-size: 16px; margin: 0 0 16px 0; font-weight: 600;">User Details</h4>
                    <div style="space-y: 12px;">
                      <div style="margin-bottom: 12px;">
                        <div style="font-size: 12px; color: #6B778C; margin-bottom: 4px;">Department</div>
                        <div style="font-weight: 500; color: #172B4D;">{user.department}</div>
                      </div>
                      <div style="margin-bottom: 12px;">
                        <div style="font-size: 12px; color: #6B778C; margin-bottom: 4px;">Last Login</div>
                        <div style="font-weight: 500; color: #172B4D;">
                          {user.lastLogin ? formatTimeAgo(user.lastLogin) : 'Never'}
                        </div>
                      </div>
                      <div style="margin-bottom: 12px;">
                        <div style="font-size: 12px; color: #6B778C; margin-bottom: 4px;">Member Since</div>
                        <div style="font-weight: 500; color: #172B4D;">
                          {new Date(user.createdAt).toLocaleDateString()}
                        </div>
                      </div>
                      <div style="margin-bottom: 12px;">
                        <div style="font-size: 12px; color: #6B778C; margin-bottom: 4px;">Permissions</div>
                        <div style="display: flex; flex-wrap: wrap; gap: 4px;">
                          <For each={user.permissions}>
                            {(permission) => (
                              <span style="padding: 4px 8px; background: #E6F3FF; color: #0052CC; border-radius: 12px; font-size: 11px; font-weight: 500;">
                                {permission}
                              </span>
                            )}
                          </For>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Recent Activity */}
                  <div>
                    <h4 style="color: #172B4D; font-size: 16px; margin: 0 0 16px 0; font-weight: 600;">Recent Activity</h4>
                    <div style="max-height: 200px; overflow-y: auto;">
                      <Show when={userActivities.length > 0} fallback={
                        <div style="text-align: center; color: #6B778C; padding: 20px; font-size: 14px;">
                          No recent activity
                        </div>
                      }>
                        <For each={userActivities}>
                          {(activity) => (
                            <div style="display: flex; align-items: flex-start; padding: 12px 0; border-bottom: 1px solid #F4F5F7;">
                              <div style={`flex-shrink: 0; width: 6px; height: 6px; border-radius: 50%; margin-right: 12px; margin-top: 6px; background: ${
                                activity.action === 'User Created' ? '#006644' :
                                activity.action === 'User Updated' ? '#0052CC' :
                                activity.action === 'Login' ? '#FF8B00' : '#6B778C'
                              };`}></div>
                              <div style="flex: 1; min-width: 0;">
                                <div style="font-weight: 500; color: #172B4D; font-size: 13px; margin-bottom: 2px;">
                                  {activity.action}
                                </div>
                                <div style="color: #8993A4; font-size: 11px;">
                                  {formatTimeAgo(activity.timestamp)}
                                </div>
                              </div>
                            </div>
                          )}
                        </For>
                      </Show>
                    </div>
                  </div>
                </div>

                {/* Action Buttons */}
                <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #F4F5F7; display: flex; gap: 12px; justify-content: flex-end;">
                  <button
                    onclick={() => {
                      setSelectedProfile(null);
                      setShowProfileModal(false);
                      setEditingUser(user);
                      setShowEditModal(true);
                      addNotification(`Opening edit form for ${user.name}`, 'info');
                    }}
                    style="padding: 10px 20px; background: #0052CC; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
                  >
                    ✏️ Edit User
                  </button>
                  <button
                    onclick={() => {
                      setShowProfileModal(false);
                      setSelectedProfile(null);
                    }}
                    style="padding: 10px 20px; background: #F4F5F7; color: #172B4D; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>
          </div>
        );
      })()}
    </Show>
  );
};

// Performance Monitor Panel
const PerformanceMonitorPanel = () => {
  const theme = getTheme();
  
  const formatBytes = (bytes) => {
    return `${bytes} MB`;
  };
  
  const formatTime = (ms) => {
    return `${ms}ms`;
  };
  
  const getPerformanceStatus = () => {
    const { memoryUsage, renderTime } = performanceMetrics;
    if (memoryUsage > 35 || renderTime > 15) return 'warning';
    if (memoryUsage > 25 || renderTime > 10) return 'caution';
    return 'good';
  };
  
  const statusColors = {
    good: '#10B981',
    caution: '#F59E0B',
    warning: '#EF4444'
  };
  
  return (
    <Show when={showPerformanceMonitor()}>
      <div 
        style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 1500; backdrop-filter: blur(3px);"
        onclick={() => setShowPerformanceMonitor(false)}
      >
        <div 
          style={`background: ${theme.bg.secondary}; border-radius: 12px; width: 90%; max-width: 900px; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 40px rgba(0,0,0,0.3); border: 1px solid ${theme.border.primary};`}
          onclick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div style="background: linear-gradient(135deg, #059669 0%, #0D9488 100%); padding: 24px; color: white; border-radius: 12px 12px 0 0;">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <div>
                <h2 style="margin: 0 0 4px 0; font-size: 24px; font-weight: 600;">📊 Performance Monitor</h2>
                <p style="margin: 0; opacity: 0.9; font-size: 14px;">Real-time system performance metrics</p>
              </div>
              <button
                onclick={() => setShowPerformanceMonitor(false)}
                style="background: rgba(255,255,255,0.2); border: none; color: white; width: 32px; height: 32px; border-radius: 50%; cursor: pointer; font-size: 16px;"
              >
                ×
              </button>
            </div>
          </div>
          
          {/* Content */}
          <div style="padding: 24px;">
            {/* Current Metrics */}
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px;">
              <div style={`background: ${theme.bg.tertiary}; padding: 16px; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <div style="display: flex; align-items: center; margin-bottom: 8px;">
                  <span style="font-size: 20px; margin-right: 8px;">⚡</span>
                  <span style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px;`}>Render Time</span>
                </div>
                <div style={`color: ${theme.text.primary}; font-size: 24px; font-weight: 600; margin-bottom: 4px;`}>{formatTime(performanceMetrics.renderTime)}</div>
                <div style={`color: ${statusColors[getPerformanceStatus()]}; font-size: 12px; font-weight: 500;`}>
                  {performanceMetrics.renderTime <= 10 ? '🟢 Excellent' : performanceMetrics.renderTime <= 15 ? '🟡 Good' : '🔴 Needs Attention'}
                </div>
              </div>
              
              <div style={`background: ${theme.bg.tertiary}; padding: 16px; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <div style="display: flex; align-items: center; margin-bottom: 8px;">
                  <span style="font-size: 20px; margin-right: 8px;">🧠</span>
                  <span style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px;`}>Memory Usage</span>
                </div>
                <div style={`color: ${theme.text.primary}; font-size: 24px; font-weight: 600; margin-bottom: 4px;`}>{formatBytes(performanceMetrics.memoryUsage)}</div>
                <div style={`color: ${statusColors[getPerformanceStatus()]}; font-size: 12px; font-weight: 500;`}>
                  {performanceMetrics.memoryUsage <= 25 ? '🟢 Optimal' : performanceMetrics.memoryUsage <= 35 ? '🟡 Moderate' : '🔴 High Usage'}
                </div>
              </div>
              
              <div style={`background: ${theme.bg.tertiary}; padding: 16px; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <div style="display: flex; align-items: center; margin-bottom: 8px;">
                  <span style="font-size: 20px; margin-right: 8px;">🔧</span>
                  <span style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px;`}>Components</span>
                </div>
                <div style={`color: ${theme.text.primary}; font-size: 24px; font-weight: 600; margin-bottom: 4px;`}>{performanceMetrics.componentCount.toLocaleString()}</div>
                <div style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500;`}>
                  DOM Elements
                </div>
              </div>
              
              <div style={`background: ${theme.bg.tertiary}; padding: 16px; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <div style="display: flex; align-items: center; margin-bottom: 8px;">
                  <span style="font-size: 20px; margin-right: 8px;">📈</span>
                  <span style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px;`}>Status</span>
                </div>
                <div style={`color: ${statusColors[getPerformanceStatus()]}; font-size: 18px; font-weight: 600; margin-bottom: 4px; text-transform: capitalize;`}>
                  {getPerformanceStatus()}
                </div>
                <div style={`color: ${theme.text.secondary}; font-size: 12px; font-weight: 500;`}>
                  Overall Health
                </div>
              </div>
            </div>
            
            {/* Performance History Chart */}
            <div style={`background: ${theme.bg.tertiary}; padding: 20px; border-radius: 8px; border: 1px solid ${theme.border.primary}; margin-bottom: 24px;`}>
              <h3 style={`color: ${theme.text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                <span style="margin-right: 8px;">📊</span> Performance History
              </h3>
              <div style="display: flex; align-items: end; gap: 4px; height: 80px; margin-bottom: 8px;">
                <For each={performanceMetrics.history}>{(entry) => 
                  <div 
                    style={`background: ${statusColors[entry.renderTime <= 10 ? 'good' : entry.renderTime <= 15 ? 'caution' : 'warning']}; width: 20px; height: ${Math.min(entry.renderTime * 4, 80)}px; border-radius: 2px; opacity: 0.7; transition: opacity 0.2s;`}
                    title={`${formatTime(entry.renderTime)} at ${new Date(entry.timestamp).toLocaleTimeString()}`}
                  ></div>
                }</For>
              </div>
              <div style={`color: ${theme.text.secondary}; font-size: 11px; text-align: center;`}>Render time over last 20 updates (hover for details)</div>
            </div>
            
            {/* Performance Alerts */}
            <Show when={performanceMetrics.alerts.length > 0}>
              <div style={`background: ${theme.bg.tertiary}; padding: 20px; border-radius: 8px; border: 1px solid ${theme.border.primary}; margin-bottom: 24px;`}>
                <h3 style={`color: ${theme.text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                  <span style="margin-right: 8px;">⚠️</span> Recent Alerts
                </h3>
                <div style="space-y: 8px;">
                  <For each={performanceMetrics.alerts.slice(-5)}>{(alert) => 
                    <div style={`padding: 12px; border-left: 3px solid ${statusColors.warning}; background: ${theme.bg.primary}; border-radius: 4px; margin-bottom: 8px;`}>
                      <div style={`color: ${theme.text.primary}; font-weight: 500; font-size: 14px; margin-bottom: 4px;`}>{alert.message}</div>
                      <div style={`color: ${theme.text.secondary}; font-size: 12px;`}>{new Date(alert.timestamp).toLocaleString()}</div>
                    </div>
                  }</For>
                </div>
              </div>
            </Show>
            
            {/* Performance Tips */}
            <div style={`background: ${theme.bg.tertiary}; padding: 20px; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
              <h3 style={`color: ${theme.text.primary}; font-size: 16px; margin: 0 0 16px 0; font-weight: 600; display: flex; align-items: center;`}>
                <span style="margin-right: 8px;">💡</span> Performance Tips
              </h3>
              <div style="space-y: 8px;">
                <div style={`color: ${theme.text.secondary}; font-size: 14px; line-height: 1.5; margin-bottom: 8px;`}>
                  • High memory usage may indicate too many DOM elements or data caching
                </div>
                <div style={`color: ${theme.text.secondary}; font-size: 14px; line-height: 1.5; margin-bottom: 8px;`}>
                  • Slow render times can be improved by optimizing reactive computations
                </div>
                <div style={`color: ${theme.text.secondary}; font-size: 14px; line-height: 1.5; margin-bottom: 8px;`}>
                  • Use keyboard shortcuts to navigate more efficiently
                </div>
                <div style={`color: ${theme.text.secondary}; font-size: 14px; line-height: 1.5;`}>
                  • Consider disabling real-time simulation if performance degrades
                </div>
              </div>
            </div>
            
            {/* Footer */}
            <div style={`margin-top: 24px; padding-top: 16px; border-top: 1px solid ${theme.border.primary}; text-align: center;`}>
              <p style={`color: ${theme.text.tertiary}; font-size: 12px; margin: 0;`}>
                📊 Monitoring updates every 2 seconds • Use ⌘⇧P to access quickly
              </p>
            </div>
          </div>
        </div>
      </div>
    </Show>
  );
};

// Admin Settings Panel
const AdminSettingsPanel = () => {
  const theme = getTheme();
  
  return (
    <Show when={showAdminPanel()}>
      <div 
        style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 1500; backdrop-filter: blur(3px);"
        onclick={() => setShowAdminPanel(false)}
      >
        <div 
          style={`background: ${theme.bg.secondary}; border-radius: 12px; width: 90%; max-width: 800px; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 40px rgba(0,0,0,0.3); border: 1px solid ${theme.border.primary};`}
          onclick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div style={`background: linear-gradient(135deg, ${theme.status.info} 0%, #4A90E2 100%); padding: 30px; color: white; border-radius: 12px 12px 0 0; position: relative;`}>
            <button
              onclick={() => setShowAdminPanel(false)}
              style="position: absolute; top: 20px; right: 20px; background: rgba(255,255,255,0.2); border: none; color: white; width: 36px; height: 36px; border-radius: 50%; cursor: pointer; font-size: 18px;"
            >
              ×
            </button>
            <h2 style="margin: 0 0 8px 0; font-size: 28px; font-weight: 600;">⚙️ System Settings</h2>
            <p style="margin: 0; opacity: 0.9; font-size: 16px;">Manage your UManager Enterprise configuration</p>
          </div>

          {/* Content */}
          <div style="padding: 30px;">
            {/* Settings Grid */}
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px; margin-bottom: 30px;">
              {/* Theme Settings */}
              <div>
                <h3 style={`color: ${theme.text.primary}; font-size: 18px; margin: 0 0 16px 0; font-weight: 600;`}>🎨 Appearance</h3>
                <div style={`padding: 20px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                  <div style="margin-bottom: 16px;">
                    <label style={`display: block; font-size: 14px; color: ${theme.text.secondary}; margin-bottom: 8px; font-weight: 500;`}>Theme Mode</label>
                    <button
                      onclick={() => {
                        setDarkMode(!darkMode());
                        addNotification(`Switched to ${!darkMode() ? 'dark' : 'light'} mode`, 'success');
                      }}
                      style={`width: 100%; padding: 12px; border: 2px solid ${theme.border.accent}; background: ${theme.bg.secondary}; color: ${theme.text.accent}; border-radius: 6px; cursor: pointer; font-weight: 500; transition: all 0.2s ease; margin-bottom: 12px;`}
                    >
                      {darkMode() ? '☀️ Switch to Light Mode' : '🌙 Switch to Dark Mode'}
                    </button>
                  </div>
                  
                  <div>
                    <label style={`display: block; font-size: 14px; color: ${theme.text.secondary}; margin-bottom: 8px; font-weight: 500;`}>Real-time Updates</label>
                    <button
                      onclick={() => {
                        realtimeActive() ? stopRealtimeSimulation() : startRealtimeSimulation();
                      }}
                      style={`width: 100%; padding: 12px; border: 2px solid ${realtimeActive() ? theme.status.success : theme.border.primary}; background: ${realtimeActive() ? theme.status.success : theme.bg.secondary}; color: ${realtimeActive() ? 'white' : theme.text.primary}; border-radius: 6px; cursor: pointer; font-weight: 500; transition: all 0.2s ease;`}
                    >
                      {realtimeActive() ? '⏸️ Stop Real-time Updates' : '▶️ Start Real-time Demo'}
                    </button>
                  </div>
                </div>
              </div>

              {/* Performance & Monitoring */}
              <div>
                <h3 style={`color: ${theme.text.primary}; font-size: 18px; margin: 0 0 16px 0; font-weight: 600;`}>📊 Performance & Monitoring</h3>
                <div style={`padding: 20px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                  <div style="margin-bottom: 16px;">
                    <button
                      onclick={() => {
                        setShowPerformanceMonitor(true);
                        addNotification('Performance monitor opened', 'success');
                      }}
                      style={`width: 100%; padding: 12px; border: 2px solid #059669; background: #059669; color: white; border-radius: 6px; cursor: pointer; font-weight: 500; transition: all 0.2s ease; margin-bottom: 12px;`}
                    >
                      📊 Open Performance Monitor
                    </button>
                  </div>
                  
                  <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                    <div style="text-align: center;">
                      <div style={`font-size: 24px; font-weight: 600; color: ${theme.status.info}; margin-bottom: 4px;`}>{users.length}</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Total Users</div>
                    </div>
                    <div style="text-align: center;">
                      <div style={`font-size: 24px; font-weight: 600; color: ${theme.status.success}; margin-bottom: 4px;`}>{activityLog.length}</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Activities</div>
                    </div>
                    <div style="text-align: center;">
                      <div style={`font-size: 20px; font-weight: 600; color: ${theme.status.warning}; margin-bottom: 4px;`}>{performanceMetrics.renderTime}ms</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Render Time</div>
                    </div>
                    <div style="text-align: center;">
                      <div style={`font-size: 20px; font-weight: 600; color: ${theme.status.info}; margin-bottom: 4px;`}>{performanceMetrics.memoryUsage}MB</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Memory Usage</div>
                    </div>
                  </div>
                  
                  <div style="margin-top: 12px; text-align: center;">
                    <div style={`font-size: 12px; color: ${theme.text.secondary}; margin-bottom: 4px;`}>System Status</div>
                    <div style="display: inline-flex; align-items: center; padding: 4px 8px; background: #10B981; color: white; border-radius: 12px; font-size: 11px; font-weight: 500;">
                      <span style="width: 6px; height: 6px; background: #34D399; border-radius: 50%; margin-right: 4px;"></span>
                      Optimal
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Advanced Settings */}
            <div style="margin-bottom: 30px;">
              <h3 style={`color: ${theme.text.primary}; font-size: 18px; margin: 0 0 16px 0; font-weight: 600;`}>⚡ Advanced Features</h3>
              <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 16px;">
                <div style={`padding: 16px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary}; text-align: center;`}>
                  <div style="font-size: 24px; margin-bottom: 8px;">🔔</div>
                  <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 4px;`}>Smart Notifications</div>
                  <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Enabled</div>
                </div>
                <div style={`padding: 16px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary}; text-align: center;`}>
                  <div style="font-size: 24px; margin-bottom: 8px;">🔍</div>
                  <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 4px;`}>Advanced Search</div>
                  <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Active</div>
                </div>
                <div style={`padding: 16px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary}; text-align: center;`}>
                  <div style="font-size: 24px; margin-bottom: 8px;">📈</div>
                  <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 4px;`}>Analytics</div>
                  <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Real-time</div>
                </div>
                <div style={`padding: 16px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary}; text-align: center;`}>
                  <div style="font-size: 24px; margin-bottom: 8px;">🛡️</div>
                  <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 4px;`}>Security</div>
                  <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Enterprise</div>
                </div>
              </div>
            </div>

            {/* Feature Flags */}
            <div style="margin-bottom: 30px;">
              <h3 style={`color: ${theme.text.primary}; font-size: 18px; margin: 0 0 16px 0; font-weight: 600;`}>🚩 Feature Flags</h3>
              <div style={`padding: 20px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px;">
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <div>
                      <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 2px;`}>Advanced Analytics</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Enhanced data visualization and insights</div>
                    </div>
                    <button
                      onclick={() => toggleFeatureFlag('advancedAnalytics')}
                      style={`width: 44px; height: 24px; border-radius: 12px; border: none; cursor: pointer; transition: all 0.2s; position: relative; background: ${featureFlags.advancedAnalytics ? '#10B981' : '#D1D5DB'};`}
                    >
                      <div style={`width: 20px; height: 20px; border-radius: 50%; background: white; position: absolute; top: 2px; transition: all 0.2s; ${featureFlags.advancedAnalytics ? 'left: 22px;' : 'left: 2px;'}`}></div>
                    </button>
                  </div>
                  
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <div>
                      <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 2px;`}>Real-time Notifications</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Live updates and instant alerts</div>
                    </div>
                    <button
                      onclick={() => toggleFeatureFlag('realTimeNotifications')}
                      style={`width: 44px; height: 24px; border-radius: 12px; border: none; cursor: pointer; transition: all 0.2s; position: relative; background: ${featureFlags.realTimeNotifications ? '#10B981' : '#D1D5DB'};`}
                    >
                      <div style={`width: 20px; height: 20px; border-radius: 50%; background: white; position: absolute; top: 2px; transition: all 0.2s; ${featureFlags.realTimeNotifications ? 'left: 22px;' : 'left: 2px;'}`}></div>
                    </button>
                  </div>
                  
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <div>
                      <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 2px;`}>Bulk Operations</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Batch processing for user management</div>
                    </div>
                    <button
                      onclick={() => toggleFeatureFlag('bulkOperations')}
                      style={`width: 44px; height: 24px; border-radius: 12px; border: none; cursor: pointer; transition: all 0.2s; position: relative; background: ${featureFlags.bulkOperations ? '#10B981' : '#D1D5DB'};`}
                    >
                      <div style={`width: 20px; height: 20px; border-radius: 50%; background: white; position: absolute; top: 2px; transition: all 0.2s; ${featureFlags.bulkOperations ? 'left: 22px;' : 'left: 2px;'}`}></div>
                    </button>
                  </div>
                  
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0;">
                    <div>
                      <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-bottom: 2px;`}>Beta Features</div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>Experimental functionality (use with caution)</div>
                    </div>
                    <button
                      onclick={() => toggleFeatureFlag('betaFeatures')}
                      style={`width: 44px; height: 24px; border-radius: 12px; border: none; cursor: pointer; transition: all 0.2s; position: relative; background: ${featureFlags.betaFeatures ? '#10B981' : '#D1D5DB'};`}
                    >
                      <div style={`width: 20px; height: 20px; border-radius: 50%; background: white; position: absolute; top: 2px; transition: all 0.2s; ${featureFlags.betaFeatures ? 'left: 22px;' : 'left: 2px;'}`}></div>
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Background Jobs */}
            <div style="margin-bottom: 30px;">
              <h3 style={`color: ${theme.text.primary}; font-size: 18px; margin: 0 0 16px 0; font-weight: 600;`}>⚙️ Background Jobs</h3>
              <div style={`padding: 20px; background: ${theme.bg.tertiary}; border-radius: 8px; border: 1px solid ${theme.border.primary};`}>
                <For each={Object.entries(backgroundJobs)}>{([jobName, jobConfig]) => 
                  <div style={`display: flex; justify-content: space-between; align-items: center; padding: 16px 0; border-bottom: 1px solid ${theme.border.primary}; ${jobName === 'backupTask' ? 'border-bottom: none;' : ''}`}>
                    <div style="flex: 1;">
                      <div style="display: flex; align-items: center; margin-bottom: 4px;">
                        <div style={`font-size: 14px; font-weight: 500; color: ${theme.text.primary}; margin-right: 8px; text-transform: capitalize;`}>
                          {jobName.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase())}
                        </div>
                        <div style={`padding: 2px 8px; border-radius: 12px; font-size: 10px; font-weight: 600; text-transform: uppercase; background: ${getJobStatusColor(jobConfig.status)}20; color: ${getJobStatusColor(jobConfig.status)};`}>
                          {jobConfig.status}
                        </div>
                      </div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary}; margin-bottom: 2px;`}>
                        Interval: {jobConfig.interval < 60 ? `${jobConfig.interval}s` : jobConfig.interval < 3600 ? `${Math.floor(jobConfig.interval/60)}m` : `${Math.floor(jobConfig.interval/3600)}h`}
                      </div>
                      <div style={`font-size: 12px; color: ${theme.text.secondary};`}>
                        Last run: {formatLastRun(jobConfig.lastRun)}
                      </div>
                    </div>
                    <div style="display: flex; align-items: center; gap: 8px;">
                      <select
                        value={jobConfig.interval}
                        onchange={(e) => updateJobInterval(jobName, parseInt(e.target.value))}
                        style={`padding: 4px 8px; border: 1px solid ${theme.border.primary}; border-radius: 4px; background: ${theme.bg.primary}; color: ${theme.text.primary}; font-size: 12px;`}
                        disabled={!jobConfig.enabled}
                      >
                        <option value={30}>30s</option>
                        <option value={60}>1m</option>
                        <option value={300}>5m</option>
                        <option value={900}>15m</option>
                        <option value={1800}>30m</option>
                        <option value={3600}>1h</option>
                        <option value={14400}>4h</option>
                        <option value={86400}>24h</option>
                      </select>
                      <button
                        onclick={() => toggleBackgroundJob(jobName)}
                        style={`width: 40px; height: 22px; border-radius: 11px; border: none; cursor: pointer; transition: all 0.2s; position: relative; background: ${jobConfig.enabled ? '#10B981' : '#D1D5DB'};`}
                      >
                        <div style={`width: 18px; height: 18px; border-radius: 50%; background: white; position: absolute; top: 2px; transition: all 0.2s; ${jobConfig.enabled ? 'left: 20px;' : 'left: 2px;'}`}></div>
                      </button>
                    </div>
                  </div>
                }</For>
              </div>
            </div>

            {/* Quick Actions */}
            <div style={`padding-top: 20px; border-top: 1px solid ${theme.border.primary}; display: flex; justify-content: space-between; align-items: center;`}>
              <div style={`color: ${theme.text.secondary}; font-size: 14px;`}>
                System running smoothly • Last updated just now
              </div>
              <div style="display: flex; gap: 12px;">
                <button
                  onclick={() => {
                    addNotification('System cache cleared successfully', 'success');
                    addNotification('Performance optimized', 'info', 4000);
                  }}
                  style={`padding: 10px 20px; background: ${theme.status.warning}; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer;`}
                >
                  🔄 Clear Cache
                </button>
                <button
                  onclick={() => {
                    addNotification('Export started - this may take a few moments', 'info');
                    if (typeof window !== 'undefined') {
                      window.setTimeout(() => {
                        addNotification('System data exported successfully', 'success');
                      }, 2000);
                    }
                  }}
                  style={`padding: 10px 20px; background: ${theme.status.info}; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer;`}
                >
                  📁 Export Data
                </button>
                <button
                  onclick={() => setShowAdminPanel(false)}
                  style={`padding: 10px 20px; background: ${theme.bg.tertiary}; color: ${theme.text.primary}; border: 1px solid ${theme.border.primary}; border-radius: 6px; font-weight: 500; cursor: pointer;`}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Show>
  );
};

// Theme Toggle Component
const ThemeToggle = () => {
  return (
    <div style="position: fixed; top: 20px; left: 20px; z-index: 2000; display: flex; flex-direction: column; gap: 8px;">
      <button
        onclick={() => {
          const newMode = !darkMode();
          setDarkMode(newMode);
          console.log('Dark mode toggled to:', newMode);
          addNotification(`Switched to ${newMode ? 'dark' : 'light'} mode`, 'info', 3000);
        }}
        style={`padding: 12px; border: 2px solid ${getTheme().border.accent}; background: ${getTheme().bg.secondary}; color: ${getTheme().text.accent}; border-radius: 50%; cursor: pointer; font-size: 18px; width: 50px; height: 50px; display: flex; align-items: center; justify-content: center; transition: all 0.3s ease; box-shadow: 0 4px 12px rgba(0,0,0,0.15);`}
        title={`Switch to ${darkMode() ? 'light' : 'dark'} mode`}
      >
        {darkMode() ? '☀️' : '🌙'}
      </button>
      
      <button
        onclick={() => {
          setShowShortcuts(true);
          addNotification('Keyboard shortcuts panel opened - press ? anytime for help!', 'info');
        }}
        style={`padding: 12px; border: 2px solid ${getTheme().border.primary}; background: ${getTheme().bg.secondary}; color: ${getTheme().text.primary}; border-radius: 50%; cursor: pointer; font-size: 16px; width: 50px; height: 50px; display: flex; align-items: center; justify-content: center; transition: all 0.3s ease; box-shadow: 0 4px 12px rgba(0,0,0,0.15);`}
        title="Show keyboard shortcuts (or press ?)"
      >
        ?
      </button>
    </div>
  );
};

// Notification Component
const NotificationCenter = () => {
  return (
    <div style="position: fixed; top: 20px; right: 20px; z-index: 2000; max-width: 400px;">
      <For each={notifications}>
        {(notification) => (
          <div 
            style={`margin-bottom: 12px; padding: 16px 20px; border-radius: 8px; backdrop-filter: blur(10px); transition: all 0.3s ease; box-shadow: 0 8px 32px rgba(0,0,0,${darkMode() ? '0.3' : '0.15'}); border: 1px solid ${getTheme().border.primary}; ${
              notification.type === 'success' ? 
                `background: linear-gradient(135deg, ${darkMode() ? '#1B4332' : '#E3FCEF'} 0%, ${darkMode() ? '#2D5A3D' : '#C3F4D6'} 100%); border-left: 4px solid #006644; color: ${darkMode() ? '#4ADE80' : '#006644'};` :
              notification.type === 'error' ? 
                `background: linear-gradient(135deg, ${darkMode() ? '#4C1D1D' : '#FFEBE6'} 0%, ${darkMode() ? '#5D2A2A' : '#FFD5CC'} 100%); border-left: 4px solid #DE350B; color: ${darkMode() ? '#F87171' : '#DE350B'};` :
              notification.type === 'warning' ? 
                `background: linear-gradient(135deg, ${darkMode() ? '#4D3319' : '#FFF4E6'} 0%, ${darkMode() ? '#5E4021' : '#FFE8CC'} 100%); border-left: 4px solid #FF8B00; color: ${darkMode() ? '#FBBF24' : '#FF8B00'};` :
                `background: linear-gradient(135deg, ${darkMode() ? '#1E3A5F' : '#E6F3FF'} 0%, ${darkMode() ? '#2A4A70' : '#CCE7FF'} 100%); border-left: 4px solid #0052CC; color: ${darkMode() ? '#60A5FA' : '#0052CC'};`
            }`}
          >
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
              <div style="flex: 1; font-size: 14px; font-weight: 500; line-height: 1.4;">
                {notification.message}
              </div>
              <button
                onclick={() => removeNotification(notification.id)}
                style="margin-left: 12px; background: none; border: none; font-size: 16px; cursor: pointer; opacity: 0.7; line-height: 1; color: inherit;"
              >
                ×
              </button>
            </div>
          </div>
        )}
      </For>
    </div>
  );
};

// Simple LoginPage
const LoginPage = () => {
  return (
    <div style={`min-height: 100vh; display: flex; align-items: center; justify-content: center; background: ${getTheme().bg.primary}; font-family: system-ui; transition: all 0.3s ease;`}>
      <div style={`background: ${getTheme().bg.secondary}; padding: 40px; border-radius: 8px; box-shadow: 0 8px 32px rgba(0,0,0,${darkMode() ? '0.3' : '0.15'}); max-width: 420px; width: 100%; border: 1px solid ${getTheme().border.primary};`}>
        <h1 style={`text-align: center; margin-bottom: 20px; color: ${getTheme().text.primary}; font-size: 32px; font-weight: 600;`}>Welcome to UManager</h1>
        <p style={`text-align: center; margin-bottom: 30px; color: ${getTheme().text.secondary}; font-size: 16px;`}>Enterprise User Management Platform</p>
        
        <div style="text-align: center; margin-bottom: 24px;">
          <a href="/dashboard" style={`background: ${getTheme().status.info}; color: white; padding: 14px 28px; border-radius: 6px; text-decoration: none; display: inline-block; font-weight: 500; margin-right: 12px; margin-bottom: 8px; transition: all 0.2s ease; box-shadow: 0 4px 12px rgba(0,82,204,0.2);`}>
            📊 Analytics Dashboard
          </a>
          <a href="/users" style={`background: ${getTheme().status.success}; color: white; padding: 14px 28px; border-radius: 6px; text-decoration: none; display: inline-block; font-weight: 500; margin-bottom: 8px; transition: all 0.2s ease; box-shadow: 0 4px 12px rgba(0,102,68,0.2);`}>
            👥 User Management
          </a>
        </div>
        
        {/* Features Grid */}
        <div style={`padding: 20px; background: ${getTheme().bg.tertiary}; border-radius: 6px; margin-top: 24px; border: 1px solid ${getTheme().border.primary};`}>
          <div style={`text-align: center; color: ${getTheme().text.secondary}; font-size: 14px; margin-bottom: 16px; font-weight: 600;`}>
            🚀 Enterprise Features
          </div>
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; font-size: 13px;">
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">📈</span>
              Analytics & Reports
            </div>
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">🔔</span>
              Smart Notifications
            </div>
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">👤</span>
              User Profiles
            </div>
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">🌙</span>
              Dark Mode
            </div>
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">🔍</span>
              Advanced Search
            </div>
            <div style={`color: ${getTheme().text.secondary}; display: flex; align-items: center;`}>
              <span style="margin-right: 8px;">⚡</span>
              Real-time Updates
            </div>
          </div>
        </div>
      </div>
      
      <ThemeToggle />
      <NotificationCenter />
    </div>
  );
};

// Simple CreateModal
const CreateModal = () => {
  const [name, setName] = createSignal('');
  const [email, setEmail] = createSignal('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (name() && email()) {
      const newUser = {
        id: Math.max(...users.map(u => u.id)) + 1,
        name: name(),
        email: email(),
        status: 'Active',
        role: 'User',
        department: 'General',
        lastLogin: null,
        createdAt: new Date().toISOString(),
        loginCount: 0,
        activityScore: 0,
        permissions: ['read'],
        lastActivity: new Date().toISOString()
      };
      setUsers([...users, newUser]);
      logActivity(newUser.id, 'User Created', `Created new user: ${newUser.name}`);
      addNotification(`User "${newUser.name}" created successfully!`, 'success');
      setShowCreateModal(false);
      setName('');
      setEmail('');
    }
  };

  return (
    <Show when={showCreateModal()}>
      <div 
        style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000;"
        onclick={() => setShowCreateModal(false)}
      >
        <div 
          style="background: white; padding: 30px; border-radius: 6px; width: 90%; max-width: 500px;"
          onclick={(e) => e.stopPropagation()}
        >
          <h2 style="margin: 0 0 20px 0;">Create New User</h2>
          <form onsubmit={handleSubmit}>
            <div style="margin-bottom: 16px;">
              <label style="display: block; margin-bottom: 4px; font-weight: 500;">Name</label>
              <input 
                type="text"
                placeholder="Enter name"
                value={name()}
                oninput={(e) => setName(e.target.value)}
                style="width: 100%; padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; box-sizing: border-box;"
                required
              />
            </div>
            <div style="margin-bottom: 20px;">
              <label style="display: block; margin-bottom: 4px; font-weight: 500;">Email</label>
              <input 
                type="email"
                placeholder="Enter email"
                value={email()}
                oninput={(e) => setEmail(e.target.value)}
                style="width: 100%; padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; box-sizing: border-box;"
                required
              />
            </div>
            <div style="display: flex; gap: 10px; justify-content: flex-end;">
              <button 
                type="button"
                onclick={() => setShowCreateModal(false)}
                style="padding: 8px 16px; border: 1px solid #DFE1E6; background: white; border-radius: 4px; cursor: pointer;"
              >
                Cancel
              </button>
              <button 
                type="submit"
                style="padding: 8px 16px; background: #0052CC; color: white; border: none; border-radius: 4px; cursor: pointer;"
              >
                Create
              </button>
            </div>
          </form>
        </div>
      </div>
    </Show>
  );
};

// Edit User Modal
const EditModal = () => {
  return (
    <Show when={showEditModal() && editingUser()}>
      {(() => {
        const user = editingUser();
        const [name, setName] = createSignal(user.name);
        const [email, setEmail] = createSignal(user.email);
        const [status, setStatus] = createSignal(user.status);

        const handleSubmit = (e) => {
          e.preventDefault();
          if (name() && email() && user) {
            // Update the user in the store
            const updatedUser = { ...user, name: name(), email: email(), status: status(), lastActivity: new Date().toISOString() };
            setUsers(users.map(u => 
              u.id === user.id ? updatedUser : u
            ));
            logActivity(user.id, 'User Updated', `Updated user information for ${name()}`);
            addNotification(`User "${name()}" updated successfully!`, 'success');
            setShowEditModal(false);
            setEditingUser(null);
          }
        };

        return (
          <div 
            style="position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000;"
            onclick={() => {
              setShowEditModal(false);
              setEditingUser(null);
            }}
          >
            <div 
              style="background: white; padding: 30px; border-radius: 6px; width: 90%; max-width: 500px; box-shadow: 0 4px 20px rgba(0,0,0,0.2);"
              onclick={(e) => e.stopPropagation()}
            >
              <h2 style="margin: 0 0 20px 0; color: #172B4D;">Edit User: {user.name}</h2>
              <form onsubmit={handleSubmit}>
                <div style="margin-bottom: 16px;">
                  <label style="display: block; margin-bottom: 4px; font-weight: 500; color: #172B4D;">Name</label>
                  <input 
                    type="text"
                    placeholder="Enter name"
                    value={name()}
                    oninput={(e) => setName(e.target.value)}
                    style="width: 100%; padding: 10px 12px; border: 2px solid #DFE1E6; border-radius: 4px; box-sizing: border-box; font-size: 14px;"
                    required
                    autofocus
                  />
                </div>
                <div style="margin-bottom: 16px;">
                  <label style="display: block; margin-bottom: 4px; font-weight: 500; color: #172B4D;">Email</label>
                  <input 
                    type="email"
                    placeholder="Enter email"
                    value={email()}
                    oninput={(e) => setEmail(e.target.value)}
                    style="width: 100%; padding: 10px 12px; border: 2px solid #DFE1E6; border-radius: 4px; box-sizing: border-box; font-size: 14px;"
                    required
                  />
                </div>
                <div style="margin-bottom: 20px;">
                  <label style="display: block; margin-bottom: 4px; font-weight: 500; color: #172B4D;">Status</label>
                  <select 
                    value={status()}
                    onchange={(e) => setStatus(e.target.value)}
                    style="width: 100%; padding: 10px 12px; border: 2px solid #DFE1E6; border-radius: 4px; box-sizing: border-box; font-size: 14px; background: white;"
                  >
                    <option value="Active">Active</option>
                    <option value="Inactive">Inactive</option>
                  </select>
                </div>
                <div style="display: flex; gap: 10px; justify-content: flex-end;">
                  <button 
                    type="button"
                    onclick={() => {
                      setShowEditModal(false);
                      setEditingUser(null);
                    }}
                    style="padding: 10px 16px; border: 1px solid #DFE1E6; background: white; color: #172B4D; border-radius: 4px; cursor: pointer; font-weight: 500;"
                  >
                    Cancel
                  </button>
                  <button 
                    type="submit"
                    style="padding: 10px 16px; background: #0052CC; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: 500; box-shadow: 0 2px 4px rgba(0,82,204,0.2);"
                  >
                    Update User
                  </button>
                </div>
              </form>
            </div>
          </div>
        );
      })()}
    </Show>
  );
};

// Users Page
const UsersPage = () => {
  const deleteUser = (id) => {
    const user = users.find(u => u.id === id);
    if (window.confirm('Are you sure you want to delete this user?')) {
      setUsers(users.filter(user => user.id !== id));
      if (user) {
        logActivity(id, 'User Deleted', `Deleted user: ${user.name}`);
        addNotification(`User "${user.name}" deleted successfully!`, 'success');
      }
    }
  };

  const editUser = (user) => {
    setEditingUser(user);
    setShowEditModal(true);
  };

  // Advanced filtered and sorted users
  const filteredUsers = () => {
    let filtered = users;

    // Text search
    const query = searchQuery().toLowerCase();
    if (query) {
      filtered = filtered.filter(user => 
        user.name.toLowerCase().includes(query) || 
        user.email.toLowerCase().includes(query) ||
        user.department.toLowerCase().includes(query)
      );
    }

    // Status filter
    if (statusFilter() !== 'all') {
      filtered = filtered.filter(user => user.status === statusFilter());
    }

    // Role filter
    if (roleFilter() !== 'all') {
      filtered = filtered.filter(user => user.role === roleFilter());
    }

    // Department filter
    if (departmentFilter() !== 'all') {
      filtered = filtered.filter(user => user.department === departmentFilter());
    }

    // Sorting
    const sortField = sortBy();
    const order = sortOrder();
    
    filtered = [...filtered].sort((a, b) => {
      let aVal = a[sortField];
      let bVal = b[sortField];
      
      if (sortField === 'createdAt' || sortField === 'lastLogin') {
        aVal = new Date(aVal || 0);
        bVal = new Date(bVal || 0);
      } else if (typeof aVal === 'string') {
        aVal = aVal.toLowerCase();
        bVal = bVal.toLowerCase();
      }
      
      if (aVal < bVal) return order === 'asc' ? -1 : 1;
      if (aVal > bVal) return order === 'asc' ? 1 : -1;
      return 0;
    });

    return filtered;
  };

  // Statistics
  const stats = () => {
    const total = users.length;
    const active = users.filter(u => u.status === 'Active').length;
    const inactive = users.filter(u => u.status === 'Inactive').length;
    return { total, active, inactive };
  };

  // Bulk selection functions
  const toggleUserSelection = (user) => {
    const selected = selectedUsers();
    const isSelected = selected.some(u => u.id === user.id);
    
    if (isSelected) {
      setSelectedUsers(selected.filter(u => u.id !== user.id));
    } else {
      setSelectedUsers([...selected, user]);
    }
  };

  const toggleAllSelection = () => {
    const filtered = filteredUsers();
    if (selectedUsers().length === filtered.length) {
      setSelectedUsers([]);
    } else {
      setSelectedUsers([...filtered]);
    }
  };

  const bulkDelete = () => {
    const selected = selectedUsers();
    if (selected.length === 0) return;
    
    if (window.confirm(`Are you sure you want to delete ${selected.length} user(s)?`)) {
      const selectedIds = selected.map(u => u.id);
      setUsers(users.filter(u => !selectedIds.includes(u.id)));
      selected.forEach(user => {
        logActivity(user.id, 'Bulk Delete', `User deleted in bulk operation: ${user.name}`);
      });
      setSelectedUsers([]);
      addNotification(`${selected.length} user(s) deleted successfully!`, 'success');
    }
  };

  const exportUsers = () => {
    const dataToExport = selectedUsers().length > 0 ? selectedUsers() : filteredUsers();
    const csvContent = [
      'Name,Email,Status',
      ...dataToExport.map(user => `"${user.name}","${user.email}","${user.status}"`)
    ].join('\n');
    
    const blob = new window.Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'users.csv';
    a.click();
    window.URL.revokeObjectURL(url);
    
    addNotification(`Exported ${dataToExport.length} user(s) to CSV!`, 'success');
  };

  return (
    <>
      {/* Add responsive CSS */}
      <style>
        {`
          @media (max-width: 768px) {
            .mobile-responsive {
              flex-direction: column !important;
              gap: 10px !important;
            }
            .mobile-stack {
              flex-direction: column !important;
            }
            .mobile-full-width {
              width: 100% !important;
              max-width: none !important;
            }
            .mobile-hide {
              display: none !important;
            }
            .mobile-show {
              display: block !important;
            }
            .mobile-scroll {
              overflow-x: auto !important;
              -webkit-overflow-scrolling: touch;
            }
          }
          
          /* Selection highlight */
          tr:hover {
            background-color: ${darkMode() ? '#3A3B45' : '#F7F8F9'} !important;
          }
          
          /* Button hover effects */
          button:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.12);
          }
          
          /* Pulse animation for live indicator */
          @keyframes pulse {
            0% {
              box-shadow: 0 0 0 0 rgba(52, 211, 153, 0.7);
            }
            70% {
              box-shadow: 0 0 0 10px rgba(52, 211, 153, 0);
            }
            100% {
              box-shadow: 0 0 0 0 rgba(52, 211, 153, 0);
            }
          }
        `}
      </style>

      <div style={`min-height: 100vh; background: ${getTheme().bg.primary}; font-family: system-ui; transition: all 0.3s ease;`}>
        <div style="padding: 20px;">
          {/* Header */}
          <div class="mobile-responsive" style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px;">
            <div style="flex: 1;">
              <h1 style={`color: ${getTheme().text.primary}; margin: 0 0 5px 0; font-size: 28px; font-weight: 600;`}>User Management</h1>
              <p style={`color: ${getTheme().text.secondary}; margin: 0; font-size: 14px;`}>
                {stats().total} total • {stats().active} active • {stats().inactive} inactive
                {selectedUsers().length > 0 && ` • ${selectedUsers().length} selected`}
              </p>
            </div>
            <div class="mobile-stack" style="display: flex; gap: 10px; flex-shrink: 0;">
              <button 
                style={`background: ${getTheme().status.info}; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-weight: 500; box-shadow: 0 4px 12px rgba(0,82,204,0.2); transition: all 0.2s ease;`}
                onclick={() => setShowCreateModal(true)}
              >
                Create User
              </button>
            </div>
          </div>

          {/* Advanced Search and Filters */}
          <div style={`background: ${getTheme().bg.secondary}; padding: 20px; border-radius: 8px; box-shadow: 0 8px 32px rgba(0,0,0,${darkMode() ? '0.3' : '0.15'}); border: 1px solid ${getTheme().border.primary}; margin-bottom: 20px;`}>
            {/* Search Bar */}
            <div style="margin-bottom: 16px;">
              <input 
                type="text"
                placeholder="Search by name, email, or department..."
                value={searchQuery()}
                oninput={(e) => setSearchQuery(e.target.value)}
                style={`width: 100%; padding: 12px 16px; border: 2px solid ${getTheme().border.primary}; border-radius: 6px; font-size: 14px; box-sizing: border-box; transition: all 0.2s; background: ${getTheme().bg.tertiary}; color: ${getTheme().text.primary};`}
                onfocus={(e) => e.target.style.borderColor = getTheme().border.accent}
                onblur={(e) => e.target.style.borderColor = getTheme().border.primary}
              />
            </div>
            
            {/* Filters Row */}
            <div class="mobile-responsive" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin-bottom: 16px;">
              <select 
                value={statusFilter()}
                onchange={(e) => setStatusFilter(e.target.value)}
                style="padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; font-size: 13px; background: white;"
              >
                <option value="all">All Statuses</option>
                <option value="Active">Active Only</option>
                <option value="Inactive">Inactive Only</option>
              </select>
              
              <select 
                value={roleFilter()}
                onchange={(e) => setRoleFilter(e.target.value)}
                style="padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; font-size: 13px; background: white;"
              >
                <option value="all">All Roles</option>
                <option value="Admin">Admin</option>
                <option value="Manager">Manager</option>
                <option value="User">User</option>
              </select>
              
              <select 
                value={departmentFilter()}
                onchange={(e) => setDepartmentFilter(e.target.value)}
                style="padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; font-size: 13px; background: white;"
              >
                <option value="all">All Departments</option>
                <For each={[...new Set(users.map(u => u.department))]}>
                  {(dept) => <option value={dept}>{dept}</option>}
                </For>
              </select>
              
              <select 
                value={sortBy()}
                onchange={(e) => setSortBy(e.target.value)}
                style="padding: 8px 12px; border: 1px solid #DFE1E6; border-radius: 4px; font-size: 13px; background: white;"
              >
                <option value="name">Sort by Name</option>
                <option value="email">Sort by Email</option>
                <option value="status">Sort by Status</option>
                <option value="role">Sort by Role</option>
                <option value="department">Sort by Department</option>
                <option value="loginCount">Sort by Logins</option>
                <option value="activityScore">Sort by Activity</option>
                <option value="createdAt">Sort by Created Date</option>
                <option value="lastLogin">Sort by Last Login</option>
              </select>
              
              <button
                onclick={() => setSortOrder(sortOrder() === 'asc' ? 'desc' : 'asc')}
                style={`padding: 8px 12px; border: 1px solid #0052CC; background: ${sortOrder() === 'asc' ? '#E6F3FF' : '#0052CC'}; color: ${sortOrder() === 'asc' ? '#0052CC' : 'white'}; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 500;`}
                title="Toggle sort order"
              >
                {sortOrder() === 'asc' ? '↑ A-Z' : '↓ Z-A'}
              </button>
            </div>

            {/* Action Bar */}
            <div style="display: flex; justify-content: space-between; align-items: center; padding-top: 16px; border-top: 1px solid #F4F5F7;">
              <div style="color: #6B778C; font-size: 14px;">
                Showing {filteredUsers().length} of {users.length} users
                {selectedUsers().length > 0 && ` • ${selectedUsers().length} selected`}
              </div>
              
              <div style="display: flex; gap: 8px;">
                <button
                  onclick={() => {
                    setSearchQuery('');
                    setStatusFilter('all');
                    setRoleFilter('all');
                    setDepartmentFilter('all');
                    setSortBy('name');
                    setSortOrder('asc');
                    addNotification('Filters cleared', 'info');
                  }}
                  style="padding: 6px 12px; border: 1px solid #DFE1E6; color: #6B778C; background: white; border-radius: 4px; cursor: pointer; font-size: 12px;"
                >
                  Clear Filters
                </button>
                
                <button 
                  onclick={exportUsers}
                  style="padding: 6px 12px; border: 1px solid #0052CC; color: #0052CC; background: white; border-radius: 4px; cursor: pointer; font-size: 12px;"
                  title={selectedUsers().length > 0 ? `Export ${selectedUsers().length} selected users` : 'Export all visible users'}
                >
                  📊 Export CSV
                </button>
                
                <Show when={selectedUsers().length > 0}>
                  <button 
                    onclick={bulkDelete}
                    style="padding: 6px 12px; border: 1px solid #DE350B; color: #DE350B; background: white; border-radius: 4px; cursor: pointer; font-size: 12px;"
                  >
                    🗑️ Delete ({selectedUsers().length})
                  </button>
                </Show>
              </div>
            </div>
          </div>
        
        <div style="background: white; border-radius: 6px; box-shadow: 0 1px 1px rgba(9, 30, 66, 0.25); overflow: hidden;">
          <Show when={filteredUsers().length > 0} fallback={
            <div style="padding: 40px; text-align: center;">
              <div style="color: #6B778C; font-size: 16px; margin-bottom: 10px;">
                {searchQuery() ? `No users found matching "${searchQuery()}"` : "No users found"}
              </div>
              <Show when={searchQuery()}>
                <button 
                  onclick={() => setSearchQuery('')}
                  style="color: #0052CC; background: none; border: none; cursor: pointer; text-decoration: underline;"
                >
                  Clear search
                </button>
              </Show>
            </div>
          }>
            <div class="mobile-scroll">
              <table style="width: 100%; border-collapse: collapse; min-width: 600px;">
                <thead>
                  <tr style="background: #F7F8F9;">
                    <th style="padding: 12px 16px; text-align: left; border-bottom: 1px solid #DFE1E6; width: 40px;">
                      <input 
                        type="checkbox"
                        checked={selectedUsers().length === filteredUsers().length && filteredUsers().length > 0}
                        onchange={toggleAllSelection}
                        style="cursor: pointer;"
                      />
                    </th>
                    <th style="padding: 12px 16px; text-align: left; border-bottom: 1px solid #DFE1E6; font-weight: 600;">Name</th>
                    <th style="padding: 12px 16px; text-align: left; border-bottom: 1px solid #DFE1E6; font-weight: 600;" class="mobile-hide">Email</th>
                    <th style="padding: 12px 16px; text-align: left; border-bottom: 1px solid #DFE1E6; font-weight: 600;">Status</th>
                    <th style="padding: 12px 16px; text-align: right; border-bottom: 1px solid #DFE1E6; font-weight: 600;">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <For each={filteredUsers()}>
                    {(user) => (
                      <tr style={`transition: background-color 0.2s; ${selectedUsers().some(u => u.id === user.id) ? 'background: #F4F8FF;' : ''}`}>
                        <td style="padding: 12px 16px; border-bottom: 1px solid #F1F2F4;">
                          <input 
                            type="checkbox"
                            checked={selectedUsers().some(u => u.id === user.id)}
                            onchange={() => toggleUserSelection(user)}
                            style="cursor: pointer;"
                          />
                        </td>
                        <td style="padding: 12px 16px; border-bottom: 1px solid #F1F2F4;">
                          <div>
                            <button
                              onclick={() => {
                                setSelectedProfile(user);
                                setShowProfileModal(true);
                                addNotification(`Viewing profile for ${user.name}`, 'info');
                              }}
                              style="background: none; border: none; font-weight: 500; color: #0052CC; text-decoration: none; cursor: pointer; font-size: 14px; text-align: left; padding: 0;"
                            >
                              {user.name}
                            </button>
                            <div class="mobile-show" style="font-size: 12px; color: #6B778C; display: none;">
                              {user.email}
                            </div>
                          </div>
                        </td>
                        <td style="padding: 12px 16px; border-bottom: 1px solid #F1F2F4; color: #6B778C;" class="mobile-hide">{user.email}</td>
                        <td style="padding: 12px 16px; border-bottom: 1px solid #F1F2F4;">
                          <span style={`padding: 3px 8px; border-radius: 3px; font-size: 12px; font-weight: 500; ${
                            user.status === 'Active' 
                              ? 'background: #E3FCEF; color: #006644;' 
                              : 'background: #F7F8F9; color: #6B778C;'
                          }`}>
                            {user.status}
                          </span>
                        </td>
                        <td style="padding: 12px 16px; border-bottom: 1px solid #F1F2F4; text-align: right;">
                          <div style="display: flex; gap: 4px; justify-content: flex-end;">
                            <button 
                              style="background: transparent; border: 1px solid #0052CC; color: #0052CC; padding: 5px 8px; border-radius: 3px; cursor: pointer; font-size: 11px; transition: all 0.2s;"
                              onclick={() => editUser(user)}
                              title="Edit user"
                            >
                              ✏️
                            </button>
                            <button 
                              style="background: transparent; border: 1px solid #DE350B; color: #DE350B; padding: 5px 8px; border-radius: 3px; cursor: pointer; font-size: 11px; transition: all 0.2s;"
                              onclick={() => deleteUser(user.id)}
                              title="Delete user"
                            >
                              🗑️
                            </button>
                          </div>
                        </td>
                      </tr>
                    )}
                  </For>
                </tbody>
              </table>
            </div>
          </Show>
        </div>
        
          <div style="margin-top: 20px; display: flex; justify-content: space-between; align-items: center;">
            <a href="/login" style="color: #0052CC; text-decoration: underline; font-weight: 500;">← Back to Login</a>
            <div style="color: #6B778C; font-size: 12px;">
              {filteredUsers().length} of {users.length} users shown
            </div>
          </div>
        </div>
        
        {/* Navigation Bar */}
        <div style="margin-top: 20px; padding: 20px; background: white; border-radius: 6px; box-shadow: 0 1px 1px rgba(9, 30, 66, 0.25);">
          <div style="display: flex; justify-content: center; gap: 20px; align-items: center;">
            <a href="/dashboard" style="color: #0052CC; text-decoration: none; padding: 8px 16px; border-radius: 4px; font-weight: 500; transition: all 0.2s; background: rgba(0, 82, 204, 0.1);">
              📊 Dashboard
            </a>
            <a href="/users" style="color: #0052CC; text-decoration: none; padding: 8px 16px; border-radius: 4px; font-weight: 500; transition: all 0.2s;">
              👥 Users
            </a>
            <a href="/login" style="color: #6B778C; text-decoration: none; padding: 8px 16px; border-radius: 4px; font-weight: 500; transition: all 0.2s;">
              🔓 Login
            </a>
          </div>
        </div>

        <CreateModal />
        <EditModal />
        <UserProfileModal />
        <PerformanceMonitorPanel />
        <KeyboardShortcutsPanel />
        <ThemeToggle />
        <NotificationCenter />
      </div>
    </>
  );
};

// Analytics Dashboard
const Dashboard = () => {
  // Show welcome notification on load
  window.setTimeout(() => {
    addNotification('🎉 Welcome to UManager Enterprise Dashboard! All analytics and features are now active.', 'success', 8000);
  }, 500);

  const stats = () => {
    const total = users.length;
    const active = users.filter(u => u.status === 'Active').length;
    const inactive = users.filter(u => u.status === 'Inactive').length;
    
    // Recent users (last 7 days)
    const sevenDaysAgo = new Date();
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
    const recentUsers = users.filter(u => new Date(u.createdAt) > sevenDaysAgo).length;
    
    // Role distribution
    const roles = ['Admin', 'Manager', 'User'];
    const roleStats = roles.map(role => ({
      role,
      count: users.filter(u => u.role === role).length,
      percentage: Math.round((users.filter(u => u.role === role).length / total) * 100)
    }));
    
    // Department distribution
    const departments = [...new Set(users.map(u => u.department))];
    const deptStats = departments.map(dept => ({
      department: dept,
      count: users.filter(u => u.department === dept).length
    }));
    
    return { total, active, inactive, recentUsers, roleStats, deptStats };
  };
  
  
  return (
    <div style={`min-height: 100vh; background: ${getTheme().bg.primary}; font-family: system-ui; padding: 20px; transition: all 0.3s ease;`}>
      <div style="max-width: 1200px; margin: 0 auto;">
        {/* Header */}
        <div style="margin-bottom: 30px;">
          <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 8px;">
            <h1 style={`color: ${getTheme().text.primary}; margin: 0; font-size: 32px; font-weight: 600;`}>Dashboard</h1>
            <Show when={realtimeActive()}>
              <div style="display: flex; align-items: center; gap: 6px; padding: 6px 12px; background: rgba(16, 185, 129, 0.1); border: 1px solid #10B981; border-radius: 20px;">
                <span style="display: inline-block; width: 6px; height: 6px; background: #10B981; border-radius: 50%; animation: pulse 2s infinite;"></span>
                <span style="color: #10B981; font-size: 12px; font-weight: 500;">LIVE</span>
              </div>
            </Show>
          </div>
          <p style={`color: ${getTheme().text.secondary}; margin: 0; font-size: 16px;`}>
            Welcome back! Here's what's happening with your users.
            {realtimeActive() && <span style="color: #10B981;"> Real-time updates active.</span>}
          </p>
        </div>

        {/* Statistics Cards */}
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px;">
          <div style={`background: ${getTheme().bg.secondary}; padding: 24px; border-radius: 8px; box-shadow: 0 8px 32px rgba(0,0,0,${darkMode() ? '0.3' : '0.15'}); border: 1px solid ${getTheme().border.primary};`}>
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
              <div>
                <h3 style={`color: ${getTheme().text.secondary}; font-size: 14px; margin: 0 0 8px 0; text-transform: uppercase; font-weight: 500;`}>Total Users</h3>
                <div style={`font-size: 28px; font-weight: 600; color: ${getTheme().text.primary};`}>{stats().total}</div>
              </div>
              <div style={`background: ${darkMode() ? 'rgba(76, 175, 80, 0.2)' : '#E3FCEF'}; color: #006644; padding: 8px; border-radius: 6px; font-size: 20px;`}>👥</div>
            </div>
          </div>
          
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
              <div>
                <h3 style="color: #6B778C; font-size: 14px; margin: 0 0 8px 0; text-transform: uppercase; font-weight: 500;">Active Users</h3>
                <div style="font-size: 28px; font-weight: 600; color: #006644;">{stats().active}</div>
              </div>
              <div style="background: #E3FCEF; color: #006644; padding: 8px; border-radius: 6px; font-size: 20px;">✅</div>
            </div>
          </div>
          
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
              <div>
                <h3 style="color: #6B778C; font-size: 14px; margin: 0 0 8px 0; text-transform: uppercase; font-weight: 500;">Inactive Users</h3>
                <div style="font-size: 28px; font-weight: 600; color: #DE350B;">{stats().inactive}</div>
              </div>
              <div style="background: #FFEBE6; color: #DE350B; padding: 8px; border-radius: 6px; font-size: 20px;">⏸️</div>
            </div>
          </div>
          
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
              <div>
                <h3 style="color: #6B778C; font-size: 14px; margin: 0 0 8px 0; text-transform: uppercase; font-weight: 500;">New This Week</h3>
                <div style="font-size: 28px; font-weight: 600; color: #0052CC;">{stats().recentUsers}</div>
              </div>
              <div style="background: #E6F3FF; color: #0052CC; padding: 8px; border-radius: 6px; font-size: 20px;">📈</div>
            </div>
          </div>
        </div>

        {/* Main Content Grid */}
        <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; margin-bottom: 30px;">
          {/* Role Distribution */}
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">User Roles</h3>
            <div style="space-y: 16px;">
              <For each={stats().roleStats}>
                {(roleStat) => (
                  <div style="margin-bottom: 16px;">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px;">
                      <span style="font-weight: 500; color: #172B4D;">{roleStat.role}</span>
                      <span style="color: #6B778C; font-size: 14px;">{roleStat.count}</span>
                    </div>
                    <div style="width: 100%; height: 6px; background: #F4F5F7; border-radius: 3px; overflow: hidden;">
                      <div 
                        style={`height: 100%; background: ${
                          roleStat.role === 'Admin' ? '#DE350B' : 
                          roleStat.role === 'Manager' ? '#FF8B00' : '#0052CC'
                        }; width: ${roleStat.percentage}%; transition: width 0.3s ease;`}
                      ></div>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>

          {/* Activity Feed */}
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">Recent Activity</h3>
            <div style="max-height: 300px; overflow-y: auto;">
              <For each={activityLog.slice(0, 8)}>
                {(activity) => (
                  <div style="display: flex; align-items: flex-start; padding: 12px 0; border-bottom: 1px solid #F4F5F7;">
                    <div style={`flex-shrink: 0; width: 8px; height: 8px; border-radius: 50%; margin-right: 12px; margin-top: 6px; background: ${
                      activity.action === 'User Created' ? '#006644' :
                      activity.action === 'User Updated' ? '#0052CC' :
                      activity.action === 'User Deleted' || activity.action === 'Bulk Delete' ? '#DE350B' :
                      activity.action === 'Login' ? '#FF8B00' : '#6B778C'
                    };`}></div>
                    <div style="flex: 1; min-width: 0;">
                      <button
                        onclick={() => {
                          const user = users.find(u => u.id === activity.userId);
                          if (user) {
                            setSelectedProfile(user);
                            setShowProfileModal(true);
                            addNotification(`Viewing profile for ${user.name}`, 'info');
                          }
                        }}
                        style="background: none; border: none; font-weight: 500; color: #0052CC; font-size: 14px; margin-bottom: 2px; cursor: pointer; text-align: left; padding: 0; text-decoration: none;"
                      >
                        {getUserName(activity.userId)}
                      </button>
                      <div style="color: #6B778C; font-size: 12px; margin-bottom: 4px;">
                        {activity.action}
                      </div>
                      <div style="color: #8993A4; font-size: 11px;">
                        {formatTimeAgo(activity.timestamp)}
                      </div>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>

          {/* Top Performers */}
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">Top Active Users</h3>
            <div style="space-y: 12px;">
              <For each={users.filter(u => u.status === 'Active').sort((a, b) => b.activityScore - a.activityScore).slice(0, 5)}>
                {(user) => (
                  <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #F4F5F7;">
                    <div>
                      <button
                        onclick={() => {
                          setSelectedProfile(user);
                          setShowProfileModal(true);
                          addNotification(`Viewing profile for ${user.name}`, 'info');
                        }}
                        style="background: none; border: none; font-weight: 500; color: #0052CC; margin-bottom: 2px; font-size: 14px; cursor: pointer; text-align: left; padding: 0;"
                      >
                        {user.name}
                      </button>
                      <div style="font-size: 11px; color: #6B778C;">{user.role} • {user.loginCount} logins</div>
                    </div>
                    <div style="text-align: right;">
                      <div style={`font-weight: 600; font-size: 16px; ${
                        user.activityScore > 90 ? 'color: #006644;' :
                        user.activityScore > 70 ? 'color: #FF8B00;' : 'color: #6B778C;'
                      }`}>
                        {user.activityScore}%
                      </div>
                      <div style="font-size: 10px; color: #8993A4;">activity</div>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>
        </div>

        {/* Enhanced Analytics Grid */}
        <div style="display: grid; grid-template-columns: 2fr 1fr; gap: 20px; margin-bottom: 30px;">
          {/* Department Performance Chart */}
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">Department Performance</h3>
            <div style="space-y: 16px;">
              <For each={stats().deptStats}>
                {(deptStat, index) => {
                  const percentage = Math.round((deptStat.count / stats().total) * 100);
                  const colors = ['#0052CC', '#006644', '#FF8B00', '#DE350B', '#8B5CF6'];
                  return (
                    <div style="margin-bottom: 20px;">
                      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                        <span style="font-weight: 500; color: #172B4D; font-size: 14px;">{deptStat.department}</span>
                        <div style="text-align: right;">
                          <div style="font-size: 16px; font-weight: 600; color: #172B4D;">{deptStat.count}</div>
                          <div style="font-size: 11px; color: #6B778C;">{percentage}% of total</div>
                        </div>
                      </div>
                      <div style="width: 100%; height: 8px; background: #F4F5F7; border-radius: 4px; overflow: hidden; position: relative;">
                        <div 
                          style={`height: 100%; background: linear-gradient(90deg, ${colors[index() % colors.length]}, ${colors[index() % colors.length]}99); width: ${percentage}%; transition: width 0.8s ease-in-out; border-radius: 4px;`}
                        ></div>
                      </div>
                    </div>
                  );
                }}
              </For>
            </div>
          </div>

          {/* User Activity Heatmap */}
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">Activity Heatmap</h3>
            <div style="display: grid; grid-template-columns: repeat(7, 1fr); gap: 4px; margin-bottom: 16px;">
              <For each={Array.from({length: 28}, (_, i) => i)}>
                {(day) => {
                  const activity = Math.random() * 100;
                  const intensity = activity > 75 ? 1 : activity > 50 ? 0.7 : activity > 25 ? 0.4 : 0.1;
                  return (
                    <div 
                      style={`width: 24px; height: 24px; border-radius: 3px; background: rgba(0, 82, 204, ${intensity}); border: 1px solid #E1E5E9;`}
                      title={`Day ${day + 1}: ${Math.round(activity)}% activity`}
                    ></div>
                  );
                }}
              </For>
            </div>
            <div style="display: flex; justify-content: space-between; font-size: 11px; color: #6B778C;">
              <span>Less</span>
              <span>More</span>
            </div>
            <div style="margin-top: 16px; padding-top: 16px; border-top: 1px solid #F4F5F7;">
              <div style="font-size: 12px; color: #6B778C; text-align: center;">
                Last 28 days activity pattern
              </div>
            </div>
          </div>
        </div>

        {/* Performance Metrics Cards */}
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; margin-bottom: 30px;">
          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h4 style="color: #172B4D; font-size: 16px; margin: 0 0 16px 0; font-weight: 600;">Login Frequency</h4>
            <div style="display: flex; align-items: end; height: 60px; gap: 6px;">
              <For each={users.slice(0, 8)}>
                {(user) => (
                  <div style="flex: 1; display: flex; flex-direction: column; align-items: center;">
                    <div 
                      style={`width: 100%; background: linear-gradient(to top, #0052CC, #4A90E2); border-radius: 2px 2px 0 0; height: ${(user.loginCount / 250) * 50}px; min-height: 2px; transition: height 0.6s ease;`}
                      title={`${user.name}: ${user.loginCount} logins`}
                    ></div>
                    <div style="font-size: 9px; color: #6B778C; margin-top: 4px; writing-mode: vertical-rl; text-orientation: mixed;">
                      {user.name.split(' ')[0]}
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>

          <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
            <h4 style="color: #172B4D; font-size: 16px; margin: 0 0 16px 0; font-weight: 600;">Activity Trends</h4>
            <div style="position: relative; height: 60px;">
              <svg width="100%" height="60" style="overflow: visible;">
                <For each={[95, 87, 92, 78, 85, 91, 88, 94]}>
                  {(point, index) => (
                    <g>
                      <circle 
                        cx={`${(index() * 100) / 7}%`} 
                        cy={`${60 - (point * 0.6)}px`} 
                        r="3" 
                        fill="#00B8A9" 
                        stroke="white" 
                        stroke-width="2"
                      />
                      {index() < 7 && (
                        <line 
                          x1={`${(index() * 100) / 7}%`} 
                          y1={`${60 - (point * 0.6)}px`} 
                          x2={`${((index() + 1) * 100) / 7}%`} 
                          y2={`${60 - ([87, 92, 78, 85, 91, 88, 94][index()] * 0.6)}px`} 
                          stroke="#00B8A9" 
                          stroke-width="2"
                        />
                      )}
                    </g>
                  )}
                </For>
              </svg>
            </div>
            <div style="display: flex; justify-content: space-between; font-size: 10px; color: #6B778C; margin-top: 8px;">
              <span>7d ago</span>
              <span>Today</span>
            </div>
          </div>
        </div>

        {/* Quick Actions */}
        <div style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
          <h3 style="color: #172B4D; font-size: 18px; margin: 0 0 20px 0; font-weight: 600;">Quick Actions</h3>
          <div style="display: flex; gap: 12px; flex-wrap: wrap;">
            <a 
              href="/users" 
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #0052CC; color: white; text-decoration: none; border-radius: 6px; font-weight: 500; transition: all 0.2s;"
            >
              👥 Manage Users
            </a>
            <button 
              onclick={() => {
                setShowCreateModal(true);
                addNotification('Create user modal opened', 'info');
              }}
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #006644; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
            >
              ➕ Add New User
            </button>
            <button 
              onclick={() => {
                window.location.reload();
                addNotification('Dashboard data refreshed', 'info');
              }}
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #FF8B00; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
            >
              🔄 Refresh Data
            </button>
            <button 
              onclick={() => {
                addNotification('Welcome to UManager Enterprise! Your comprehensive user management solution is ready.', 'info', 10000);
                addNotification(`You have ${stats().total} total users with ${stats().active} currently active.`, 'info', 8000);
              }}
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #0052CC; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
            >
              💡 System Status
            </button>
            <button 
              onclick={() => {
                setShowAdminPanel(true);
                addNotification('Admin settings panel opened', 'info');
              }}
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #8B5CF6; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
            >
              ⚙️ Admin Settings
            </button>
            <button 
              onclick={() => {
                setShowPerformanceMonitor(true);
                addNotification('Performance monitor opened', 'info');
              }}
              style="display: inline-flex; align-items: center; padding: 12px 20px; background: #059669; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s;"
            >
              📊 Performance
            </button>
            <button 
              onclick={() => {
                realtimeActive() ? stopRealtimeSimulation() : startRealtimeSimulation();
              }}
              style={`display: inline-flex; align-items: center; padding: 12px 20px; background: ${realtimeActive() ? '#10B981' : '#6B7280'}; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer; transition: all 0.2s; position: relative;`}
            >
              {realtimeActive() ? (
                <>
                  <span style="display: inline-block; width: 8px; height: 8px; background: #34D399; border-radius: 50%; margin-right: 8px; animation: pulse 2s infinite;"></span>
                  ⏸️ Stop Live Demo
                </>
              ) : (
                '▶️ Start Live Demo'
              )}
            </button>
          </div>
        </div>

        {/* Navigation Bar */}
        <div style="margin-top: 30px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
          <div style="display: flex; justify-content: center; gap: 20px; align-items: center;">
            <a href="/dashboard" style="color: #0052CC; text-decoration: none; padding: 12px 20px; border-radius: 6px; font-weight: 500; transition: all 0.2s; background: rgba(0, 82, 204, 0.1); border: 2px solid #0052CC;">
              📊 Dashboard
            </a>
            <a href="/users" style="color: #0052CC; text-decoration: none; padding: 12px 20px; border-radius: 6px; font-weight: 500; transition: all 0.2s; border: 2px solid transparent;">
              👥 Users Management
            </a>
            <a href="/login" style="color: #6B778C; text-decoration: none; padding: 12px 20px; border-radius: 6px; font-weight: 500; transition: all 0.2s; border: 2px solid transparent;">
              🔓 Login Page
            </a>
          </div>
        </div>
      </div>
      
      <CreateModal />
      <UserProfileModal />
      <PerformanceMonitorPanel />
      <AdminSettingsPanel />
      <KeyboardShortcutsPanel />
      <ThemeToggle />
      <NotificationCenter />
    </div>
  );
};

const App = () => {
  return (
    <Router>
      <Route path="/login" component={LoginPage} />
      <Route path="/users" component={UsersPage} />
      <Route path="/dashboard" component={Dashboard} />
      <Route path="/" component={() => <Dashboard />} />
    </Router>
  );
};

export default App;