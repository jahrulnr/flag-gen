import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiService } from '../../core/api.service';
import {
  effectiveSearchQuery,
  scheduleDebouncedSearch,
  SearchDebounceHandle,
} from '../../core/list-search';
import { EvaluationResult, Flag, rowID, rowKey, rowName } from '../../core/models';
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
  private searchDebounce: SearchDebounceHandle = null;

  readonly projectId = signal('');
  readonly environmentKey = signal('');
  readonly flags = signal<Flag[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly searchText = signal('');
  readonly loading = signal(false);
  readonly valueTypeMenuOpen = signal(false);
  readonly formDialogOpen = signal(false);
  readonly formSubmitting = signal(false);
  readonly formError = signal<string | null>(null);
  readonly successMessage = signal<string | null>(null);
  readonly evaluateOpen = signal(false);
  readonly evaluating = signal(false);
  readonly evaluateError = signal<string | null>(null);
  readonly evaluateResult = signal<EvaluationResult | null>(null);
  readonly evaluateFlagKey = signal('');
  evaluateForm = {
    userId: 'user-1',
    attributesJson: '{\n  "country": "ID"\n}',
  };

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
      q: effectiveSearchQuery(this.searchText()),
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
    this.searchText.set(q);
    this.offset.set(0);
    this.searchDebounce = scheduleDebouncedSearch(this.searchDebounce, q, () => this.reload());
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
    const body = this.payloadOrShowError();
    if (!body) {
      return;
    }
    this.formSubmitting.set(true);
    this.formError.set(null);
    this.api.post<{ flag: Flag }>(this.flagsPath(), body).subscribe({
      next: () => {
        this.formSubmitting.set(false);
        this.closeFormDialog();
        this.successMessage.set(`Flag ${this.form.key} created`);
        this.reload();
      },
      error: (err: unknown) => {
        this.formSubmitting.set(false);
        this.formError.set(errorText(err, 'Failed to create flag'));
      },
    });
  }

  openCreate(): void {
    this.form = this.defaultForm();
    this.formError.set(null);
    this.successMessage.set(null);
    this.formDialogOpen.set(true);
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
      defaultValue: defaultValueToFormString(
        flag.value?.default ?? flag.Value?.Default,
        flag.value?.type ?? flag.Value?.Type ?? 'boolean',
      ),
      rolloutPercentage: flag.rollout?.percentage ?? flag.Rollout?.Percentage ?? 0,
      rolloutStickiness: flag.rollout?.stickiness ?? flag.Rollout?.Stickiness ?? 'userId',
      rulesJson: JSON.stringify(flag.rules ?? flag.Rules ?? [], null, 2),
      version: flag.version ?? flag.Version ?? 1,
    };
    this.formError.set(null);
    this.successMessage.set(null);
    this.formDialogOpen.set(true);
  }

  openEvaluate(flag: Flag): void {
    this.evaluateFlagKey.set(this.key(flag));
    this.evaluateResult.set(null);
    this.evaluateError.set(null);
    this.evaluateOpen.set(true);
  }

  closeEvaluate(): void {
    if (this.evaluating()) {
      return;
    }
    this.evaluateOpen.set(false);
    this.evaluateError.set(null);
  }

  runEvaluate(): void {
    const flagKey = this.evaluateFlagKey();
    if (!flagKey) {
      return;
    }
    const attrs = parseJSON(this.evaluateForm.attributesJson, invalidJSON);
    if (attrs === invalidJSON || attrs === null || typeof attrs !== 'object' || Array.isArray(attrs)) {
      this.evaluateError.set('Attributes must be a JSON object.');
      return;
    }
    this.evaluating.set(true);
    this.evaluateError.set(null);
    this.evaluateResult.set(null);
    this.api
      .post<{ results: Record<string, EvaluationResult> }>(`${this.flagsPath()}/evaluate`, {
        context: {
          userId: this.evaluateForm.userId,
          attributes: attrs,
        },
        flags: [flagKey],
      })
      .subscribe({
        next: (res) => {
          this.evaluating.set(false);
          this.evaluateResult.set(res.results?.[flagKey] ?? null);
        },
        error: (err: unknown) => {
          this.evaluating.set(false);
          this.evaluateError.set(errorText(err, 'Failed to evaluate flag'));
        },
      });
  }

  closeFormDialog(): void {
    if (this.formSubmitting()) {
      return;
    }
    this.formDialogOpen.set(false);
    this.formError.set(null);
    this.valueTypeMenuOpen.set(false);
  }

  update(): void {
    if (!this.form.id) {
      return;
    }
    const body = this.payloadOrShowError(true);
    if (!body) {
      return;
    }
    this.formSubmitting.set(true);
    this.formError.set(null);
    this.api.patch<{ flag: Flag }>(`${this.flagsPath()}/${this.form.id}`, body).subscribe({
      next: () => {
        this.formSubmitting.set(false);
        this.closeFormDialog();
        this.successMessage.set(`Flag ${this.form.key} updated`);
        this.reload();
      },
      error: (err: unknown) => {
        this.formSubmitting.set(false);
        this.formError.set(errorText(err, 'Failed to update flag'));
      },
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

  formatResultValue(result: EvaluationResult | null): string {
    if (!result) {
      return '';
    }
    if (typeof result.value === 'string') {
      return result.value;
    }
    return JSON.stringify(result.value, null, 2);
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
      return;
    }
    const raw = String(this.form.defaultValue).trim();
    if (raw === '' || raw === '[object Object]') {
      this.form.defaultValue = '{}';
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

  private payloadOrShowError(includeVersion = false): Record<string, unknown> | null {
    const rules = parseRules(this.form.rulesJson);
    if (!rules.ok) {
      this.formError.set(rules.message);
      return null;
    }
    const defaultValue = this.form.valueType === 'boolean' ? this.form.defaultValue === 'true' : parseJSON(this.form.defaultValue, invalidJSON);
    if (defaultValue === invalidJSON) {
      this.formError.set('Default value must be a valid JSON literal when value type is JSON.');
      return null;
    }
    const payload: Record<string, unknown> = {
      key: this.form.key,
      name: this.form.name,
      description: this.form.description,
      enabled: this.form.enabled,
      clientSide: this.form.clientSide,
      value: {
        type: this.form.valueType,
        default: defaultValue,
      },
      rules: rules.value,
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

const invalidJSON = Symbol('invalid-json');

/** Serialize API default for the edit form (avoids `[object Object]` for JSON flags). */
function defaultValueToFormString(raw: unknown, valueType: string): string {
  if (valueType === 'json') {
    if (raw === undefined || raw === null) {
      return '{}';
    }
    if (typeof raw === 'string') {
      const trimmed = raw.trim();
      if (trimmed === '' || trimmed === '[object Object]') {
        return '{}';
      }
      try {
        JSON.parse(trimmed);
        return trimmed;
      } catch {
        return JSON.stringify(raw);
      }
    }
    return JSON.stringify(raw, null, 2);
  }
  if (typeof raw === 'boolean') {
    return raw ? 'true' : 'false';
  }
  return String(raw).trim().toLowerCase() === 'true' ? 'true' : 'false';
}

function parseJSON(value: string, fallback: unknown): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return fallback;
  }
}

function parseRules(value: string): { ok: true; value: unknown[] } | { ok: false; message: string } {
  const parsed = parseJSON(value, invalidJSON);
  if (parsed === invalidJSON) {
    return { ok: false, message: 'Targeting rules must be valid JSON. Use [] when no targeting is needed.' };
  }
  if (!Array.isArray(parsed)) {
    return { ok: false, message: 'Targeting rules must be a JSON array. Use [] when no targeting is needed.' };
  }
  const badIndex = parsed.findIndex((rule) => rule === null || typeof rule !== 'object' || Array.isArray(rule));
  if (badIndex >= 0) {
    return { ok: false, message: `Targeting rule #${badIndex + 1} must be an object, not a string or primitive value.` };
  }
  return { ok: true, value: parsed };
}

function errorText(err: unknown, fallback: string): string {
  if (err && typeof err === 'object' && 'message' in err) {
    const msg = String((err as { message?: unknown }).message ?? '').trim();
    if (msg) {
      return msg;
    }
  }
  return fallback;
}
