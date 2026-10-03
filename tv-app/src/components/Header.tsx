import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { THEME } from '../config/constants';

interface HeaderProps {
  syncStatus: 'connected' | 'reconnecting' | 'disconnected';
  boardId: string;
}

export const Header: React.FC<HeaderProps> = ({ syncStatus, boardId }) => {
  const [currentTime, setCurrentTime] = useState<string>('');
  const [currentDate, setCurrentDate] = useState<string>('');

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setCurrentTime(
        now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      );
      setCurrentDate(
        now.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
      );
    };

    updateTime();
    const timer = setInterval(updateTime, 1000);
    return () => clearInterval(timer);
  }, []);

  const getStatusColor = () => {
    switch (syncStatus) {
      case 'connected':
        return THEME.accentSuccess;
      case 'reconnecting':
        return THEME.accentOrange;
      default:
        return THEME.danger;
    }
  };

  const getStatusText = () => {
    switch (syncStatus) {
      case 'connected':
        return 'LIVE SYNC';
      case 'reconnecting':
        return 'CONNECTING...';
      default:
        return 'OFFLINE';
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.left}>
        <Text style={styles.logo}>📺 HomeBoard</Text>
        <View style={styles.badgeContainer}>
          <View style={[styles.statusDot, { backgroundColor: getStatusColor() }]} />
          <Text style={[styles.statusText, { color: getStatusColor() }]}>{getStatusText()}</Text>
        </View>
        <Text style={styles.roomTag}>Living Room</Text>
      </View>

      <View style={styles.right}>
        <Text style={styles.clockTime}>{currentTime}</Text>
        <Text style={styles.clockDate}>{currentDate}</Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 28,
    borderBottomWidth: 1,
    borderBottomColor: THEME.cardBorder,
  },
  left: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  logo: {
    fontSize: 28,
    fontWeight: '800',
    color: '#ffffff',
    letterSpacing: 0.5,
  },
  badgeContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 4,
    paddingHorizontal: 10,
    borderRadius: 14,
    backgroundColor: '#1b232e',
    gap: 6,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  roomTag: {
    fontSize: 14,
    color: THEME.textMuted,
    fontWeight: '500',
  },
  right: {
    alignItems: 'flex-end',
  },
  clockTime: {
    fontSize: 26,
    fontWeight: '700',
    color: '#ffffff',
  },
  clockDate: {
    fontSize: 13,
    color: THEME.textMuted,
    fontWeight: '500',
  },
});
