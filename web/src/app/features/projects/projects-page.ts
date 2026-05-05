import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Project, rowID, rowKey, rowName } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { ListToolbarComponent } from '../../ui/list-toolbar/list-toolbar.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-projects-page',
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    WikiButtonComponent,
    DestructiveConfirmDialogComponent,
    InlineApiErrorComponent,
    ListToolbarComponent,
  ],
  templateUrl: './projects-page.html',
  styleUrl: './projects-page.scss',
})
export class ProjectsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  readonly projects = signal<Project[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly q = signal('');
  readonly loading = signal(false);
  form = { key: '', name: '', description: '' };

  ngOnInit(): void {
    this.reload();
  }

  reload(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.loading.set(true);
    const query = new URLSearchParams({
      q: this.q(),
      limit: String(this.limit()),
      offset: String(this.offset()),
    });
    this.api.get<{ projects: Project[]; total?: number; limit?: number; offset?: number }>(`/projects?${query.toString()}`).subscribe({
      next: (res) => {
        this.projects.set(res.projects ?? []);
        this.total.set(res.total ?? 0);
        this.limit.set(res.limit ?? this.limit());
        this.offset.set(res.offset ?? this.offset());
        this.loading.set(false);
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load projects'),
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
    this.api
      .post<{ project: Project }>('/projects', {
        key: this.form.key,
        name: this.form.name,
        description: this.form.description || null,
      })
      .subscribe({
        next: (res) => {
          this.form = { key: '', name: '', description: '' };
          this.reload();
          const pid = rowID(res.project);
          if (pid) {
            void this.router.navigate(['/dashboard/projects', pid, 'environments']);
          }
        },
        error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to create project'),
      });
  }

  id(project: Project): string {
    return rowID(project);
  }

  key(project: Project): string {
    return rowKey(project);
  }

  name(project: Project): string {
    return rowName(project);
  }

  remove(project: Project): void {
    const k = this.key(project);
    const label = this.name(project);
    this.destroyDialog().open({
      title: 'Delete project',
      body: `Permanently remove project "${label}" (${k}). All environments, flags, and segments under this project must be removed first or the server will reject the delete.`,
      expectedKey: k,
      confirmLabel: 'Delete project permanently',
      onConfirm: () => {
        this.error.set(null);
        this.errorRef.set(null);
        this.api.delete(`/projects/${this.id(project)}`).subscribe({
          next: () => this.reload(),
          error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to delete project'),
        });
      },
    });
  }
}
