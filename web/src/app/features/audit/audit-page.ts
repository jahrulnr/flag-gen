import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { AuditEntry, Project, rowID, rowKey, rowName } from '../../core/models';
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
  private projectSearchDebounce: ReturnType<typeof setTimeout> | null = null;
  readonly rows = signal<AuditEntry[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly searched = signal(false);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly loading = signal(false);
  readonly projectPickerOpen = signal(false);
  readonly projectPickerQuery = signal('');
  readonly projectOptions = signal<Project[]>([]);
  readonly projectSearchLoading = signal(false);
  readonly selectedProject = signal<Project | null>(null);

  filter = { projectId: '', entityType: 'flag', entityId: '' };

  ngOnInit(): void {
    this.loadProjectOptions('');
    if (!this.filter.projectId) {
      const pid = this.workspace.lastWorkspaceProjectId();
      if (pid) {
        this.filter.projectId = pid;
        this.search();
      }
    }
  }

  projectTriggerLabel(): string {
    if (!this.filter.projectId) {
      return 'Choose project';
    }
    const selected = this.selectedProject();
    return selected ? rowName(selected) : shortProjectId(this.filter.projectId);
  }

  projectTriggerMeta(): string {
    if (!this.filter.projectId) {
      return 'Search by project name or key';
    }
    const selected = this.selectedProject();
    const key = selected ? rowKey(selected) : '';
    return key ? `Key ${key}` : 'Selected project';
  }

  projectName(project: Project): string {
    return rowName(project);
  }

  projectMeta(project: Project): string {
    const parts = [`Key ${rowKey(project)}`];
    const id = rowID(project);
    if (id) {
      parts.push(id);
    }
    return parts.join(' · ');
  }

  isProjectSelected(project: Project): boolean {
    return rowID(project) === this.filter.projectId;
  }

  toggleProjectPicker(): void {
    this.projectPickerOpen.update((open) => {
      const next = !open;
      if (next) {
        this.loadProjectOptions(this.projectPickerQuery());
      }
      return next;
    });
  }

  closeProjectPicker(): void {
    this.projectPickerOpen.set(false);
  }

  selectProject(project: Project): void {
    const id = rowID(project);
    if (!id) {
      return;
    }
    this.filter.projectId = id;
    this.selectedProject.set(project);
    this.projectPickerQuery.set('');
    this.offset.set(0);
    this.closeProjectPicker();
    this.search();
  }

  setProjectPickerQuery(q: string): void {
    this.projectPickerQuery.set(q);
    if (this.projectSearchDebounce) {
      clearTimeout(this.projectSearchDebounce);
    }
    this.projectSearchDebounce = setTimeout(() => this.loadProjectOptions(q), 180);
  }

  loadProjectOptions(q: string): void {
    this.projectSearchLoading.set(true);
    const query = new URLSearchParams({
      q: q.trim(),
      limit: '20',
      offset: '0',
    });
    this.api.get<{ projects: Project[] }>(`/projects?${query.toString()}`).subscribe({
      next: (res) => {
        const projects = res.projects ?? [];
        this.projectOptions.set(projects);
        if (this.filter.projectId && !this.selectedProject()) {
          const selected = projects.find((project) => rowID(project) === this.filter.projectId);
          if (selected) {
            this.selectedProject.set(selected);
          }
        }
        this.projectSearchLoading.set(false);
      },
      error: () => {
        this.projectOptions.set([]);
        this.projectSearchLoading.set(false);
      },
    });
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
      error: (err: unknown) => {
        this.loading.set(false);
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load audit');
      },
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

function shortProjectId(id: string): string {
  if (!id || id.length < 10) {
    return id;
  }
  return `${id.slice(0, 8)}…`;
}
