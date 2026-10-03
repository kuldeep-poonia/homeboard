import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  BackHandler,
  SafeAreaView,
  StatusBar,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { BoardCard } from './src/components/BoardCard';
import { Header } from './src/components/Header';
import { IdleScreen } from './src/components/IdleScreen';
import { PersistentQR } from './src/components/PersistentQR';
import { CONFIG, THEME } from './src/config/constants';
import { HomeBoardAPI } from './src/services/api';
import { TVWebSocketClient } from './src/services/websocket';
import { BoardSession, CategoryInfo, CategoryKey, Item } from './src/types';

export default function App() {
  const [boardSession, setBoardSession] = useState<BoardSession | null>(null);
  const [items, setItems] = useState<Item[]>([]);
  const [qrToken, setQrToken] = useState<string | null>(null);
  const [qrSecondsLeft, setQrSecondsLeft] = useState<number>(480);
  const [syncStatus, setSyncStatus] = useState<'connected' | 'reconnecting' | 'disconnected'>('disconnected');
  const [focusedItemId, setFocusedItemId] = useState<string | null>(null);
  const [isIdle, setIsIdle] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const wsClientRef = useRef<TVWebSocketClient | null>(null);
  const idleTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Reset idle timer upon any remote activity
  const reportRemoteActivity = useCallback(() => {
    if (isIdle) {
      setIsIdle(false);
    }
    if (idleTimerRef.current) clearTimeout(idleTimerRef.current);
    idleTimerRef.current = setTimeout(() => {
      setIsIdle(true);
    }, CONFIG.IDLE_TIMEOUT_MS);
  }, [isIdle]);

  // Initialize Board
  useEffect(() => {
    let isMounted = true;

    async function init() {
      try {
        const session = await HomeBoardAPI.createBoard();
        if (!isMounted) return;
        setBoardSession(session);
        setErrorMessage(null);
      } catch (err: unknown) {
        if (!isMounted) return;
        const msg = err instanceof Error ? err.message : 'Failed to connect to backend';
        setErrorMessage(msg);
        setSyncStatus('disconnected');
      }
    }

    init();
    reportRemoteActivity();

    const backListener = BackHandler.addEventListener('hardwareBackPress', () => {
      if (isIdle) {
        setIsIdle(false);
        reportRemoteActivity();
        return true;
      }
      return false;
    });

    return () => {
      isMounted = false;
      backListener.remove();
      if (idleTimerRef.current) clearTimeout(idleTimerRef.current);
    };
  }, []);

  // Fetch Items & Connect WebSocket once session is established
  useEffect(() => {
    if (!boardSession) return;

    let isMounted = true;

    async function loadItems() {
      try {
        if (!boardSession) return;
        const list = await HomeBoardAPI.getItems(boardSession.board_id, boardSession.tv_secret);
        if (isMounted) setItems(list);
      } catch (e) {
        // preserve existing items on temporary network drop
      }
    }

    loadItems();

    // Start live WebSocket sync
    const ws = new TVWebSocketClient(
      boardSession.board_id,
      boardSession.tv_secret,
      (type, payload) => {
        reportRemoteActivity();
        if (type === 'item.created') {
          const newItem = payload as Item;
          setItems((prev) => [newItem, ...prev.filter((it) => it.id !== newItem.id)]);
        } else if (type === 'item.updated') {
          const updatedItem = payload as Item;
          setItems((prev) =>
            updatedItem.archived
              ? prev.filter((it) => it.id !== updatedItem.id)
              : prev.map((it) => (it.id === updatedItem.id ? updatedItem : it))
          );
        } else if (type === 'item.deleted') {
          const deleted = payload as { id: string };
          setItems((prev) => prev.filter((it) => it.id !== deleted.id));
        }
      },
      (status) => {
        setSyncStatus(status);
        if (status === 'connected') {
          loadItems(); // Resync state upon reconnect
        }
      }
    );

    ws.connect();
    wsClientRef.current = ws;

    return () => {
      isMounted = false;
      ws.disconnect();
    };
  }, [boardSession]);

  // Fetch & Auto-Refresh Join Token for Corner QR
  useEffect(() => {
    if (!boardSession) return;

    let isMounted = true;

    async function fetchToken() {
      try {
        if (!boardSession) return;
        const res = await HomeBoardAPI.getJoinToken(boardSession.board_id, boardSession.tv_secret);
        if (isMounted) {
          setQrToken(res.token);
          setQrSecondsLeft(480);
        }
      } catch (e) {
        console.error('Failed fetching QR token', e);
      }
    }

    fetchToken();
    const tokenInterval = setInterval(fetchToken, CONFIG.QR_REFRESH_INTERVAL_MS);

    const countdownTimer = setInterval(() => {
      setQrSecondsLeft((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);

    return () => {
      isMounted = false;
      clearInterval(tokenInterval);
      clearInterval(countdownTimer);
    };
  }, [boardSession]);

  const handleItemPress = async (item: Item) => {
    reportRemoteActivity();
    if (!boardSession) return;
    try {
      setFocusedItemId(item.id);
      await HomeBoardAPI.toggleItemDone(
        boardSession.board_id,
        boardSession.tv_secret,
        item.id,
        !item.done
      );
    } catch (e) {
      console.error('Failed to toggle item', e);
    }
  };

  const handleItemLongPress = async (item: Item) => {
    reportRemoteActivity();
    if (!boardSession) return;
    try {
      await HomeBoardAPI.archiveItem(
        boardSession.board_id,
        boardSession.tv_secret,
        item.id
      );
    } catch (e) {
      console.error('Failed to archive item', e);
    }
  };

  const filterItemsForCategory = (cat: CategoryInfo) => {
    return items.filter((it) => cat.types.includes(it.type) && !it.archived);
  };

  const pendingCount = items.filter((it) => !it.done && !it.archived).length;

  return (
    <SafeAreaView style={styles.safeArea}>
      <StatusBar barStyle="light-content" hidden={true} />

      {/* Ambient Screensaver / Idle View */}
      {isIdle && (
        <IdleScreen
          pendingCount={pendingCount}
          onWake={reportRemoteActivity}
        />
      )}

      {/* 10-Foot Television Top Navigation Header */}
      <Header
        syncStatus={syncStatus}
        boardId={boardSession?.board_id || 'Initializing...'}
      />

      {/* Error or Offline Banner */}
      {errorMessage && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>⚠️ {errorMessage}</Text>
        </View>
      )}

      {/* Main 10-Foot 4-Card Dashboard Area */}
      <View style={styles.mainContent}>
        <View style={styles.cardsRow}>
          {CONFIG.CATEGORIES.map((cat) => (
            <BoardCard
              key={cat.key}
              category={cat}
              items={filterItemsForCategory(cat)}
              focusedItemId={focusedItemId}
              onItemPress={handleItemPress}
              onItemLongPress={handleItemLongPress}
            />
          ))}
        </View>

        {/* Bottom Bar: Ambient Remote Shortcuts & Persistent QR */}
        <View style={styles.bottomBar}>
          <View style={styles.remoteHints}>
            <View style={styles.hintTag}>
              <Text style={styles.hintKey}>[ D-PAD ]</Text>
              <Text style={styles.hintAction}>Navigate</Text>
            </View>
            <View style={styles.hintTag}>
              <Text style={styles.hintKey}>[ SELECT / OK ]</Text>
              <Text style={styles.hintAction}>Toggle Done</Text>
            </View>
            <View style={styles.hintTag}>
              <Text style={styles.hintKey}>[ HOLD OK ]</Text>
              <Text style={styles.hintAction}>Archive</Text>
            </View>
          </View>

          <PersistentQR token={qrToken} refreshSeconds={qrSecondsLeft} />
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: THEME.bg,
  },
  mainContent: {
    flex: 1,
    paddingHorizontal: 24,
    paddingTop: 16,
    paddingBottom: 20,
    justifyContent: 'space-between',
  },
  cardsRow: {
    flex: 1,
    flexDirection: 'row',
    marginBottom: 16,
  },
  bottomBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: THEME.cardBorder,
  },
  remoteHints: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  hintTag: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    backgroundColor: '#161b22',
    paddingVertical: 6,
    paddingHorizontal: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#21262d',
  },
  hintKey: {
    fontSize: 12,
    fontWeight: '800',
    color: THEME.accentOrange,
  },
  hintAction: {
    fontSize: 13,
    color: THEME.textMuted,
    fontWeight: '500',
  },
  errorBanner: {
    backgroundColor: 'rgba(218, 54, 51, 0.2)',
    borderColor: THEME.danger,
    borderWidth: 1,
    paddingVertical: 8,
    paddingHorizontal: 24,
    alignItems: 'center',
  },
  errorText: {
    color: '#f85149',
    fontSize: 14,
    fontWeight: '600',
  },
});
