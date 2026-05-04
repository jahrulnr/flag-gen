import { Injectable, signal } from '@angular/core';

import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class RealtimeService {
  readonly lastMessage = signal<string | null>(null);
  private socket: WebSocket | null = null;

  connect(projectId: string, environmentKey: string): void {
    if (!environment.broadcastDevToken || !projectId || !environmentKey || this.socket) {
      return;
    }
    const apiURL = new URL(environment.apiBaseUrl, globalThis.location?.origin ?? 'http://localhost:6123');
    const wsProtocol = apiURL.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${wsProtocol}//${apiURL.host}/api/broadcast/flags?projectId=${encodeURIComponent(projectId)}&environmentKey=${encodeURIComponent(environmentKey)}&token=${encodeURIComponent(environment.broadcastDevToken)}`;
    this.socket = new WebSocket(url);
    this.socket.onmessage = (event) => this.lastMessage.set(event.data);
    this.socket.onclose = () => {
      this.socket = null;
    };
  }

  disconnect(): void {
    this.socket?.close();
    this.socket = null;
  }
}
