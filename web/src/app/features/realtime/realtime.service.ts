import { Injectable, signal } from '@angular/core';

import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class RealtimeService {
  readonly lastMessage = signal<string | null>(null);
  readonly lastEvent = signal<RealtimeEventSummary | null>(null);
  private socket: WebSocket | null = null;

  connect(projectId: string, environmentKey: string): void {
    if (!environment.broadcastDevToken || !projectId || !environmentKey || this.socket) {
      return;
    }
    const apiURL = new URL(environment.apiBaseUrl, globalThis.location?.origin ?? 'http://localhost:6123');
    const wsProtocol = apiURL.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${wsProtocol}//${apiURL.host}/api/broadcast/flags?projectId=${encodeURIComponent(projectId)}&environmentKey=${encodeURIComponent(environmentKey)}&token=${encodeURIComponent(environment.broadcastDevToken)}`;
    this.socket = new WebSocket(url);
    this.socket.onmessage = (event) => {
      this.lastMessage.set(event.data);
      this.lastEvent.set(summarizeRealtimeEvent(event.data));
    };
    this.socket.onclose = () => {
      this.socket = null;
    };
  }

  disconnect(): void {
    this.socket?.close();
    this.socket = null;
  }
}

export interface RealtimeEventSummary {
  type: string;
  flagKey?: string;
  version?: number;
  scopeVersion?: number;
  environmentKey?: string;
}

function summarizeRealtimeEvent(raw: string): RealtimeEventSummary | null {
  try {
    const data = JSON.parse(raw) as Record<string, unknown>;
    return {
      type: stringValue(data['type']) || 'message',
      flagKey: stringValue(data['flagKey']),
      version: numberValue(data['version'] ?? data['aggregateVersion']),
      scopeVersion: numberValue(data['scopeVersion']),
      environmentKey: stringValue(data['environmentKey']),
    };
  } catch {
    return null;
  }
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function numberValue(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}
