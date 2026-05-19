import { CommonModule } from '@angular/common';
import { Component, computed, DestroyRef, inject, OnInit, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs/operators';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { ProjectDirectoryService } from '../../core/project-directory.service';
import { ConnectivityService, ConnectivityState } from '../../core/connectivity.service';
import { Flag, rowID, rowKey, SystemStatus } from '../../core/models';
import { WorkspaceContextService } from '../../core/workspace-context.service';
import { RealtimeService } from '../realtime/realtime.service';

@Component({
  selector: 'app-dashboard-shell',
  imports: [CommonModule, RouterLink, RouterLinkActive, RouterOutlet],
  templateUrl: './dashboard-shell.html',
  styleUrl: './dashboard-shell.scss',
})
export class DashboardShell implements OnInit {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  readonly projectDirectory = inject(ProjectDirectoryService);
  private readonly workspace = inject(WorkspaceContextService);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  private readonly connectivity = inject(ConnectivityService);
  readonly realtime = inject(RealtimeService);

  readonly status = signal<SystemStatus | null>(null);
  readonly selectedProjectId = signal('');
  readonly selectedEnvironmentKey = signal('');
  /** Short label for the flag crumb on history URLs (resolved from list endpoint). */
  readonly historyFlagLabel = signal<string | null>(null);
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
    this.projectDirectory.reload();
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
    this.projectDirectory.reload();
  }

  signOut(): void {
    this.auth.signOut();
    void this.router.navigateByUrl('/login');
  }

  private applyUrl(url: string): void {
    const path = url.split('?')[0];
    this.historyFlagLabel.set(null);
    if (
      path.includes('/dashboard/audit') ||
      path.includes('/dashboard/users') ||
      path.includes('/dashboard/roles') ||
      path.includes('/dashboard/developer-wiki')
    ) {
      this.selectedProjectId.set('');
      this.selectedEnvironmentKey.set('');
      return;
    }
    if (/\/dashboard\/projects\/?$/.test(path) || path.endsWith('/dashboard/projects')) {
      this.selectedProjectId.set('');
      this.selectedEnvironmentKey.set('');
      return;
    }
    const hist = path.match(
      /\/dashboard\/projects\/([^/]+)\/environments\/([^/]+)\/flags\/([^/]+)\/history\/?$/,
    );
    if (hist) {
      this.selectedProjectId.set(hist[1]);
      this.selectedEnvironmentKey.set(hist[2]);
      this.workspace.lastWorkspaceProjectId.set(hist[1]);
      this.workspace.lastWorkspaceEnvironmentKey.set(hist[2]);
      this.ensureProjectLoaded(hist[1]);
      this.loadHistoryFlagCrumb(hist[1], hist[2], hist[3]);
      return;
    }
    const seg = path.match(/\/dashboard\/projects\/([^/]+)\/segments/);
    if (seg) {
      this.selectedProjectId.set(seg[1]);
      this.selectedEnvironmentKey.set('');
      this.workspace.lastWorkspaceProjectId.set(seg[1]);
      this.ensureProjectLoaded(seg[1]);
      return;
    }
    const env = path.match(/\/dashboard\/projects\/([^/]+)\/environments(?:\/([^/]+))?/);
    if (env) {
      this.selectedProjectId.set(env[1]);
      this.selectedEnvironmentKey.set(env[2] ?? '');
      this.workspace.lastWorkspaceProjectId.set(env[1]);
      if (env[2]) {
        this.workspace.lastWorkspaceEnvironmentKey.set(env[2]);
      }
      this.ensureProjectLoaded(env[1]);
      return;
    }
  }

  private ensureProjectLoaded(projectId: string): void {
    if (!projectId) {
      return;
    }
    if (!this.projectDirectory.projects().some((p) => rowID(p) === projectId)) {
      this.projectDirectory.reload();
    }
  }

  private loadHistoryFlagCrumb(
    projectId: string,
    environmentKey: string,
    flagRouteParam: string,
  ): void {
    this.api
      .get<{ flags: Flag[] }>(`/projects/${projectId}/environments/${environmentKey}/flags`)
      .subscribe({
        next: (res) => {
          const flags = res.flags ?? [];
          const hit =
            flags.find((f) => rowID(f) === flagRouteParam) ??
            flags.find((f) => rowKey(f) === flagRouteParam);
          const label = hit ? rowKey(hit) : shortRouteId(flagRouteParam);
          this.historyFlagLabel.set(label);
        },
        error: () => this.historyFlagLabel.set(shortRouteId(flagRouteParam)),
      });
  }

  projectTitle(projectId: string): string {
    return this.projectDirectory.displayName(projectId);
  }

  /** Same visible string as environment key → add micro-labels so crumbs are not ambiguous. */
  projectEnvLabelCollides(projectId: string, envKey: string): boolean {
    return !!projectId && !!envKey && this.projectTitle(projectId).trim() === envKey.trim();
  }

  isAuditRoute(): boolean {
    return this.routerUrl().split('?')[0].includes('/dashboard/audit');
  }

  isUsersRoute(): boolean {
    return this.routerUrl().split('?')[0].includes('/dashboard/users');
  }

  isRolesRoute(): boolean {
    return this.routerUrl().split('?')[0].includes('/dashboard/roles');
  }

  isWikiGuideRoute(): boolean {
    return this.routerUrl().split('?')[0].includes('/dashboard/wiki/guide');
  }

  isWikiFlagGenRoute(): boolean {
    return this.routerUrl().split('?')[0].includes('/dashboard/wiki/flag-gen');
  }

  isWikiDeveloperRoute(): boolean {
    const path = this.routerUrl().split('?')[0];
    return path.includes('/dashboard/wiki/developer') || path.includes('/dashboard/developer-wiki');
  }

  isWikiRoute(): boolean {
    return this.isWikiGuideRoute() || this.isWikiFlagGenRoute() || this.isWikiDeveloperRoute();
  }

  wikiBreadcrumbLabel(): string {
    if (this.isWikiGuideRoute()) return 'Panduan';
    if (this.isWikiFlagGenRoute()) return 'Konsep Flag Gen';
    return 'Developer';
  }

  isAccessRoute(): boolean {
    return this.isUsersRoute() || this.isRolesRoute();
  }

  isSystemRoute(): boolean {
    return this.isAuditRoute();
  }

  isProjectsListCurrent(): boolean {
    const u = this.routerUrl().split('?')[0];
    return /\/dashboard\/projects\/?$/.test(u);
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

function shortRouteId(id: string): string {
  if (!id || id.length <= 12) {
    return id;
  }
  return `${id.slice(0, 10)}…`;
}
