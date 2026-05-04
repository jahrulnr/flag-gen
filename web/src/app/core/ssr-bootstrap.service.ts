import { inject, Injectable, makeStateKey, TransferState } from '@angular/core';
import { catchError, forkJoin, map, Observable, of, tap } from 'rxjs';

import { environment } from '../../environments/environment';
import { ApiService } from './api.service';
import { CsrfService } from './csrf.service';
import { CSRFResponse, DashboardBootstrap, SystemStatus } from './models';

const BOOTSTRAP_KEY = makeStateKey<DashboardBootstrap>('dashboard-bootstrap');

@Injectable({ providedIn: 'root' })
export class SsrBootstrapService {
  private readonly api = inject(ApiService);
  private readonly csrf = inject(CsrfService);
  private readonly transferState = inject(TransferState);

  load(): Observable<DashboardBootstrap> {
    const cached = this.transferState.get(BOOTSTRAP_KEY, null);
    if (cached) {
      this.csrf.set(cached.csrf);
      return of(cached);
    }

    return forkJoin({
      status: this.api.get<SystemStatus>('/system/status').pipe(catchError(() => of(null))),
      csrf: this.api.get<CSRFResponse>('/security/csrf').pipe(
        map((res) => res.csrf),
        catchError(() => of(null)),
      ),
    }).pipe(
      map(({ status, csrf }) => ({
        status,
        csrf,
        app: {
          name: 'Flag Gen',
          basePath: environment.appBasePath,
        },
      })),
      tap((bootstrap) => {
        this.csrf.set(bootstrap.csrf);
        this.transferState.set(BOOTSTRAP_KEY, bootstrap);
      }),
    );
  }
}
