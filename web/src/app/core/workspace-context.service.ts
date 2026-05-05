import { effect, Injectable, signal } from '@angular/core';

/** Lightweight cross-page context: last workspace project/environment visited. */
@Injectable({ providedIn: 'root' })
export class WorkspaceContextService {
  private readonly storageProjectIdKey = 'flaggen.workspace.projectId';
  private readonly storageEnvKey = 'flaggen.workspace.environmentKey';

  /** Last projectId seen in any /dashboard/projects/:projectId/... route. */
  readonly lastWorkspaceProjectId = signal<string>(safeLocalStorageGet(this.storageProjectIdKey));

  /** Last environmentKey seen in any /dashboard/projects/:projectId/environments/:environmentKey/... route. */
  readonly lastWorkspaceEnvironmentKey = signal<string>(safeLocalStorageGet(this.storageEnvKey));

  constructor() {
    effect(() => safeLocalStorageSet(this.storageProjectIdKey, this.lastWorkspaceProjectId()));
    effect(() => safeLocalStorageSet(this.storageEnvKey, this.lastWorkspaceEnvironmentKey()));
  }
}

function safeLocalStorageGet(key: string): string {
  try {
    return (globalThis?.localStorage?.getItem(key) ?? '').toString();
  } catch {
    return '';
  }
}

function safeLocalStorageSet(key: string, value: string): void {
  try {
    const next = (value ?? '').toString();
    if (!next) {
      globalThis?.localStorage?.removeItem(key);
      return;
    }
    globalThis?.localStorage?.setItem(key, next);
  } catch {
    // Storage may be blocked; context remains in-memory only.
  }
}

