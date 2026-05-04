import { computed, Injectable, signal } from '@angular/core';

import { WikiContent, WikiId, WIKI_CONTENT } from './wiki-content';

@Injectable({ providedIn: 'root' })
export class WikiService {
  private readonly openState = signal(false);
  private readonly activeIdState = signal<WikiId>('projects');
  private readonly pageIdState = signal<WikiId>('projects');
  private readonly lastTriggerState = signal<HTMLElement | null>(null);

  readonly isOpen = this.openState.asReadonly();
  readonly activeId = this.activeIdState.asReadonly();
  readonly pageId = this.pageIdState.asReadonly();
  readonly lastTrigger = this.lastTriggerState.asReadonly();
  readonly activeContent = computed<WikiContent>(() => WIKI_CONTENT[this.activeIdState()]);
  readonly pageContent = computed<WikiContent>(() => WIKI_CONTENT[this.pageIdState()]);

  setPage(id: WikiId): void {
    this.pageIdState.set(id);
    if (!this.openState()) {
      this.activeIdState.set(id);
    }
  }

  open(id = this.pageIdState(), trigger?: HTMLElement): void {
    if (trigger) {
      this.lastTriggerState.set(trigger);
    }
    this.activeIdState.set(id);
    this.openState.set(true);
  }

  close(): void {
    this.openState.set(false);
  }

  toggle(id = this.pageIdState(), trigger?: HTMLElement): void {
    if (this.openState() && this.activeIdState() === id) {
      this.close();
      return;
    }
    this.open(id, trigger);
  }
}
