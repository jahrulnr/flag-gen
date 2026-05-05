import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { AuditEntry } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { WorkspaceContextService } from '../../core/workspace-context.service';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { ListToolbarComponent } from '../../ui/list-toolbar/list-toolbar.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import { formatDisplayTimestamp } from './format-display-timestamp';

@Component({
  selector: 'app-audit-page',
  imports: [CommonModule, FormsModule, WikiButtonComponent, InlineApiErrorComponent, ListToolbarComponent],
  templateUrl: './audit-page.html',
  styleUrl: './audit-page.scss',
})
export class AuditPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly workspace = inject(WorkspaceContextService);
  readonly rows = signal<AuditEntry[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly searched = signal(false);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly loading = signal(false);

  filter = { projectId: '', entityType: 'flag', entityId: '' };

  ngOnInit(): void {
    if (!this.filter.projectId) {
      const pid = this.workspace.lastWorkspaceProjectId();
      if (pid) {
        this.filter.projectId = pid;
        this.search();
      }
    }
  }

  formatTimestamp(raw: unknown): string {
    return formatDisplayTimestamp(raw);
  }

  entityPrimary(row: AuditEntry): string {
    const key = row.entityKey ?? row.EntityKey;
    const name = row.entityName ?? row.EntityName;
    const id = row.entityId ?? row.EntityID ?? '';
    return (name || key || id || '').toString();
  }

  entitySecondary(row: AuditEntry): string {
    const type = (row.entityType ?? row.EntityType ?? '').toString();
    const key = row.entityKey ?? row.EntityKey;
    const id = row.entityId ?? row.EntityID ?? '';
    const parts: string[] = [];
    if (type) parts.push(type);
    if (key) parts.push(String(key));
    if (id) parts.push(id);
    return parts.join(' · ');
  }

  correlationShort(row: AuditEntry): string {
    const raw = (row.correlationId ?? row.CorrelationID ?? '')?.toString() ?? '';
    if (!raw) return '—';
    if (raw.length <= 14) return raw;
    return `${raw.slice(0, 10)}…`;
  }

  async copy(text: string | null | undefined): Promise<void> {
    const val = (text ?? '').toString();
    if (!val) return;
    try {
      await navigator.clipboard.writeText(val);
    } catch {
      // Fallback: best-effort for older browsers; do nothing if blocked.
    }
  }

  search(): void {
    if (!this.filter.projectId) {
      this.error.set('Project ID is required');
      this.errorRef.set(null);
      return;
    }
    this.searched.set(true);
    this.loading.set(true);
    const query = new URLSearchParams({
      entityType: this.filter.entityType,
      entityId: this.filter.entityId,
      limit: String(this.limit()),
      offset: String(this.offset()),
    });
    this.api
      .get<{ audit: AuditEntry[]; total?: number; limit?: number; offset?: number }>(`/projects/${this.filter.projectId}/audit?${query.toString()}`)
      .subscribe({
      next: (res) => {
        this.error.set(null);
        this.errorRef.set(null);
        this.rows.set(res.audit ?? []);
        this.total.set(res.total ?? 0);
        this.limit.set(res.limit ?? this.limit());
        this.offset.set(res.offset ?? this.offset());
        this.loading.set(false);
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load audit'),
    });
  }

  setQuery(q: string): void {
    this.filter.entityId = q;
    this.offset.set(0);
    this.search();
  }

  prevPage(): void {
    this.offset.set(Math.max(0, this.offset() - this.limit()));
    this.search();
  }

  nextPage(): void {
    this.offset.set(this.offset() + this.limit());
    this.search();
  }
}
