import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { SsrBootstrapService } from '../../core/ssr-bootstrap.service';
import { SystemStatus } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-lifecycle-page',
  imports: [CommonModule, FormsModule, WikiButtonComponent, InlineApiErrorComponent],
  templateUrl: './lifecycle-page.html',
  styleUrl: './lifecycle-page.scss',
})
export class LifecyclePage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly bootstrap = inject(SsrBootstrapService);
  private readonly router = inject(Router);

  readonly status = signal<SystemStatus | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);

  install = {
    email: 'admin@example.com',
    name: 'Admin',
    password: '',
    projectKey: 'default',
    environmentKey: 'production',
  };

  ngOnInit(): void {
    this.bootstrap.load().subscribe({
      next: (data) => {
        this.status.set(data.status);
        this.loading.set(false);
        if (data.status?.database?.state === 'db_ready' || data.status?.database?.state === 'migration_optional') {
          void this.router.navigate(['/dashboard/projects']);
        }
      },
      error: (err: unknown) => {
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load system status');
        this.loading.set(false);
      },
    });
  }

  submitInstall(): void {
    const body = {
      admin: {
        email: this.install.email,
        name: this.install.name,
        password: this.install.password,
      },
      seed: {
        projectKey: this.install.projectKey,
        environmentKey: this.install.environmentKey,
      },
    };
    this.api.post('/system/install', body).subscribe({
      next: () => this.reloadStatus(),
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Install failed'),
    });
  }

  runMigration(): void {
    this.api.post('/system/migrate', {}).subscribe({
      next: () => this.reloadStatus(),
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Migration failed'),
    });
  }

  reloadStatus(): void {
    this.loading.set(true);
    this.api.get<SystemStatus>('/system/status').subscribe({
      next: (status) => {
        this.status.set(status);
        this.loading.set(false);
        if (status.database?.state === 'db_ready' || status.database?.state === 'migration_optional') {
          void this.router.navigate(['/dashboard/projects']);
        }
      },
      error: (err: unknown) => {
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to reload status');
        this.loading.set(false);
      },
    });
  }
}
