import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { ApiService } from '../../core/api.service';
import { FlagVersion } from '../../core/models';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-flag-history-page',
  imports: [CommonModule, WikiButtonComponent],
  templateUrl: './flag-history-page.html',
  styleUrl: './flag-history-page.scss',
})
export class FlagHistoryPage implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly versions = signal<FlagVersion[]>([]);
  readonly error = signal<string | null>(null);

  projectId = '';
  environmentKey = '';
  flagId = '';

  ngOnInit(): void {
    this.route.paramMap.subscribe((params) => {
      this.projectId = params.get('projectId') ?? '';
      this.environmentKey = params.get('environmentKey') ?? '';
      this.flagId = params.get('flagId') ?? '';
      this.reload();
    });
  }

  reload(): void {
    this.api
      .get<{ versions: FlagVersion[] }>(`/projects/${this.projectId}/environments/${this.environmentKey}/flags/${this.flagId}/versions`)
      .subscribe({
        next: (res) => this.versions.set(res.versions ?? []),
        error: (err: { message?: string }) => this.error.set(err.message ?? 'Failed to load versions'),
      });
  }
}
