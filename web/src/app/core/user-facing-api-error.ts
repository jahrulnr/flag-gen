import type { WritableSignal } from '@angular/core';

/** Maps API `{ error, message, ... }` to operator-facing strings; strips internal bracket codes from messages. */

const BRACKET_PREFIX = /^\[[A-Z0-9_]+\]\s*/;

/** Strip `[CODE]` prefixes that may leak from pkg/error conventions. */
export function stripBracketErrorCodes(raw: string): string {
  let next = raw.trim();
  let prev = '';
  while (prev !== next) {
    prev = next;
    next = next.replace(BRACKET_PREFIX, '').trim();
  }
  return next || raw.trim();
}

const CODE_HINTS: Record<string, string> = {
  validation_error:
    'Some fields failed validation. Check required values and formatting, then fix the highlighted items and try again.',
  bad_request: 'The request could not be processed. Adjust your input and try again.',
  not_found: 'The item was not found. It may have been deleted by someone else.',
  conflict: 'This change conflicts with the current server state (for example duplicate key or stale version). Refresh and retry.',
  unauthorized: 'You are not authenticated. Sign in again and retry.',
  forbidden:
    'This action was blocked (often an expired secure session token). Reload the page, then verify you have permission.',
  dependency_error:
    'A dependent service responded with an error. Wait a moment and retry; contact an engineer if this continues.',
  degraded: 'The service is temporarily degraded. Retry in a moment; contact an engineer if outages persist.',
  rate_limited: 'Too many requests. Wait briefly and try again.',
  locked: 'The resource or operation is locked. Try again shortly.',
  internal_error:
    'An unexpected server error occurred. Retry once; contact an engineer with the reference code if it continues.',
};

function fallbackForHttpStatus(status: number): string {
  if (status === 0) {
    return 'Unable to reach the server. Check your network and retry.';
  }
  if (status === 413) {
    return 'Payload too large.';
  }
  if (status === 419 || status === 440) {
    return 'Secure session expired. Reload the page and try again.';
  }
  if (status === 401) {
    return CODE_HINTS['unauthorized'] ?? 'You are not authenticated.';
  }
  if (status === 403) {
    return CODE_HINTS['forbidden'] ?? CODE_HINTS['forbidden'];
  }
  if (status === 404) {
    return CODE_HINTS['not_found'] ?? 'Not found.';
  }
  if (status === 409) {
    return CODE_HINTS['conflict'] ?? 'Conflict.';
  }
  if (status === 422) {
    return CODE_HINTS['validation_error'] ?? 'Validation failed.';
  }
  if (status === 429) {
    return 'Too many requests. Wait and try again.';
  }
  if (status >= 500) {
    return CODE_HINTS['internal_error'] ?? 'Server error.';
  }
  if (status >= 400) {
    return CODE_HINTS['bad_request'] ?? 'Request error.';
  }
  return 'Something went wrong.';
}

/** Human headline for dashboards; optionally uses stripped server message when it looks helpful (non-empty, not redundant). */
export function userFacingHttpErrorMessage(
  status: number,
  errorCode: string | undefined | null,
  apiMessageRaw: string | undefined | null,
): string {
  const codeKey = typeof errorCode === 'string' ? errorCode.toLowerCase().trim() : '';
  const codeHint =
    codeKey && Object.prototype.hasOwnProperty.call(CODE_HINTS, codeKey)
      ? CODE_HINTS[codeKey as keyof typeof CODE_HINTS]
      : null;

  const stripped = stripBracketErrorCodes((apiMessageRaw ?? '').trim());
  /* Strip generic CSRF/forbidden echoes if we prefer friendlier banner */
  if (status === 403 && codeHint) {
    const lowerStripped = stripped.toLowerCase();
    if (!lowerStripped || lowerStripped.includes('csrf') || lowerStripped.includes('forbidden')) {
      return codeHint;
    }
  }

  /* Prefer API message after strip when it carries specifics */
  if (stripped.length > 3 && stripped.length < 560) {
    const looksBareCode = stripped === codeKey.replace(/_/g, '') || stripped === codeKey;
    const looksBareEnum = /^[A-Z][A-Z0-9_]*$/.test(stripped);
    if (!looksBareCode && !looksBareEnum) {
      return stripped.charAt(0).toUpperCase() + stripped.slice(1);
    }
  }

  if (codeHint) {
    return codeHint;
  }

  return fallbackForHttpStatus(status);
}

export interface MappedHttpClientErrorPayload {
  status: number;
  code?: string;
  correlationId?: string;
  rawMessage: string;
  userMessage: string;
}

export function looksLikeMappedHttpClientError(val: unknown): val is MappedHttpClientErrorPayload {
  if (!val || typeof val !== 'object') {
    return false;
  }
  const o = val as Record<string, unknown>;
  return typeof o['status'] === 'number' && typeof o['userMessage'] === 'string' && typeof o['rawMessage'] === 'string';
}

/** Apply mapped Http client error — or fallback — into dashboard error signals */
export function applyApiFailureToSignals(
  line: WritableSignal<string | null>,
  ref: WritableSignal<string | null>,
  err: unknown,
  fallback: string,
): void {
  if (looksLikeMappedHttpClientError(err)) {
    line.set(err.userMessage);
    ref.set(err.correlationId ?? null);
    return;
  }
  line.set(fallback);
  ref.set(null);
}
