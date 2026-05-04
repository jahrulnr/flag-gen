import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, Observable, switchMap, throwError } from 'rxjs';

import { environment } from '../../environments/environment';
import { ApiErrorBody } from './models';
import { CsrfService } from './csrf.service';
import { type MappedHttpClientErrorPayload, userFacingHttpErrorMessage } from './user-facing-api-error';

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly csrf = inject(CsrfService);

  get<T>(path: string): Observable<T> {
    return this.http.get<T>(this.url(path), { withCredentials: true }).pipe(catchError((err) => this.mapError(err)));
  }

  post<T>(path: string, body: unknown): Observable<T> {
    return this.csrf.ensure().pipe(
      switchMap(() =>
        this.http.post<T>(this.url(path), body, {
          headers: this.unsafeHeaders(),
          withCredentials: true,
        }),
      ),
      catchError((err) => this.mapError(err)),
    );
  }

  patch<T>(path: string, body: unknown): Observable<T> {
    return this.csrf.ensure().pipe(
      switchMap(() =>
        this.http.patch<T>(this.url(path), body, {
          headers: this.unsafeHeaders(),
          withCredentials: true,
        }),
      ),
      catchError((err) => this.mapError(err)),
    );
  }

  delete<T>(path: string): Observable<T> {
    return this.csrf.ensure().pipe(
      switchMap(() =>
        this.http.delete<T>(this.url(path), {
          headers: this.unsafeHeaders(),
          withCredentials: true,
        }),
      ),
      catchError((err) => this.mapError(err)),
    );
  }

  private url(path: string): string {
    return `${environment.apiBaseUrl}${path.startsWith('/') ? path : `/${path}`}`;
  }

  private unsafeHeaders(): HttpHeaders {
    return new HttpHeaders({ 'Idempotency-Key': newCorrelationID() });
  }

  private mapError(err: unknown): Observable<never> {
    if (err instanceof HttpErrorResponse) {
      const body = err.error as ApiErrorBody | null;
      const rawMessage = typeof body?.message === 'string' ? body.message : typeof err.message === 'string' ? err.message : '';
      const correlationId = body?.correlationId;
      const code = typeof body?.error === 'string' ? body.error : undefined;
      const payload: MappedHttpClientErrorPayload = {
        status: err.status ?? 0,
        code,
        correlationId,
        rawMessage,
        userMessage: userFacingHttpErrorMessage(err.status ?? 0, code, rawMessage),
      };
      return throwError(() => ({
        ...payload,
        message: payload.userMessage,
      }));
    }
    return throwError(() => err);
  }
}

function newCorrelationID(): string {
  const cryptoRef = globalThis.crypto;
  if (cryptoRef && 'randomUUID' in cryptoRef) {
    return cryptoRef.randomUUID();
  }
  return `fg-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
