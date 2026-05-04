import { CommonModule } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { AuditEntry } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import { formatDisplayTimestamp } from './format-display-timestamp';

@Component({
  selector: 'app-audit-page',
  imports: [CommonModule, FormsModule, WikiButtonComponent, InlineApiErrorComponent],
  templateUrl: './audit-page.html',
  styleUrl: './audit-page.scss',
})
export class AuditPage {
  private readonly api = inject(ApiService);
  readonly rows = signal<AuditEntry[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly searched = signal(false);
  filter = { projectId: '', entityType: 'flag', entityId: '' };

  formatTimestamp(raw: unknown): string {
    return formatDisplayTimestamp(raw);
  }

  search(): void {
    if (!this.filter.projectId) {
      this.error.set('Project ID is required');
      this.errorRef.set(null);
      return;
    }
    this.searched.set(true);
    const query = new URLSearchParams({
      entityType: this.filter.entityType,
      entityId: this.filter.entityId,
      limit: '100',
    });
    this.api.get<{ audit: AuditEntry[] }>(`/projects/${this.filter.projectId}/audit?${query.toString()}`).subscribe({
      next: (res) => {
        this.error.set(null);
        this.errorRef.set(null);
        this.rows.set(res.audit ?? []);
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load audit'),
    });
  }
}
