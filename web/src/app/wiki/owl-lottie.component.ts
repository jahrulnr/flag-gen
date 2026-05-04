import { CommonModule, isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit,
  ChangeDetectorRef,
  Component,
  ElementRef,
  Input,
  OnDestroy,
  PLATFORM_ID,
  ViewChild,
  inject,
} from '@angular/core';

import type { DotLottie } from '@lottiefiles/dotlottie-web';

@Component({
  selector: 'app-owl-lottie',
  imports: [CommonModule],
  template: `
    <div class="owl-lottie__viewport">
      <canvas
        #canvas
        class="owl-lottie__canvas"
        [class.owl-lottie__canvas--hidden]="loadFailed"
        [attr.aria-hidden]="decorative"
        [style.transform]="'scale(' + viewportZoom + ')'"
      ></canvas>
    </div>
    <span class="owl-lottie__fallback" aria-hidden="true">?</span>
  `,
  styleUrl: './owl-lottie.component.scss',
})
export class OwlLottieComponent implements AfterViewInit, OnDestroy {
  private readonly platformId = inject(PLATFORM_ID);
  private readonly cdr = inject(ChangeDetectorRef);
  private destroyed = false;
  private player: DotLottie | null = null;
  private motionQuery: MediaQueryList | null = null;
  private readonly onMotionChange = (): void => this.syncReducedMotion();

  @ViewChild('canvas', { static: true }) private readonly canvas?: ElementRef<HTMLCanvasElement>;
  @Input() decorative = true;

  /**
   * Lottie canvas is 512×512 with extra padding; ~400×400 is the owl — zoom = 512/400 crops ~56px per side.
   * Override if the asset framing changes.
   */
  @Input() viewportZoom = 512 / 200;

  /** DotLottie failed to fetch or parse — show textual fallback inside the FAB/drawer cell. */
  loadFailed = false;

  async ngAfterViewInit(): Promise<void> {
    if (!isPlatformBrowser(this.platformId) || !this.canvas) {
      return;
    }

    const { DotLottie } = await import('@lottiefiles/dotlottie-web');
    if (this.destroyed) {
      return;
    }
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const player = new DotLottie({
      autoplay: !reducedMotion,
      canvas: this.canvas.nativeElement,
      loop: !reducedMotion,
      src: 'assets/owl.lottie',
    });
    this.player = player;

    player.addEventListener('loadError', () => {
      if (this.destroyed) {
        return;
      }
      this.loadFailed = true;
      this.cdr.markForCheck();
    });

    player.addEventListener('load', () => {
      if (this.destroyed) {
        return;
      }
      this.syncReducedMotion();
    });

    this.motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    this.motionQuery.addEventListener('change', this.onMotionChange);
    this.syncReducedMotion();
  }

  ngOnDestroy(): void {
    this.destroyed = true;
    this.motionQuery?.removeEventListener('change', this.onMotionChange);
    this.motionQuery = null;
    this.player?.destroy();
    this.player = null;
  }

  private syncReducedMotion(): void {
    if (!this.player || this.loadFailed) {
      return;
    }
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduce) {
      this.player.setLoop(false);
      this.player.freeze();
    } else {
      this.player.unfreeze();
      this.player.setLoop(true);
      void this.player.play();
    }
  }
}
