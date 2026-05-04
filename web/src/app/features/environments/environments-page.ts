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
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-environments-page',
  imports: [CommonModule, FormsModule, RouterLink, WikiButtonComponent, DestructiveConfirmDialogComponent, InlineApiErrorComponent],
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
    this.api.get<{ environments: EnvironmentRow[] }>(`/projects/${this.projectId()}/environments`).subscribe({
      next: (res) => this.environments.set(res.environments ?? []),
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load environments'),
    });
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
