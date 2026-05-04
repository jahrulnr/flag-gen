import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';

@Component({
  selector: 'app-inline-api-error',
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (message(); as msg) {
      <div class="inline-api-error" role="alert">
        <p class="inline-api-error__main">{{ msg }}</p>
        @if (reference(); as ref) {
          <p class="inline-api-error__ref">Report to admin · reference <code>{{ ref }}</code></p>
        }
      </div>
    }
  `,
  styles: `
    .inline-api-error {
      background: var(--danger-soft);
      border: 1px solid rgb(251 113 133 / 22%);
      border-radius: var(--radius-sm);
      color: #fecdd3;
      margin: 0;
      padding: 0.75rem 0.9rem;
    }

    .inline-api-error__main {
      margin: 0;
    }

    .inline-api-error__ref {
      color: rgb(148 163 184 / 88%);
      font-size: 0.78rem;
      margin: 0.55rem 0 0;
    }

    .inline-api-error__ref code {
      color: rgb(203 213 225 / 95%);
      font-size: inherit;
      word-break: break-all;
    }
  `,
})
export class InlineApiErrorComponent {
  readonly message = input<string | null>(null);
  /** Correlation ID or similar footer only */
  readonly reference = input<string | null>(null);
}
