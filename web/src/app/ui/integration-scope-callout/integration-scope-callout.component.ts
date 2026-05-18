import { CommonModule } from '@angular/common';
import { Component, computed, inject, input } from '@angular/core';

import { ProjectDirectoryService } from '../../core/project-directory.service';

@Component({
  selector: 'app-integration-scope-callout',
  imports: [CommonModule],
  templateUrl: './integration-scope-callout.component.html',
  styleUrl: './integration-scope-callout.component.scss',
})
export class IntegrationScopeCalloutComponent {
  private readonly projectDirectory = inject(ProjectDirectoryService);

  /** Project UUID used in API paths and WebSocket query params. */
  readonly projectId = input.required<string>();
  /** Environment key (e.g. development, production) when on env-scoped pages. */
  readonly environmentKey = input<string>('');

  readonly projectKey = computed(() => this.projectDirectory.displayKey(this.projectId()));

  readonly showEnvironment = computed(() => (this.environmentKey() ?? '').trim().length > 0);
}
