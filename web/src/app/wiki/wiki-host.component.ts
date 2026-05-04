import { CommonModule } from '@angular/common';
import { Component, ElementRef, HostListener, OnInit, ViewChild, effect, inject } from '@angular/core';

import { OwlLottieComponent } from './owl-lottie.component';
import { WikiRouteService } from './wiki-route.service';
import { WikiService } from './wiki.service';

@Component({
  selector: 'app-wiki-host',
  imports: [CommonModule, OwlLottieComponent],
  templateUrl: './wiki-host.component.html',
  styleUrl: './wiki-host.component.scss',
})
export class WikiHostComponent implements OnInit {
  private readonly routeService = inject(WikiRouteService);
  readonly wiki = inject(WikiService);

  @ViewChild('closeButton') private readonly closeButton?: ElementRef<HTMLButtonElement>;
  @ViewChild('drawer') private readonly drawer?: ElementRef<HTMLElement>;

  constructor() {
    effect(() => {
      if (this.wiki.isOpen()) {
        queueMicrotask(() => this.closeButton?.nativeElement.focus());
      }
    });
  }

  ngOnInit(): void {
    this.routeService.bindToRouter();
  }

  openFromFab(event: Event): void {
    this.wiki.open(this.wiki.pageId(), event.currentTarget as HTMLElement);
  }

  close(): void {
    this.wiki.close();
    queueMicrotask(() => this.wiki.lastTrigger()?.focus());
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.wiki.isOpen()) {
      this.close();
    }
  }

  @HostListener('document:keydown.tab', ['$event'])
  trapFocus(event: Event): void {
    if (!this.wiki.isOpen() || !this.drawer) {
      return;
    }
    const keyboardEvent = event as KeyboardEvent;
    const focusable = this.drawer.nativeElement.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])',
    );
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (!first || !last) {
      return;
    }
    if (keyboardEvent.shiftKey && document.activeElement === first) {
      keyboardEvent.preventDefault();
      last.focus();
    } else if (!keyboardEvent.shiftKey && document.activeElement === last) {
      keyboardEvent.preventDefault();
      first.focus();
    }
  }
}
