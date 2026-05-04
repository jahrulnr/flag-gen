import { CommonModule } from '@angular/common';
import { Component, computed, DestroyRef, inject, OnInit, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs/operators';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { ConnectivityService, ConnectivityState } from '../../core/connectivity.service';
import { Project, rowID, rowKey, rowName, SystemStatus } from '../../core/models';
import { RealtimeService } from '../realtime/realtime.service';

@Component({
  selector: 'app-dashboard-shell',
  imports: [CommonModule, RouterLink, RouterLinkActive, RouterOutlet],
  templateUrl: './dashboard-shell.html',
  styleUrl: './dashboard-shell.scss',
})
export class DashboardShell implements OnInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  private readonly connectivity = inject(ConnectivityService);
  readonly realtime = inject(RealtimeService);

  readonly projects = signal<Project[]>([]);
  readonly status = signal<SystemStatus | null>(null);
  readonly selectedProjectId = signal('');
  readonly selectedEnvironmentKey = signal('');
  readonly routerUrl = signal(this.router.url);
  readonly drift = computed(() => this.status()?.database?.state === 'migration_optional');

  readonly connectivityLabel = computed(() => {
    const m: Record<ConnectivityState, string> = {
      connected: 'Connected',
      reconnecting: 'Reconnect',
      disconnected: 'Disconnected',
    };
    return m[this.connectivity.state()];
  });

  ngOnInit(): void {
    this.connectivity.attach(this.destroyRef);

    this.api.get<SystemStatus>('/system/status').subscribe({
      next: (status) => this.status.set(status),
      error: () => this.status.set(null),
    });
    this.reloadProjects();
    this.applyUrl(this.router.url);

    this.router.events
      .pipe(
        filter((e): e is NavigationEnd => e instanceof NavigationEnd),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(() => {
        this.routerUrl.set(this.router.url);
        this.applyUrl(this.router.url);
      });
  }

  reloadProjects(): void {
    this.api.get<{ projects: Project[] }>('/projects').subscribe({
      next: (res) => this.setProjects(res.projects ?? []),
      error: () => this.setProjects([]),
    });
  }

  private setProjects(projects: Project[]): void {
    this.projects.set(projects);
  }

  private applyUrl(url: string): void {
    const path = url.split('?')[0];
    if (path.includes('/dashboard/audit')) {
      this.selectedProjectId.set('');
      this.selectedEnvironmentKey.set('');
      return;
    }
    if (/\/dashboard\/projects\/?$/.test(path) || path.endsWith('/dashboard/projects')) {
      this.selectedProjectId.set('');
      this.selectedEnvironmentKey.set('');
      return;
    }
    const seg = path.match(/\/dashboard\/projects\/([^/]+)\/segments/);
    if (seg) {
      this.selectedProjectId.set(seg[1]);
      this.selectedEnvironmentKey.set('');
      this.ensureProjectLoaded(seg[1]);
      return;
    }
    const env = path.match(/\/dashboard\/projects\/([^/]+)\/environments(?:\/([^/]+))?/);
    if (env) {
      this.selectedProjectId.set(env[1]);
      this.selectedEnvironmentKey.set(env[2] ?? '');
      this.ensureProjectLoaded(env[1]);
      return;
    }
  }

  private ensureProjectLoaded(projectId: string): void {
    if (!projectId) {
      return;
    }
    if (!this.projects().some((p) => rowID(p) === projectId)) {
      this.reloadProjects();
    }
  }

  projectTitle(projectId: string): string {
    const p = this.projects().find((x) => rowID(x) === projectId);
    return p ? rowName(p) : projectId;
  }

  isFlagsArea(): boolean {
    return this.routerUrl().includes('/flags');
  }

  isFlagHistory(): boolean {
    return this.routerUrl().includes('/history');
  }

  isSegmentsArea(): boolean {
    return this.routerUrl().includes('/segments');
  }

  isEnvironmentsList(): boolean {
    const u = this.routerUrl().split('?')[0];
    if (u.includes('/segments')) {
      return false;
    }
    return /\/dashboard\/projects\/[^/]+\/environments\/?$/.test(u);
  }

  connectivityClass(): string {
    switch (this.connectivity.state()) {
      case 'connected':
        return 'conn conn--ok';
      case 'reconnecting':
        return 'conn conn--warn';
      default:
        return 'conn conn--bad';
    }
  }
}
