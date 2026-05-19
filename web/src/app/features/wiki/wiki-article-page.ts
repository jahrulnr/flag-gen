import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { WikiPageContent } from '../../wiki/wiki-pages-content';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';
import type { WikiId } from '../../wiki/wiki-content';

@Component({
  selector: 'app-wiki-article-page',
  imports: [CommonModule, RouterLink, WikiButtonComponent],
  templateUrl: './wiki-article-page.html',
  styleUrl: './wiki-shared.scss',
  host: { class: 'wiki-article-host' },
})
export class WikiArticlePageComponent {
  @Input({ required: true }) content!: WikiPageContent;
  @Input() wikiButtonId?: WikiId;
}
