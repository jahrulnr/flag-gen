import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';

import { CsrfService } from './csrf.service';

export const apiHeadersInterceptor: HttpInterceptorFn = (req, next) => {
  const csrf = inject(CsrfService);
  const headers: Record<string, string> = {
    'X-Correlation-ID': req.headers.get('X-Correlation-ID') ?? newCorrelationID(),
  };
  const token = csrf.token();
  if (token && !isSafe(req.method)) {
    headers['X-CSRF-Token'] = token;
  }
  return next(req.clone({ setHeaders: headers, withCredentials: true }));
};

function isSafe(method: string): boolean {
  return method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
}

function newCorrelationID(): string {
  const cryptoRef = globalThis.crypto;
  if (cryptoRef && 'randomUUID' in cryptoRef) {
    return cryptoRef.randomUUID();
  }
  return `fg-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
