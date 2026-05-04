import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Flag, rowID, rowKey, rowName } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import { RealtimeService } from '../realtime/realtime.service';

@Component({
  selector: 'app-flags-page',
  imports: [CommonModule, FormsModule, RouterLink, WikiButtonComponent],
  templateUrl: './flags-page.html',
  styleUrl: './flags-page.scss',
})
export class FlagsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly realtime = inject(RealtimeService);

  readonly projectId = signal('');
  readonly environmentKey = signal('');
  readonly flags = signal<Flag[]>([]);
  readonly error = signal<string | null>(null);

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
    this.api.get<{ flags: Flag[] }>(this.flagsPath()).subscribe({
      next: (res) => this.flags.set(res.flags ?? []),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load flags'),
    });
  }

  create(): void {
    const body = this.toPayload();
    this.api.post<{ flag: Flag }>(this.flagsPath(), body).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to create flag'),
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
    this.api.patch<{ flag: Flag }>(`${this.flagsPath()}/${this.form.id}`, this.toPayload(true)).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to update flag'),
    });
  }

  remove(flag: Flag): void {
    const id = rowID(flag);
    const version = flag.version ?? flag.Version ?? 1;
    this.api.delete(`${this.flagsPath()}/${id}?version=${version}`).subscribe({
      next: () => this.reload(),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to delete flag'),
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
