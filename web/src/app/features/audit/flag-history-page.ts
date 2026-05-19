import { CommonModule } from '@angular/common';
import { Component, computed, inject, OnInit, signal, viewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Flag, FlagVersion, rowID, rowKey, rowName } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import { formatDisplayTimestamp } from './format-display-timestamp';

@Component({
  selector: 'app-flag-history-page',
  imports: [CommonModule, WikiButtonComponent, InlineApiErrorComponent, DestructiveConfirmDialogComponent],
  templateUrl: './flag-history-page.html',
  styleUrl: './flag-history-page.scss',
})
export class FlagHistoryPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly versions = signal<FlagVersion[]>([]);
  readonly flagMeta = signal<Flag | null>(null);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly rollingBack = signal(false);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);

  projectId = '';
  environmentKey = '';
  flagId = '';

  formatTimestamp(raw: unknown): string {
    return formatDisplayTimestamp(raw);
  }

  readonly historyEyebrow = computed(() => {
    const f = this.flagMeta();
    return f ? rowKey(f) : this.flagId ? shortIdLabel(this.flagId) : 'Flag';
  });

  readonly flagHeadingDetail = computed(() => {
    const f = this.flagMeta();
    if (!f) {
      return '';
    }
    const k = rowKey(f);
    const n = rowName(f);
    return n !== k ? n : '';
  });

  ngOnInit(): void {
    this.route.paramMap.subscribe((params) => {
      this.projectId = params.get('projectId') ?? '';
      this.environmentKey = params.get('environmentKey') ?? '';
      this.flagId = params.get('flagId') ?? '';
      this.reload();
      this.reloadFlagMeta();
    });
  }

  reload(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.api
      .get<{ versions: FlagVersion[] }>(`/projects/${this.projectId}/environments/${this.environmentKey}/flags/${this.flagId}/versions`)
      .subscribe({
        next: (res) => this.versions.set(res.versions ?? []),
        error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load versions'),
      });
  }

  currentFlagVersion(): number {
    const f = this.flagMeta();
    return Number(f?.version ?? f?.Version ?? 0);
  }

  versionNumber(row: FlagVersion): number {
    return Number(row.version ?? row.Version ?? 0);
  }

  canRollback(row: FlagVersion): boolean {
    const v = this.versionNumber(row);
    const cur = this.currentFlagVersion();
    return v > 0 && cur > 0 && v < cur && !this.rollingBack();
  }

  rollbackTo(row: FlagVersion): void {
    const targetVersion = this.versionNumber(row);
    const cur = this.currentFlagVersion();
    const flag = this.flagMeta();
    const flagKey = flag ? rowKey(flag) : this.flagId;
    if (!targetVersion || !cur || !flagKey) {
      return;
    }
    this.destroyDialog().open({
      title: 'Rollback flag version',
      body: `Restore this flag to version v${targetVersion}? Current version is v${cur}. This creates a new version from the historical snapshot.`,
      expectedKey: flagKey,
      confirmLabel: 'Rollback to this version',
      onConfirm: () => this.executeRollback(targetVersion, cur, flagKey),
    });
  }

  private executeRollback(targetVersion: number, currentVersion: number, flagKey: string): void {
    this.rollingBack.set(true);
    this.error.set(null);
    this.errorRef.set(null);
    const path = `/projects/${this.projectId}/environments/${this.environmentKey}/flags/${encodeURIComponent(flagKey)}/rollback`;
    this.api.post<{ flag: Flag }>(path, { targetVersion, version: currentVersion }).subscribe({
      next: () => {
        this.rollingBack.set(false);
        this.reloadFlagMeta();
        this.reload();
      },
      error: (err: unknown) => {
        this.rollingBack.set(false);
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to rollback flag');
      },
    });
  }

  private reloadFlagMeta(): void {
    this.flagMeta.set(null);
    if (!this.projectId || !this.environmentKey || !this.flagId) {
      return;
    }
    const path = `/projects/${this.projectId}/environments/${this.environmentKey}/flags`;
    this.api.get<{ flags: Flag[] }>(path).subscribe({
      next: (res) => {
        const flags = res.flags ?? [];
        const byId = flags.find((x) => rowID(x) === this.flagId);
        const byKey = flags.find((x) => rowKey(x) === this.flagId);
        this.flagMeta.set(byId ?? byKey ?? null);
      },
      error: () => this.flagMeta.set(null),
    });
  }
}

function shortIdLabel(id: string): string {
  if (id.length <= 14) {
    return id;
  }
  return `${id.slice(0, 10)}…`;
}
