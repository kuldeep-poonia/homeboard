import { CONFIG } from '../config/constants';
import { BoardSession, Item, JoinTokenResponse } from '../types';

export class HomeBoardAPI {
  private static baseUrl = CONFIG.API_BASE_URL;

  public static setBaseUrl(url: string) {
    this.baseUrl = url.replace(/\/+$/, '');
  }

  public static getQRImageUrl(token: string): string {
    return `${this.baseUrl}/qr/${token}.png`;
  }

  public static async createBoard(): Promise<BoardSession> {
    const res = await fetch(`${this.baseUrl}/v1/boards`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
    if (!res.ok) {
      throw new Error(`Failed to create board: HTTP ${res.status}`);
    }
    return await res.json();
  }

  public static async getJoinToken(boardId: string, tvSecret: string): Promise<JoinTokenResponse> {
    const res = await fetch(`${this.baseUrl}/v1/boards/${boardId}/join-tokens`, {
      method: 'POST',
      headers: {
        'X-TV-Secret': tvSecret,
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      throw new Error(`Failed to fetch join token: HTTP ${res.status}`);
    }
    return await res.json();
  }

  public static async getItems(boardId: string, tvSecret: string): Promise<Item[]> {
    const res = await fetch(`${this.baseUrl}/v1/boards/${boardId}/items`, {
      method: 'GET',
      headers: {
        'X-TV-Secret': tvSecret,
      },
    });
    if (!res.ok) {
      throw new Error(`Failed to fetch items: HTTP ${res.status}`);
    }
    return await res.json();
  }

  public static async toggleItemDone(boardId: string, tvSecret: string, itemId: string, done: boolean): Promise<Item> {
    const res = await fetch(`${this.baseUrl}/v1/boards/${boardId}/items/${itemId}`, {
      method: 'PATCH',
      headers: {
        'X-TV-Secret': tvSecret,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ done }),
    });
    if (!res.ok) {
      throw new Error(`Failed to update item: HTTP ${res.status}`);
    }
    return await res.json();
  }

  public static async archiveItem(boardId: string, tvSecret: string, itemId: string): Promise<Item> {
    const res = await fetch(`${this.baseUrl}/v1/boards/${boardId}/items/${itemId}`, {
      method: 'PATCH',
      headers: {
        'X-TV-Secret': tvSecret,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ archived: true }),
    });
    if (!res.ok) {
      throw new Error(`Failed to archive item: HTTP ${res.status}`);
    }
    return await res.json();
  }
}
