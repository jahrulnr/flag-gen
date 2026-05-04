import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Project, rowID, rowKey, rowName } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-projects-page',
  imports: [CommonModule, FormsModule, RouterLink, WikiButtonComponent],
  templateUrl: './projects-page.html',
  styleUrl: './projects-page.scss',
})
export class ProjectsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  readonly projects = signal<Project[]>([]);
  readonly error = signal<string | null>(null);
  form = { key: '', name: '', description: '' };

  ngOnInit(): void {
    this.reload();
  }

  reload(): void {
    this.api.get<{ projects: Project[] }>('/projects').subscribe({
      next: (res) => this.projects.set(res.projects ?? []),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load projects'),
    });
  }

  create(): void {
    this.api.post<{ project: Project }>('/projects', {
      key: this.form.key,
      name: this.form.name,
      description: this.form.description || null,
    }).subscribe({
      next: (res) => {
        this.form = { key: '', name: '', description: '' };
        this.reload();
        const pid = rowID(res.project);
        if (pid) {
          void this.router.navigate(['/dashboard/projects', pid, 'environments']);
        }
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to create project'),
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
    const label = `${this.name(project)} (${this.key(project)})`;
    if (!confirm(`Delete project "${label}"? Environments and segments must be removed first.`)) {
      return;
    }
    this.error.set(null);
    this.api.delete(`/projects/${this.id(project)}`).subscribe({
      next: () => this.reload(),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to delete project'),
    });
  }
}
