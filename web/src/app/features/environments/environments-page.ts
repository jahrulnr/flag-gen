import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { EnvironmentRow, rowKey, rowName } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-environments-page',
  imports: [CommonModule, FormsModule, RouterLink, WikiButtonComponent],
  templateUrl: './environments-page.html',
  styleUrl: './environments-page.scss',
})
export class EnvironmentsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly projectId = signal('');
  readonly environments = signal<EnvironmentRow[]>([]);
  readonly error = signal<string | null>(null);
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
    this.api.get<{ environments: EnvironmentRow[] }>(`/projects/${this.projectId()}/environments`).subscribe({
      next: (res) => this.environments.set(res.environments ?? []),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load environments'),
    });
  }

  create(): void {
    this.api.post(`/projects/${this.projectId()}/environments`, this.form).subscribe({
      next: () => {
        this.form = { key: '', name: '', sortOrder: 0 };
        this.reload();
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to create environment'),
    });
  }

  key(env: EnvironmentRow): string {
    return rowKey(env);
  }

  name(env: EnvironmentRow): string {
    return rowName(env);
  }

  remove(env: EnvironmentRow): void {
    const k = this.key(env);
    if (!confirm(`Delete environment "${this.name(env)}" (${k})? All flags must be removed first.`)) {
      return;
    }
    this.error.set(null);
    const path = `/projects/${this.projectId()}/environments/${encodeURIComponent(k)}`;
    this.api.delete(path).subscribe({
      next: () => this.reload(),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to delete environment'),
    });
  }
}
