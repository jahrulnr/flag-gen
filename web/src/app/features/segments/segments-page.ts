import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { Segment, rowID, rowKey, rowName } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-segments-page',
  imports: [CommonModule, FormsModule, WikiButtonComponent],
  templateUrl: './segments-page.html',
  styleUrl: './segments-page.scss',
})
export class SegmentsPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly projectId = signal('');
  readonly segments = signal<Segment[]>([]);
  readonly error = signal<string | null>(null);
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
    this.api.get<{ segments: Segment[] }>(`/projects/${this.projectId()}/segments`).subscribe({
      next: (res) => this.segments.set(res.segments ?? []),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load segments'),
    });
  }

  create(): void {
    this.api.post(`/projects/${this.projectId()}/segments`, this.payload()).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to create segment'),
    });
  }

  update(): void {
    this.api.patch(`/projects/${this.projectId()}/segments/${this.form.id}`, this.payload()).subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to update segment'),
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
    this.api.delete(`/projects/${this.projectId()}/segments/${rowID(segment)}`).subscribe({
      next: () => this.reload(),
      error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to delete segment'),
    });
  }

  key(segment: Segment): string {
    return rowKey(segment);
  }

  name(segment: Segment): string {
    return rowName(segment);
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
