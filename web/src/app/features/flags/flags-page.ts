import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Flag, rowID, rowKey, rowName } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { ProjectDirectoryService } from '../../core/project-directory.service';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { ListToolbarComponent } from '../../ui/list-toolbar/list-toolbar.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import { RealtimeService } from '../realtime/realtime.service';

@Component({
  selector: 'app-flags-page',
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    WikiButtonComponent,
    DestructiveConfirmDialogComponent,
    InlineApiErrorComponent,
    ListToolbarComponent,
  ],
  templateUrl: './flags-page.html',
  styleUrl: './flags-page.scss',
})
export class FlagsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly projectDirectory = inject(ProjectDirectoryService);
  private readonly route = inject(ActivatedRoute);
  private readonly realtime = inject(RealtimeService);

  readonly projectId = signal('');
  readonly environmentKey = signal('');
  readonly flags = signal<Flag[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly q = signal('');
  readonly loading = signal(false);
  readonly valueTypeMenuOpen = signal(false);

  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);

  form = this.defaultForm();

  ngOnInit(): void {
    this.route.paramMap.subscribe((params) => {
      this.projectId.set(params.get('projectId') ?? '');
      this.environmentKey.set(params.get('environmentKey') ?? '');
      this.realtime.connect(this.projectId(), this.environmentKey());
      this.reload();
    });
  }

  reload(): void {
    if (!this.projectId() || !this.environmentKey()) {
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
    this.api.get<{ flags: Flag[]; total?: number; limit?: number; offset?: number }>(`${this.flagsPath()}?${query.toString()}`).subscribe({
      next: (res) => {
        this.flags.set(res.flags ?? []);
        this.total.set(res.total ?? 0);
        this.limit.set(res.limit ?? this.limit());
        this.offset.set(res.offset ?? this.offset());
        this.loading.set(false);
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load flags'),
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
    const body = this.toPayload();
    this.error.set(null);
    this.errorRef.set(null);
    this.api.post<{ flag: Flag }>(this.flagsPath(), body).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to create flag'),
    });
  }

  edit(flag: Flag): void {
    this.form = {
      id: rowID(flag),
      key: rowKey(flag),
      name: rowName(flag),
      description: flag.description ?? flag.Description ?? '',
      enabled: flag.enabled ?? flag.Enabled ?? false,
      clientSide: flag.clientSide ?? flag.ClientSide ?? false,
      valueType: flag.value?.type ?? flag.Value?.Type ?? 'boolean',
      defaultValue: String(flag.value?.default ?? flag.Value?.Default ?? false),
      rolloutPercentage: flag.rollout?.percentage ?? flag.Rollout?.Percentage ?? 0,
      rolloutStickiness: flag.rollout?.stickiness ?? flag.Rollout?.Stickiness ?? 'userId',
      rulesJson: JSON.stringify(flag.rules ?? flag.Rules ?? [], null, 2),
      version: flag.version ?? flag.Version ?? 1,
    };
  }

  update(): void {
    if (!this.form.id) {
      return;
    }
    this.error.set(null);
    this.errorRef.set(null);
    this.api.patch<{ flag: Flag }>(`${this.flagsPath()}/${this.form.id}`, this.toPayload(true)).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to update flag'),
    });
  }

  remove(flag: Flag): void {
    const id = rowID(flag);
    const version = flag.version ?? flag.Version ?? 1;
    const k = this.key(flag);
    const displayName = this.name(flag);
    this.destroyDialog().open({
      title: 'Delete flag',
      body: `Remove "${displayName}" (${k}) from environment "${this.environmentKey()}". Clients will stop receiving this configuration for that key.`,
      expectedKey: k,
      confirmLabel: 'Delete flag permanently',
      onConfirm: () => {
        this.error.set(null);
        this.errorRef.set(null);
        this.api.delete(`${this.flagsPath()}/${id}?version=${version}`).subscribe({
          next: () => this.reload(),
          error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to delete flag'),
        });
      },
    });
  }

  id(flag: Flag): string {
    return rowID(flag);
  }

  key(flag: Flag): string {
    return rowKey(flag);
  }

  name(flag: Flag): string {
    return rowName(flag);
  }

  version(flag: Flag): number {
    return flag.version ?? flag.Version ?? 0;
  }

  projectLabel(): string {
    return this.projectDirectory.displayName(this.projectId());
  }

  flagsEyebrow(): string {
    const p = this.projectLabel().trim();
    const env = this.environmentKey().trim();
    if (p === env) {
      return `Project and environment share the label “${p}”`;
    }
    return `Proj. ${p} · Env ${env}`;
  }

  onValueTypeChange(): void {
    if (this.form.valueType === 'boolean') {
      const raw = String(this.form.defaultValue).trim().toLowerCase();
      this.form.defaultValue = raw === 'true' ? 'true' : 'false';
    }
  }

  valueTypeLabel(): string {
    return this.form.valueType === 'json' ? 'JSON' : 'Boolean';
  }

  toggleValueTypeMenu(): void {
    this.valueTypeMenuOpen.update((open) => !open);
  }

  closeValueTypeMenu(): void {
    this.valueTypeMenuOpen.set(false);
  }

  chooseValueType(valueType: 'boolean' | 'json'): void {
    this.form.valueType = valueType;
    this.onValueTypeChange();
    this.closeValueTypeMenu();
  }

  private flagsPath(): string {
    return `/projects/${this.projectId()}/environments/${this.environmentKey()}/flags`;
  }

  private toPayload(includeVersion = false): Record<string, unknown> {
    const rules = parseJSON(this.form.rulesJson, []);
    const payload: Record<string, unknown> = {
      key: this.form.key,
      name: this.form.name,
      description: this.form.description,
      enabled: this.form.enabled,
      clientSide: this.form.clientSide,
      value: {
        type: this.form.valueType,
        default: this.form.valueType === 'boolean' ? this.form.defaultValue === 'true' : parseJSON(this.form.defaultValue, this.form.defaultValue),
      },
      rules,
      rollout: {
        percentage: Number(this.form.rolloutPercentage),
        stickiness: this.form.rolloutStickiness,
      },
    };
    if (includeVersion) {
      payload['version'] = Number(this.form.version);
    }
    return payload;
  }

  private defaultForm() {
    return {
      id: '',
      key: '',
      name: '',
      description: '',
      enabled: false,
      clientSide: false,
      valueType: 'boolean',
      defaultValue: 'false',
      rolloutPercentage: 0,
      rolloutStickiness: 'userId',
      rulesJson: '[]',
      version: 1,
    };
  }
}

function parseJSON(value: string, fallback: unknown): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return fallback;
  }
}
