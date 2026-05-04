import { isPlatformBrowser } from '@angular/common';
import { DestroyRef, inject, Injectable, PLATFORM_ID, signal } from '@angular/core';

import { environment } from '../../environments/environment';

export type ConnectivityState = 'connected' | 'reconnecting' | 'disconnected';

/** Browser reachability: online/offline + lightweight ping to API /health (not under /api). */
@Injectable({ providedIn: 'root' })
export class ConnectivityService {
  private readonly platformId = inject(PLATFORM_ID);
  readonly state = signal<ConnectivityState>('connected');

  private pingTimer: number | null = null;
  private reconnectTimer: number | null = null;
  private inFlight = false;
  private boundOffline?: () => void;
  private boundOnline?: () => void;

  /** Attach listeners; cleans up when destroyRef fires (e.g. shell destroyed). */
  attach(destroyRef: DestroyRef): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }
    const cleanup = (): void => {
      if (this.boundOffline) {
        window.removeEventListener('offline', this.boundOffline);
      }
      if (this.boundOnline) {
        window.removeEventListener('online', this.boundOnline);
      }
      if (this.pingTimer != null) {
        clearInterval(this.pingTimer);
        this.pingTimer = null;
      }
      this.clearReconnectTimer();
    };
    destroyRef.onDestroy(cleanup);

    this.boundOffline = (): void => {
      this.clearReconnectTimer();
      this.state.set('disconnected');
    };
    this.boundOnline = (): void => {
      this.state.set('reconnecting');
      void this.pingOnce();
    };
    window.addEventListener('offline', this.boundOffline);
    window.addEventListener('online', this.boundOnline);

    this.applyNavigatorOnline();
    this.pingTimer = window.setInterval(() => void this.pingOnce(), 25_000);
    void this.pingOnce();
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer != null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private applyNavigatorOnline(): void {
    if (!navigator.onLine) {
      this.state.set('disconnected');
    }
  }

  private healthUrl(): string {
    const base = environment.apiBaseUrl.replace(/\/api\/?$/, '');
    return `${base}/health`;
  }

  private async pingOnce(): Promise<void> {
    if (!isPlatformBrowser(this.platformId) || this.inFlight) {
      return;
    }
    if (!navigator.onLine) {
      this.state.set('disconnected');
      return;
    }
    this.inFlight = true;
    const prev = this.state();
    try {
      const ctrl = new AbortController();
      const t = window.setTimeout(() => ctrl.abort(), 5_000);
      const res = await fetch(this.healthUrl(), { method: 'GET', signal: ctrl.signal, cache: 'no-store' });
      clearTimeout(t);
      if (res.ok) {
        this.clearReconnectTimer();
        this.state.set('connected');
      } else {
        this.markDegradedFrom(prev);
      }
    } catch {
      this.markDegradedFrom(prev);
    } finally {
      this.inFlight = false;
    }
  }

  private markDegradedFrom(prev: ConnectivityState): void {
    if (!navigator.onLine) {
      this.state.set('disconnected');
      return;
    }
    this.state.set('reconnecting');
    if (this.reconnectTimer == null) {
      this.reconnectTimer = window.setTimeout(() => void this.pingOnce(), 4_000);
    }
  }
}
