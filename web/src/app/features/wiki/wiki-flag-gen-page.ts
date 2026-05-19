import { Component } from '@angular/core';

import { WIKI_PAGES } from '../../wiki/wiki-pages-content';
import { WikiArticlePageComponent } from './wiki-article-page';

@Component({
  selector: 'app-wiki-flag-gen-page',
  imports: [WikiArticlePageComponent],
  template: `<app-wiki-article-page [content]="content" wikiButtonId="wiki-flag-gen" />`,
  styleUrl: './wiki-shared.scss',
})
export class WikiFlagGenPage {
  readonly content = WIKI_PAGES['wiki-flag-gen'];
}
