import { Component } from '@angular/core';

import { WIKI_PAGES } from '../../wiki/wiki-pages-content';
import { WikiArticlePageComponent } from './wiki-article-page';

@Component({
  selector: 'app-wiki-guide-page',
  imports: [WikiArticlePageComponent],
  template: `<app-wiki-article-page [content]="content" wikiButtonId="wiki-guide" />`,
  styleUrl: './wiki-shared.scss',
})
export class WikiGuidePage {
  readonly content = WIKI_PAGES['wiki-guide'];
}
