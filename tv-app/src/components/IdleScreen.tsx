import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { THEME } from '../config/constants';

interface IdleScreenProps {
  pendingCount: number;
  onWake: () => void;
}

export const IdleScreen: React.FC<IdleScreenProps> = ({ pendingCount, onWake }) => {
  const [timeStr, setTimeStr] = useState<string>('');
  const [dateStr, setDateStr] = useState<string>('');
  const [offsetY, setOffsetY] = useState<number>(0);

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setTimeStr(
        now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      );
      setDateStr(
        now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' })
      );
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);

    // Subtle drift every 30 seconds to prevent OLED screen burn-in
    const driftInterval = setInterval(() => {
      setOffsetY((prev) => (prev === 0 ? 12 : prev === 12 ? -12 : 0));
    }, 30000);

    return () => {
      clearInterval(interval);
      clearInterval(driftInterval);
    };
  }, []);

  return (
    <View style={styles.overlay} onTouchStart={onWake}>
      <View style={[styles.content, { transform: [{ translateY: offsetY }] }]}>
        <Text style={styles.clock}>{timeStr}</Text>
        <Text style={styles.date}>{dateStr}</Text>
        <View style={styles.summaryBadge}>
          <Text style={styles.summaryText}>
            {pendingCount === 0 ? 'All caught up' : `${pendingCount} items on HomeBoard`}
          </Text>
        </View>
        <Text style={styles.wakeHint}>Press any remote button to wake</Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  overlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(5, 7, 10, 0.92)',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 999,
  },
  content: {
    alignItems: 'center',
  },
  clock: {
    fontSize: 96,
    fontWeight: '800',
    color: '#ffffff',
    letterSpacing: -1,
  },
  date: {
    fontSize: 26,
    color: THEME.textMuted,
    marginTop: 8,
    fontWeight: '500',
  },
  summaryBadge: {
    marginTop: 24,
    paddingHorizontal: 18,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: '#161b22',
    borderWidth: 1,
    borderColor: THEME.cardBorder,
  },
  summaryText: {
    fontSize: 16,
    color: THEME.accentOrange,
    fontWeight: '600',
  },
  wakeHint: {
    fontSize: 14,
    color: '#484f58',
    marginTop: 40,
  },
});
