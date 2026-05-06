import { CommonModule } from '@angular/common';
import { Component, OnInit, inject, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ApiService } from '../../core/api.service';
import { Role } from '../../core/models';
import { applyApiFailureToSignals } from '../../core/user-facing-api-error';
import { DestructiveConfirmDialogComponent } from '../../ui/destructive-confirm-dialog/destructive-confirm-dialog.component';
import { InlineApiErrorComponent } from '../../ui/inline-api-error/inline-api-error.component';
import { WikiButtonComponent } from '../../wiki/wiki-button.component';

@Component({
  selector: 'app-roles-page',
  imports: [
    CommonModule,
    FormsModule,
    WikiButtonComponent,
    InlineApiErrorComponent,
    DestructiveConfirmDialogComponent,
  ],
  templateUrl: './roles-page.html',
  styleUrl: './roles-page.scss',
})
export class RolesPage implements OnInit {
  private readonly api = inject(ApiService);
  readonly destroyDialog = viewChild.required(DestructiveConfirmDialogComponent);
  readonly roles = signal<Role[]>([]);
  readonly error = signal<string | null>(null);
  readonly errorRef = signal<string | null>(null);
  readonly loading = signal(false);

  readonly permissionOptions = [
    '*:read',
    '*:write',
    'flags:evaluate',
    'flags:read',
    'flags:write',
    'segments:*',
    'admin:*',
    '*',
  ];
  form = this.defaultForm();

  ngOnInit(): void {
    this.reload();
  }

  reload(): void {
    this.error.set(null);
    this.errorRef.set(null);
    this.loading.set(true);
    this.api.get<{ roles: Role[] }>('/roles').subscribe({
      next: (res) => {
        this.roles.set(res.roles ?? []);
        this.loading.set(false);
      },
      error: (err: unknown) => {
        this.loading.set(false);
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to load roles');
      },
    });
  }

  save(): void {
    const payload = {
      key: this.form.key,
      name: this.form.name,
      description: this.form.description,
      permissions: this.permissionsFromForm(),
    };
    const request = this.form.id
      ? this.api.patch(`/roles/${this.form.id}`, payload)
      : this.api.post('/roles', payload);
    request.subscribe({
      next: () => {
        this.form = this.defaultForm();
        this.reload();
      },
      error: (err: unknown) =>
        applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to save role'),
    });
  }

  edit(role: Role): void {
    this.form = {
      id: role.id ?? '',
      key: role.key ?? '',
      name: role.name ?? '',
      description: role.description ?? '',
      permissions: (role.permissions ?? []).join('\n'),
      systemRole: role.systemRole ?? false,
    };
  }

  remove(role: Role): void {
    if (role.systemRole) {
      this.error.set('System roles cannot be deleted.');
      return;
    }
    this.destroyDialog().open({
      title: 'Delete role',
      body: `Delete role "${role.name}" (${role.key}). Users assigned to this role must be moved first.`,
      expectedKey: role.key ?? '',
      confirmLabel: 'Delete role',
      onConfirm: () => {
        this.api.delete(`/roles/${role.id}`).subscribe({
          next: () => this.reload(),
          error: (err: unknown) =>
            applyApiFailureToSignals(this.error, this.errorRef, err, 'Failed to delete role'),
        });
      },
    });
  }

  cancel(): void {
    this.form = this.defaultForm();
  }

  addPermission(permission: string): void {
    const current = new Set(this.permissionsFromForm());
    current.add(permission);
    this.form.permissions = Array.from(current).join('\n');
  }

  permissions(role: Role): string[] {
    return role.permissions ?? [];
  }

  private permissionsFromForm(): string[] {
    return this.form.permissions
      .split(/\n|,/)
      .map((p) => p.trim())
      .filter(Boolean);
  }

  private defaultForm() {
    return { id: '', key: '', name: '', description: '', permissions: '*:read', systemRole: false };
  }
}
