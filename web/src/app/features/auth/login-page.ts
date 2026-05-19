import { CommonModule } from '@angular/common';
import { Component, inject, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

import { AuthService } from '../../core/auth.service';

@Component({
  selector: 'app-login-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './login-page.html',
  styleUrl: './login-page.scss',
})
export class LoginPage implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  email = 'admin@local';
  password = '';
  error = '';

  ngOnInit(): void {
    this.auth.ensureDashboardAccess().subscribe((ok) => {
      if (ok) {
        void this.router.navigateByUrl('/dashboard/projects');
      }
    });
  }

  submit(): void {
    if (!this.email.trim() || !this.password.trim()) {
      this.error = 'Email and password are required.';
      return;
    }
    this.error = '';
    this.auth.login(this.email, this.password).subscribe({
      next: () => void this.router.navigateByUrl('/dashboard/projects'),
      error: (err: { message?: string }) => {
        this.error = err.message ?? 'Sign in failed.';
      },
    });
  }
}
