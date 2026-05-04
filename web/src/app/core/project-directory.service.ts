import { Injectable, inject, signal } from '@angular/core';

import { ApiService } from './api.service';
import { Project, rowID, rowKey, rowName } from './models';

/** Shared `/projects` list for display names outside the shell (avoids repeating UUID-only labels). */
@Injectable({ providedIn: 'root' })
export class ProjectDirectoryService {
  private readonly api = inject(ApiService);
  readonly projects = signal<Project[]>([]);

  reload(): void {
    this.api.get<{ projects: Project[] }>('/projects').subscribe({
      next: (res) => this.projects.set(res.projects ?? []),
      error: () => this.projects.set([]),
    });
  }

  displayName(projectId: string): string {
    const p = this.projects().find((x) => rowID(x) === projectId);
    return p ? rowName(p) : shortId(projectId);
  }

  displayKey(projectId: string): string {
    const p = this.projects().find((x) => rowID(x) === projectId);
    return p ? rowKey(p) : shortId(projectId);
  }
}

function shortId(id: string): string {
  if (!id || id.length < 10) {
    return id;
  }
  return `${id.slice(0, 8)}…`;
}
