import { isPlatformBrowser } from '@angular/common';
import { inject, Injectable, PLATFORM_ID, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';

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

  readonly signedIn = signal(this.restore());
  readonly user = signal<AuthUser | null>(null);

  signInDev(): void {
    this.signedIn.set(true);
    if (this.browser) {
      window.localStorage.setItem(STORAGE_KEY, 'dev');
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

  private restore(): boolean {
    if (!this.browser) {
      return false;
    }
    return window.localStorage.getItem(STORAGE_KEY) === 'dev';
  }
}
