import { Injectable, inject } from '@angular/core';
import { ActivatedRouteSnapshot, NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs/operators';

import { WikiId } from './wiki-content';
import { WikiService } from './wiki.service';

@Injectable({ providedIn: 'root' })
export class WikiRouteService {
  private readonly router = inject(Router);
  private readonly wiki = inject(WikiService);

  bindToRouter(): void {
    this.sync(this.router.routerState.snapshot.root);
    this.router.events.pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd)).subscribe(() => {
      this.sync(this.router.routerState.snapshot.root);
    });
  }

  private sync(snapshot: ActivatedRouteSnapshot): void {
    const leaf = deepest(snapshot);
    const wikiId = leaf.data['wikiId'] as WikiId | undefined;
    if (wikiId) {
      this.wiki.setPage(wikiId);
    }
  }
}

function deepest(snapshot: ActivatedRouteSnapshot): ActivatedRouteSnapshot {
  let current = snapshot;
  while (current.firstChild) {
    current = current.firstChild;
  }
  return current;
}
