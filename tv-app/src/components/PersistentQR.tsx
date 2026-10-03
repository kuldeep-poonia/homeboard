import React from 'react';
import { Image, StyleSheet, Text, View } from 'react-native';
import { THEME } from '../config/constants';
import { HomeBoardAPI } from '../services/api';

interface PersistentQRProps {
  token: string | null;
  refreshSeconds: number;
}

export const PersistentQR: React.FC<PersistentQRProps> = ({ token, refreshSeconds }) => {
  if (!token) {
    return (
      <View style={styles.container}>
        <View style={styles.qrPlaceholder}>
          <Text style={styles.loadingText}>Generating QR...</Text>
        </View>
      </View>
    );
  }

  const qrImageUrl = HomeBoardAPI.getQRImageUrl(token);

  return (
    <View style={styles.container}>
      <View style={styles.qrFrame}>
        <Image
          source={{ uri: qrImageUrl }}
          style={styles.qrImage}
          resizeMode="contain"
        />
      </View>
      <View style={styles.infoCol}>
        <Text style={styles.qrTitle}>Instant Phone Sync</Text>
        <Text style={styles.qrSubtitle}>Point phone camera to scan</Text>
        <Text style={styles.timerText}>
          Rotates in {Math.floor(refreshSeconds / 60)}m {refreshSeconds % 60}s
        </Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#161b22',
    borderColor: THEME.cardBorder,
    borderWidth: 1,
    borderRadius: 14,
    padding: 10,
    gap: 12,
  },
  qrFrame: {
    width: 84,
    height: 84,
    backgroundColor: '#ffffff',
    borderRadius: 8,
    padding: 4,
    alignItems: 'center',
    justifyContent: 'center',
  },
  qrImage: {
    width: 76,
    height: 76,
  },
  qrPlaceholder: {
    width: 84,
    height: 84,
    backgroundColor: '#21262d',
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
  },
  loadingText: {
    fontSize: 10,
    color: THEME.textMuted,
    textAlign: 'center',
  },
  infoCol: {
    justifyContent: 'center',
    maxWidth: 160,
  },
  qrTitle: {
    fontSize: 14,
    fontWeight: '700',
    color: '#ffffff',
  },
  qrSubtitle: {
    fontSize: 12,
    color: THEME.textMuted,
    marginTop: 2,
  },
  timerText: {
    fontSize: 10,
    color: THEME.accentOrange,
    fontWeight: '600',
    marginTop: 4,
  },
});
