import { isPlatformBrowser } from '@angular/common';
import { inject, Injectable, PLATFORM_ID, signal } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';

import { ApiService } from './api.service';

const STORAGE_KEY = 'flag-gen-dev-session';

interface AuthUser {
  id: string;
  email: string;
  name: string;
  roleId?: string;
  roleKey?: string;
  role: string;
  permissions?: string[];
}

interface AuthResponse {
  user: AuthUser;
  correlationId?: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly platformId = inject(PLATFORM_ID);
  private readonly api = inject(ApiService);
  private readonly browser = isPlatformBrowser(this.platformId);

  readonly signedIn = signal(this.hasStoredSession());
  readonly user = signal<AuthUser | null>(null);

  constructor() {
    if (!this.browser) {
      return;
    }
    const marker = window.localStorage.getItem(STORAGE_KEY);
    if (marker === 'dev') {
      window.localStorage.removeItem(STORAGE_KEY);
      return;
    }
    if (marker === 'session') {
      this.refreshUserFromServer();
    }
  }

  login(email: string, password: string): Observable<AuthResponse> {
    return this.api.post<AuthResponse>('/auth/login', { email, password }).pipe(
      tap((res) => {
        this.user.set(res.user);
        this.signedIn.set(true);
        if (this.browser) {
          window.localStorage.setItem(STORAGE_KEY, 'session');
        }
      }),
    );
  }

  signOut(): void {
    this.signedIn.set(false);
    this.user.set(null);
    if (this.browser) {
      window.localStorage.removeItem(STORAGE_KEY);
    }
    this.api.post('/auth/logout', {}).subscribe({ error: () => undefined });
  }

  /** Route guard: password login re-validates flag_gen_session cookie via /auth/me. */
  ensureDashboardAccess(): Observable<boolean> {
    if (!this.browser) {
      return of(false);
    }
    const marker = window.localStorage.getItem(STORAGE_KEY);
    if (marker !== 'session') {
      this.signedIn.set(false);
      return of(false);
    }
    if (this.signedIn() && this.user()) {
      return of(true);
    }
    return this.api.get<AuthResponse>('/auth/me').pipe(
      tap((res) => {
        this.user.set(res.user);
        this.signedIn.set(true);
      }),
      map(() => true),
      catchError(() => {
        this.signOut();
        return of(false);
      }),
    );
  }

  private hasStoredSession(): boolean {
    if (!this.browser) {
      return false;
    }
    const marker = window.localStorage.getItem(STORAGE_KEY);
    return marker === 'session';
  }

  private refreshUserFromServer(): void {
    this.api.get<AuthResponse>('/auth/me').subscribe({
      next: (res) => {
        this.user.set(res.user);
        this.signedIn.set(true);
      },
      error: () => this.signOut(),
    });
  }
}
