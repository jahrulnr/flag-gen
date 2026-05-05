import { inject, Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { map, Observable, of, tap } from 'rxjs';

import { environment } from '../../environments/environment';
import { CSRFContract, CSRFResponse } from './models';

@Injectable({ providedIn: 'root' })
export class CsrfService {
  private readonly http = inject(HttpClient);
  private readonly current = signal<CSRFContract | null>(null);

  token(): string | null {
    return this.currentOrEmbedded()?.token ?? null;
  }

  set(contract: CSRFContract | null): void {
    this.current.set(contract);
  }

  ensure(): Observable<CSRFContract> {
    const existing = this.currentOrEmbedded();
    if (existing && new Date(existing.expiresAt).getTime() > Date.now() + 30_000) {
      return of(existing);
    }
    return this.refresh();
  }

  refresh(): Observable<CSRFContract> {
    return this.http.get<CSRFResponse>(`${environment.apiBaseUrl}/security/csrf`, { withCredentials: true }).pipe(
      map((res) => res.csrf),
      tap((contract) => this.current.set(contract)),
    );
  }

  private currentOrEmbedded(): CSRFContract | null {
    const existing = this.current();
    if (existing) {
      return existing;
    }
    const embedded = embeddedCSRF();
    if (embedded) {
      this.current.set(embedded);
    }
    return embedded;
  }
}

function embeddedCSRF(): CSRFContract | null {
  const doc = globalThis.document;
  const el = doc?.getElementById('flag-gen-bootstrap');
  const raw = el?.textContent;
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { csrf?: CSRFContract | null };
    return parsed.csrf ?? null;
  } catch {
    return null;
  }
}
