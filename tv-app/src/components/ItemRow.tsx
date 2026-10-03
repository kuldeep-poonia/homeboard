import React from 'react';
import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { THEME } from '../config/constants';
import { Item } from '../types';

interface ItemRowProps {
  item: Item;
  isFocused: boolean;
  onPress: () => void;
  onLongPress: () => void;
}

export const ItemRow: React.FC<ItemRowProps> = ({
  item,
  isFocused,
  onPress,
  onLongPress,
}) => {
  return (
    <TouchableOpacity
      activeOpacity={0.85}
      onPress={onPress}
      onLongPress={onLongPress}
      style={[
        styles.container,
        item.done && styles.containerDone,
        isFocused && styles.containerFocused,
      ]}
    >
      <View style={styles.left}>
        <View style={[styles.checkbox, item.done && styles.checkboxDone]}>
          {item.done && <Text style={styles.checkIcon}>✓</Text>}
        </View>
        <Text
          style={[
            styles.text,
            item.done && styles.textDone,
            isFocused && styles.textFocused,
          ]}
          numberOfLines={2}
        >
          {item.text}
        </Text>
      </View>

      <View style={styles.right}>
        <View style={styles.typeBadge}>
          <Text style={styles.typeBadgeText}>{item.type}</Text>
        </View>
      </View>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderRadius: 10,
    backgroundColor: '#1f242c',
    marginVertical: 4,
    borderWidth: 2,
    borderColor: 'transparent',
  },
  containerDone: {
    opacity: 0.5,
    backgroundColor: '#14181f',
  },
  containerFocused: {
    borderColor: THEME.cardBorderFocused,
    backgroundColor: '#27303d',
    transform: [{ scale: 1.02 }],
  },
  left: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
    gap: 12,
  },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 6,
    borderWidth: 2,
    borderColor: THEME.textMuted,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkboxDone: {
    backgroundColor: THEME.accentSuccess,
    borderColor: THEME.accentSuccess,
  },
  checkIcon: {
    color: '#ffffff',
    fontSize: 14,
    fontWeight: 'bold',
  },
  text: {
    fontSize: 18,
    color: '#ffffff',
    flex: 1,
    lineHeight: 24,
  },
  textDone: {
    textDecorationLine: 'line-through',
    color: THEME.textMuted,
  },
  textFocused: {
    color: '#ffffff',
    fontWeight: '600',
  },
  right: {
    marginLeft: 8,
  },
  typeBadge: {
    paddingVertical: 2,
    paddingHorizontal: 8,
    borderRadius: 4,
    backgroundColor: '#2d333b',
  },
  typeBadgeText: {
    fontSize: 11,
    color: THEME.textMuted,
    textTransform: 'uppercase',
    fontWeight: '700',
  },
});
