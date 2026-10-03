import React from 'react';
import { FlatList, StyleSheet, Text, View } from 'react-native';
import { THEME } from '../config/constants';
import { CategoryInfo, Item } from '../types';
import { ItemRow } from './ItemRow';

interface BoardCardProps {
  category: CategoryInfo;
  items: Item[];
  focusedItemId: string | null;
  onItemPress: (item: Item) => void;
  onItemLongPress: (item: Item) => void;
}

export const BoardCard: React.FC<BoardCardProps> = ({
  category,
  items,
  focusedItemId,
  onItemPress,
  onItemLongPress,
}) => {
  return (
    <View style={styles.card}>
      <View style={styles.header}>
        <View style={styles.titleRow}>
          <Text style={styles.title}>{category.title}</Text>
          <View style={styles.countBadge}>
            <Text style={styles.countText}>{items.length}</Text>
          </View>
        </View>
        <Text style={styles.badge}>{category.badge}</Text>
      </View>

      {items.length === 0 ? (
        <View style={styles.emptyContainer}>
          <Text style={styles.emptyEmoji}>📋</Text>
          <Text style={styles.emptyText}>Empty</Text>
          <Text style={styles.emptySubtext}>Scan QR to add</Text>
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => (
            <ItemRow
              item={item}
              isFocused={focusedItemId === item.id}
              onPress={() => onItemPress(item)}
              onLongPress={() => onItemLongPress(item)}
            />
          )}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={styles.listContent}
        />
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  card: {
    flex: 1,
    backgroundColor: THEME.cardBg,
    borderColor: THEME.cardBorder,
    borderWidth: 1,
    borderRadius: 16,
    padding: 18,
    marginHorizontal: 8,
    height: '100%',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 4,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 16,
    paddingBottom: 10,
    borderBottomWidth: 1,
    borderBottomColor: '#21262d',
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  title: {
    fontSize: 22,
    fontWeight: '700',
    color: '#ffffff',
  },
  countBadge: {
    backgroundColor: '#21262d',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
  },
  countText: {
    fontSize: 12,
    fontWeight: '700',
    color: THEME.textMuted,
  },
  badge: {
    fontSize: 11,
    fontWeight: '700',
    color: THEME.accentBlue,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  listContent: {
    paddingBottom: 12,
  },
  emptyContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 32,
  },
  emptyEmoji: {
    fontSize: 32,
    marginBottom: 8,
    opacity: 0.6,
  },
  emptyText: {
    fontSize: 16,
    color: THEME.textMuted,
    fontWeight: '600',
  },
  emptySubtext: {
    fontSize: 13,
    color: '#484f58',
    marginTop: 4,
  },
});
