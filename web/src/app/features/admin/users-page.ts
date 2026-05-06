import { CommonModule } from '@angular/common';
import { Component, OnInit, inject, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { DashboardUser, Role } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-users-page',
  imports: [
    CommonModule,
    FormsModule,
    WikiButtonComponent,
    InlineApiErrorComponent,
    DestructiveConfirmDialogComponent,
  ],
  templateUrl: './users-page.html',
  styleUrl: './users-page.scss',
})
export class UsersPage implements OnInit {
  private readonly api = inject(ApiService);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);
  readonly users = signal<DashboardUser[]>([]);
  readonly roles = signal<Role[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly loading = signal(false);

  form = this.defaultForm();

  ngOnInit(): void {
    this.reload();
  }

  reload(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.loading.set(true);
    this.api.get<{ users: DashboardUser[] }>('/users').subscribe({
      next: (res) => {
        this.users.set(res.users ?? []);
        this.loading.set(false);
      },
      error: (err: unknown) => {
        this.loading.set(false);
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load users');
      },
    });
    this.api.get<{ roles: Role[] }>('/roles').subscribe({
      next: (res) => this.roles.set(res.roles ?? []),
      error: (err: unknown) =>
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load roles'),
    });
  }

  save(): void {
    const role = this.roles().find((r) => r.id === this.form.roleId);
    const payload = {
      email: this.form.email,
      name: this.form.name,
      password: this.form.password,
      roleId: this.form.roleId,
      roleKey: role?.key ?? '',
      status: this.form.status,
    };
    const request = this.form.id
      ? this.api.patch(`/users/${this.form.id}`, payload)
      : this.api.post('/users', payload);
    request.subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) =>
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to save user'),
    });
  }

  edit(user: DashboardUser): void {
    this.form = {
      id: user.id ?? '',
      email: user.email ?? '',
      name: user.name ?? '',
      password: '',
      roleId: user.roleId ?? '',
      status: user.status ?? 'active',
    };
  }

  disable(user: DashboardUser): void {
    this.destroyDialog().open({
      title: 'Disable user',
      body: `Disable access for "${user.email}". Existing dashboard sessions will be rejected on the next request.`,
      expectedKey: user.email ?? '',
      confirmLabel: 'Disable user',
      onConfirm: () => {
        this.api.delete(`/users/${user.id}`).subscribe({
          next: () => this.reload(),
          error: (err: unknown) =>
            applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to disable user'),
        });
      },
    });
  }

  cancel(): void {
    this.form = this.defaultForm();
  }

  roleLabel(user: DashboardUser): string {
    return user.role ? `${user.role} (${user.roleKey})` : (user.roleKey ?? 'No role');
  }

  private defaultForm() {
    return { id: '', email: '', name: '', password: '', roleId: '', status: 'active' };
  }
}
