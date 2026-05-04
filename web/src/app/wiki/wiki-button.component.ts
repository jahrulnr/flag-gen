import { Component, Input, inject } from '@angular/core';

import { WikiId } from './wiki-content';
import { WikiService } from './wiki.service';

@Component({
  selector: 'app-wiki-button',
  template: `
    <button type="button" class="wiki-button" (click)="open($event)">
      <i class="fa-solid fa-book-open" aria-hidden="true"></i>
      Wiki
    </button>
  `,
  styleUrl: './wiki-button.component.scss',
})
export class WikiButtonComponent {
  private readonly wiki = inject(WikiService);

  @Input({ required: true }) wikiId!: WikiId;

  open(event: Event): void {
    this.wiki.open(this.wikiId, event.currentTarget as HTMLElement);
  }
}
