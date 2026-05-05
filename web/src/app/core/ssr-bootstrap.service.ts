import { inject, Injectable, makeStateKey, TransferState } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';

import { environment } from '../../environments/environment';
import { ApiService } from './api.service';
import { CsrfService } from './csrf.service';
import { DashboardBootstrap, SystemStatus } from './models';

const BOOTSTRAP_KEY = makeStateKey<DashboardBootstrap>('dashboard-bootstrap');

@Injectable({ providedIn: 'root' })
export class SsrBootstrapService {
  private readonly api = inject(ApiService);
  private readonly csrf = inject(CsrfService);
  private readonly transferState = inject(TransferState);

  load(): Observable<DashboardBootstrap> {
    const cached = this.transferState.get(BOOTSTRAP_KEY, null);
    if (cached) {
      if (cached.csrf) {
        this.csrf.set(cached.csrf);
      }
      return of(cached);
    }

    return this.api.get<SystemStatus>('/system/status').pipe(
      catchError(() => of(null)),
      map((status) => ({
        status,
        csrf: null,
        app: {
          name: 'Flag Gen',
          basePath: environment.appBasePath,
        },
      })),
      tap((bootstrap) => {
        if (bootstrap.csrf) {
          this.csrf.set(bootstrap.csrf);
        }
        this.transferState.set(BOOTSTRAP_KEY, bootstrap);
      }),
    );
  }
}
