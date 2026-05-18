import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { EnvironmentRow, rowKey, rowName } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { ProjectDirectoryService } from '../../core/project-directory.service';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { IntegrationScopeCalloutComponent } from '../../ui/integration-scope-callout/integration-scope-callout.component';
import { ListToolbarComponent } from '../../ui/list-toolbar/list-toolbar.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-environments-page',
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    WikiButtonComponent,
    DestructiveConfirmDialogComponent,
    InlineApiErrorComponent,
    IntegrationScopeCalloutComponent,
    ListToolbarComponent,
  ],
  templateUrl: './environments-page.html',
  styleUrl: './environments-page.scss',
})
export class EnvironmentsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly projectDirectory = inject(ProjectDirectoryService);
  private readonly route = inject(ActivatedRoute);
  readonly projectId = signal('');
  readonly environments = signal<EnvironmentRow[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly q = signal('');
  readonly loading = signal(false);
  form = { key: '', name: '', sortOrder: 0 };

  ngOnInit(): void {
    this.route.paramMap.subscribe((params) => {
      this.projectId.set(params.get('projectId') ?? '');
      this.reload();
    });
  }

  reload(): void {
    if (!this.projectId()) {
      return;
    }
    this.error.set(null);
    this.errorRef.set(null);
    this.loading.set(true);
    const query = new URLSearchParams({
      q: this.q(),
      limit: String(this.limit()),
      offset: String(this.offset()),
    });
    this.api
      .get<{ environments: EnvironmentRow[]; total?: number; limit?: number; offset?: number }>(`/projects/${this.projectId()}/environments?${query.toString()}`)
      .subscribe({
        next: (res) => {
          this.environments.set(res.environments ?? []);
          this.total.set(res.total ?? 0);
          this.limit.set(res.limit ?? this.limit());
          this.offset.set(res.offset ?? this.offset());
          this.loading.set(false);
        },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load environments'),
    });
  }

  setQuery(q: string): void {
    this.q.set(q);
    this.offset.set(0);
    this.reload();
  }

  prevPage(): void {
    this.offset.set(Math.max(0, this.offset() - this.limit()));
    this.reload();
  }

  nextPage(): void {
    this.offset.set(this.offset() + this.limit());
    this.reload();
  }

  create(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.api.post(`/projects/${this.projectId()}/environments`, this.form).subscribe({
      next: () => {
        this.form = { key: '', name: '', sortOrder: 0 };
        this.reload();
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to create environment'),
    });
  }

  key(env: EnvironmentRow): string {
    return rowKey(env);
  }

  name(env: EnvironmentRow): string {
    return rowName(env);
  }

  projectLabel(): string {
    return this.projectDirectory.displayName(this.projectId());
  }

  remove(env: EnvironmentRow): void {
    const k = this.key(env);
    const nm = this.name(env);
    this.destroyDialog().open({
      title: 'Delete environment',
      body: `Remove environment "${nm}" (${k}) from this project. All flags in this environment must be deleted first or the server will reject the delete.`,
      expectedKey: k,
      confirmLabel: 'Delete environment permanently',
      onConfirm: () => {
        this.error.set(null);
        this.errorRef.set(null);
        const path = `/projects/${this.projectId()}/environments/${encodeURIComponent(k)}`;
        this.api.delete(path).subscribe({
          next: () => this.reload(),
          error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to delete environment'),
        });
      },
    });
  }
}
