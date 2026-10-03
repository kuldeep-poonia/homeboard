export type ItemType = 'reminder' | 'shopping' | 'event' | 'movie' | 'status';

export interface Item {
  id: string;
  board_id: string;
  type: ItemType;
  text: string;
  when_ts?: string;
  done: boolean;
  archived: boolean;
  created_by_device?: string;
  created_at: string;
  updated_at: string;
}

export interface BoardSession {
  board_id: string;
  tv_secret: string;
}

export interface JoinTokenResponse {
  token: string;
  url: string;
  expires_at: string;
}

export interface ConnectedDevice {
  id: string;
  label: string;
  created_at: string;
  last_seen: string;
}

export type CategoryKey = 'today' | 'upcoming' | 'buy' | 'family';

export interface CategoryInfo {
  key: CategoryKey;
  title: string;
  badge: string;
  types: readonly ItemType[];
}
