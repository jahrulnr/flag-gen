import { JsonPipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ClientFlag, ClientFlagsResponse, EvaluateResponse, FlagGenService } from './flag-gen.service';
import { environment } from '../environments/environment';

@Component({
  selector: 'app-root',
  imports: [FormsModule, JsonPipe],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {
  private readonly flagsApi = inject(FlagGenService);

  readonly config = environment;

  // Sample 1: list flags
  readonly flagsLoading = signal(false);
  readonly flagsResponse = signal<ClientFlagsResponse | null>(null);

  // Sample 2: evaluate
  readonly evaluateLoading = signal(false);
  readonly evaluateResponse = signal<EvaluateResponse | null>(null);

  readonly error = signal<string | null>(null);

  evaluateFlagKey = '';
  userId = 'user-demo-1';
  attributesJson = '{\n  "country": "ID"\n}';

  loadFlags(): void {
    this.flagsLoading.set(true);
    this.error.set(null);
    this.flagsApi.listClientFlags().subscribe({
      next: (res) => {
        this.flagsResponse.set(res);
        this.flagsLoading.set(false);
        const first = res.flags?.[0];
        if (first && !this.evaluateFlagKey) {
          this.evaluateFlagKey = first.key;
        }
      },
      error: (err: Error) => {
        this.error.set(err.message ?? String(err));
        this.flagsLoading.set(false);
      },
    });
  }

  runEvaluate(): void {
    const key = this.evaluateFlagKey.trim();
    if (!key) {
      return;
    }
    let attributes: Record<string, unknown> | null = null;
    const raw = this.attributesJson.trim();
    if (raw) {
      try {
        const parsed: unknown = JSON.parse(raw);
        if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
          this.error.set('Attributes must be a JSON object.');
          return;
        }
        attributes = parsed as Record<string, unknown>;
      } catch {
        this.error.set('Attributes must be valid JSON.');
        return;
      }
    }
    this.evaluateLoading.set(true);
    this.error.set(null);
    this.evaluateResponse.set(null);
    this.flagsApi.evaluate([key], this.userId, attributes).subscribe({
      next: (res) => {
        this.evaluateResponse.set(res);
        this.evaluateLoading.set(false);
      },
      error: (err: Error) => {
        this.error.set(err.message ?? String(err));
        this.evaluateLoading.set(false);
      },
    });
  }

  flagRows(): ClientFlag[] {
    return this.flagsResponse()?.flags ?? [];
  }
}
