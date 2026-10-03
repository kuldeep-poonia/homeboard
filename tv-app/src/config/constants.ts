export const CONFIG = {
  // Local backend server address; use 10.0.2.2 for Android simulator or localhost for web/desktop
  API_BASE_URL: process.env.API_BASE_URL || 'http://localhost:8080',
  QR_REFRESH_INTERVAL_MS: 8 * 60 * 1000, // 8 minutes (prior to 10 min TTL)
  POLLING_FALLBACK_MS: 10 * 1000,         // 10s fallback if WebSocket disconnects
  IDLE_TIMEOUT_MS: 3 * 60 * 1000,        // 3 minutes of inactivity triggers ambient idle
  CATEGORIES: [
    { key: 'today', title: 'Today', badge: 'Active', types: ['reminder', 'status'] },
    { key: 'upcoming', title: 'Upcoming', badge: 'Events', types: ['event'] },
    { key: 'buy', title: 'Buy', badge: 'List', types: ['shopping'] },
    { key: 'family', title: 'Family & Movies', badge: 'Watch', types: ['movie'] },
  ] as const,
};

export const THEME = {
  bg: '#0d1117',
  cardBg: '#161b22',
  cardBorder: '#30363d',
  cardBorderFocused: '#FF9900', // Amazon Fire TV Focus Orange
  cardBorderSelected: '#58a6ff',
  text: '#ffffff',
  textMuted: '#8b949e',
  accentSuccess: '#238636',
  accentBlue: '#1f6feb',
  accentOrange: '#FF9900',
  danger: '#da3633',
};
