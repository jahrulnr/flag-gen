import { CommonModule } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { AuditEntry } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-audit-page',
  imports: [CommonModule, FormsModule, WikiButtonComponent],
  templateUrl: './audit-page.html',
  styleUrl: './audit-page.scss',
})
export class AuditPage {
  private readonly api = inject(ApiService);
  readonly rows = signal<AuditEntry[]>([]);
  readonly error = signal<string | null>(null);
  filter = { projectId: '', entityType: 'flag', entityId: '' };

  search(): void {
    if (!this.filter.projectId) {
      this.error.set('Project ID is required');
      return;
    }
    const query = new URLSearchParams({
      entityType: this.filter.entityType,
      entityId: this.filter.entityId,
      limit: '100',
    });
    this.api.get<{ audit: AuditEntry[] }>(`/projects/${this.filter.projectId}/audit?${query.toString()}`).subscribe({
      next: (res) => this.rows.set(res.audit ?? []),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load audit'),
    });
  }
}
