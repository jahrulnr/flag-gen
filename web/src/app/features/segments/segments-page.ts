import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';

import { ApiService } from '../../core/api.service';
import {
  effectiveSearchQuery,
  scheduleDebouncedSearch,
  SearchDebounceHandle,
} from '../../core/list-search';
import { Segment, rowID, rowKey, rowName } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { ProjectDirectoryService } from '../../core/project-directory.service';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { ListToolbarComponent } from '../../ui/list-toolbar/list-toolbar.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-segments-page',
  imports: [
    CommonModule,
    FormsModule,
    WikiButtonComponent,
    DestructiveConfirmDialogComponent,
    InlineApiErrorComponent,
    ListToolbarComponent,
  ],
  templateUrl: './segments-page.html',
  styleUrl: './segments-page.scss',
})
export class SegmentsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly projectDirectory = inject(ProjectDirectoryService);
  private readonly route = inject(ActivatedRoute);
  private searchDebounce: SearchDebounceHandle = null;
  readonly projectId = signal('');
  readonly segments = signal<Segment[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);
  readonly total = signal(0);
  readonly limit = signal(50);
  readonly offset = signal(0);
  readonly searchText = signal('');
  readonly loading = signal(false);
  readonly successMessage = signal<string | null>(null);

  form = this.defaultForm();

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
    this.error.set(null);
    this.errorRef.set(null);
    this.loading.set(true);
    const query = new URLSearchParams({
      q: effectiveSearchQuery(this.searchText()),
      limit: String(this.limit()),
      offset: String(this.offset()),
    });
    this.api.get<{ segments: Segment[]; total?: number; limit?: number; offset?: number }>(`/projects/${this.projectId()}/segments?${query.toString()}`).subscribe({
      next: (res) => {
        this.segments.set(res.segments ?? []);
        this.total.set(res.total ?? 0);
        this.limit.set(res.limit ?? this.limit());
        this.offset.set(res.offset ?? this.offset());
        this.loading.set(false);
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load segments'),
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
    this.error.set(null);
    this.errorRef.set(null);
    this.api.post(`/projects/${this.projectId()}/segments`, this.payload()).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to create segment'),
    });
  }

  update(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.api.patch(`/projects/${this.projectId()}/segments/${this.form.id}`, this.payload()).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to update segment'),
    });
  }

  edit(segment: Segment): void {
    this.form = {
      id: rowID(segment),
      key: rowKey(segment),
      name: rowName(segment),
      rulesJson: JSON.stringify(segment.rules ?? segment.RulesJSON ?? [], null, 2),
    };
  }

  remove(segment: Segment): void {
    const sid = rowID(segment);
    const k = rowKey(segment);
    const nm = rowName(segment);
    this.destroyDialog().open({
      title: 'Delete segment',
      body: `Remove segment "${nm}" (${k}) from this project. Targeting rules that reference it may break until recreated.`,
      expectedKey: k,
      confirmLabel: 'Delete segment permanently',
      onConfirm: () => {
        this.error.set(null);
        this.errorRef.set(null);
        this.api.delete(`/projects/${this.projectId()}/segments/${sid}`).subscribe({
          next: () => this.reload(),
          error: (err: unknown) => applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to delete segment'),
        });
      },
    });
  }

  key(segment: Segment): string {
    return rowKey(segment);
  }

  name(segment: Segment): string {
    return rowName(segment);
  }

  projectLabel(): string {
    return this.projectDirectory.displayName(this.projectId());
  }

  private payload(): Record<string, unknown> {
    return {
      key: this.form.key,
      name: this.form.name,
      rules: parseJSON(this.form.rulesJson, []),
    };
  }

  private defaultForm() {
    return { id: '', key: '', name: '', rulesJson: '[]' };
  }
}

function parseJSON(value: string, fallback: unknown): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return fallback;
  }
}
