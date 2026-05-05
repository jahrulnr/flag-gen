import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

@Component({
  selector: 'app-list-toolbar',
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './list-toolbar.component.html',
  styleUrl: './list-toolbar.component.scss',
})
export class ListToolbarComponent {
  /** Search query (two-way from parent). */
  readonly q = input<string>('');
  readonly qChange = output<string>();

  /** Pagination metadata from API. */
  readonly total = input<number>(0);
  readonly limit = input<number>(50);
  readonly offset = input<number>(0);
  readonly loading = input<boolean>(false);

  /** Optional placeholder for search. */
  readonly placeholder = input<string>('Search…');
  readonly showSearch = input<boolean>(true);

  readonly from = computed(() => Math.min(this.total(), this.total() === 0 ? 0 : this.offset() + 1));
  readonly to = computed(() => Math.min(this.total(), this.offset() + this.limit()));

  readonly canPrev = computed(() => this.offset() > 0);
  readonly canNext = computed(() => this.offset() + this.limit() < this.total());

  readonly prev = output<void>();
  readonly next = output<void>();
  readonly refresh = output<void>();

  onInput(e: Event): void {
    const v = (e.target as HTMLInputElement).value ?? '';
    this.qChange.emit(v);
  }
}

