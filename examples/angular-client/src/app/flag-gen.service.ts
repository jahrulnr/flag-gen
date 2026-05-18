import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

import { environment } from '../environments/environment';

export interface ClientFlag {
  key: string;
  enabled: boolean;
  value: unknown;
  version: number;
}

export interface ClientFlagsResponse {
  flags: ClientFlag[];
  correlationId?: string;
}

/** Matches flag-gen evaluation result (POST/GET evaluate). */
export interface EvaluationResult {
  value: unknown;
  version: number;
  reason: string;
  matchedRuleId: string | null;
  source: string;
}

/** POST /v1/client/.../flags/evaluate */
export interface EvaluateResponse {
  results: Record<string, EvaluationResult>;
  timestamp: string;
  correlationId?: string;
}

@Injectable({ providedIn: 'root' })
export class FlagGenService {
  private readonly http = inject(HttpClient);

  private scopePath(): string {
    const { projectId, environmentKey } = environment;
    if (!projectId) {
      throw new Error('Set projectId in src/environments/environment.ts (UUID from GET /v1/projects)');
    }
    return `/client/projects/${projectId}/environments/${environmentKey}`;
  }

  private headers(): HttpHeaders {
    const h: Record<string, string> = {};
    if (environment.publicApiKey) {
      h['X-Public-Api-Key'] = environment.publicApiKey;
    }
    return new HttpHeaders(h);
  }

  listClientFlags(): Observable<ClientFlagsResponse> {
    return this.http.get<ClientFlagsResponse>(`${environment.flagGenBaseUrl}${this.scopePath()}/flags`, {
      headers: this.headers(),
    });
  }

  /**
   * POST evaluate — same contract as flag-gen (context + flags).
   * Only clientSide flags are evaluated on the server.
   */
  evaluate(
    flagKeys: string[],
    userId: string,
    attributes: Record<string, unknown> | null,
  ): Observable<EvaluateResponse> {
    return this.http.post<EvaluateResponse>(
      `${environment.flagGenBaseUrl}${this.scopePath()}/flags/evaluate`,
      {
        context: { userId, attributes: attributes ?? {} },
        flags: flagKeys,
      },
      { headers: this.headers() },
    );
  }
}
