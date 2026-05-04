import { CommonModule } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  ViewChild,
  AfterViewChecked,
  inject,
  signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';

export interface DestructiveConfirmOptions {
  title: string;
  body: string;
  expectedKey: string;
  confirmLabel: string;
  keyFieldLabel?: string;
  onConfirm: () => void;
}

@Component({
  selector: 'app-destructive-confirm-dialog',
  imports: [CommonModule, FormsModule],
  templateUrl: './destructive-confirm-dialog.component.html',
  styleUrl: './destructive-confirm-dialog.component.scss',
  changeDetection: ChangeDetectionStrategy.Default,
})
export class DestructiveConfirmDialogComponent implements AfterViewChecked {
  private readonly host = inject(ElementRef);

  @ViewChild('dialogEl') private dialogEl?: ElementRef<HTMLDialogElement>;

  readonly state = signal<DestructiveConfirmOptions | null>(null);
  challengeInput = '';

  private pendingOpen = false;

  ngAfterViewChecked(): void {
    if (!this.pendingOpen || !this.dialogEl?.nativeElement) {
      return;
    }
    this.pendingOpen = false;
    this.dialogEl.nativeElement.showModal();
    this.focusChallenge();
  }

  focusChallenge(): void {
    const root = this.host.nativeElement as HTMLElement;
    queueMicrotask(() => {
      const input = root.querySelector<HTMLInputElement>('#destructive-key-challenge');
      input?.focus();
      input?.select();
    });
  }

  open(opts: DestructiveConfirmOptions): void {
    this.challengeInput = '';
    this.state.set(opts);
    this.pendingOpen = true;
  }

  close(): void {
    this.dialogEl?.nativeElement.close();
    this.state.set(null);
    this.challengeInput = '';
  }

  onDialogClose(): void {
    this.state.set(null);
    this.challengeInput = '';
  }

  cancel(): void {
    this.close();
  }

  submit(): void {
    const s = this.state();
    if (!s || !this.keyMatches()) {
      return;
    }
    const fn = s.onConfirm;
    this.close();
    fn();
  }

  keyMatches(): boolean {
    const s = this.state();
    return !!s && this.challengeInput === s.expectedKey;
  }
}
