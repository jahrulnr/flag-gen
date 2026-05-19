import { Routes } from '@angular/router';

import { RolesPage } from './features/admin/roles-page';
import { UsersPage } from './features/admin/users-page';
import { authGuard } from './core/auth.guard';
import { AuditPage } from './features/audit/audit-page';
import { FlagHistoryPage } from './features/audit/flag-history-page';
import { LoginPage } from './features/auth/login-page';
import { DashboardShell } from './features/dashboard/dashboard-shell';
import { DeveloperWikiPage } from './features/wiki/developer-wiki-page';
import { WikiFlagGenPage } from './features/wiki/wiki-flag-gen-page';
import { WikiGuidePage } from './features/wiki/wiki-guide-page';
import { EnvironmentsPage } from './features/environments/environments-page';
import { FlagsPage } from './features/flags/flags-page';
import { LifecyclePage } from './features/lifecycle/lifecycle-page';
import { ProjectsPage } from './features/projects/projects-page';
import { SegmentsPage } from './features/segments/segments-page';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'login' },
  { path: 'login', component: LoginPage, data: { wikiId: 'lifecycle' } },
  { path: 'lifecycle', component: LifecyclePage, data: { wikiId: 'lifecycle' } },
  {
    path: 'dashboard',
    component: DashboardShell,
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'projects' },
      { path: 'projects', component: ProjectsPage, data: { wikiId: 'projects' } },
      {
        path: 'projects/:projectId/environments',
        component: EnvironmentsPage,
        data: { wikiId: 'environments' },
      },
      {
        path: 'projects/:projectId/environments/:environmentKey/flags',
        component: FlagsPage,
        data: { wikiId: 'flags' },
      },
      {
        path: 'projects/:projectId/environments/:environmentKey/flags/:flagId/history',
        component: FlagHistoryPage,
        data: { wikiId: 'flag-history' },
      },
      {
        path: 'projects/:projectId/segments',
        component: SegmentsPage,
        data: { wikiId: 'segments' },
      },
      { path: 'audit', component: AuditPage, data: { wikiId: 'audit' } },
      { path: 'wiki', pathMatch: 'full', redirectTo: 'wiki/guide' },
      { path: 'wiki/guide', component: WikiGuidePage, data: { wikiId: 'wiki-guide' } },
      { path: 'wiki/flag-gen', component: WikiFlagGenPage, data: { wikiId: 'wiki-flag-gen' } },
      { path: 'wiki/developer', component: DeveloperWikiPage, data: { wikiId: 'developer-wiki' } },
      { path: 'developer-wiki', redirectTo: 'wiki/developer' },
      { path: 'users', component: UsersPage, data: { wikiId: 'users' } },
      { path: 'roles', component: RolesPage, data: { wikiId: 'roles' } },
    ],
  },
  { path: '**', redirectTo: 'lifecycle' },
];
