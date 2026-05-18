export interface ApiErrorBody {
  error: string;
  message: string;
  details?: unknown;
  correlationId?: string;
  timestamp?: string;
}

export interface ApiEnvelope {
  correlationId?: string;
}

export interface SystemStatus extends ApiEnvelope {
  appVersion?: string;
  database?: {
    state: DatabaseState;
    connected: boolean;
    empty: boolean;
    currentSchemaVersion?: number;
    targetSchemaVersion?: number;
    minimumCompatibleSchemaVersion?: number;
    blockingBelowSchemaVersion?: number;
    dirty?: boolean;
    canInstall?: boolean;
    canMigrate?: boolean;
    runtimeReadCompatible?: boolean;
  };
  message?: string;
}

export type DatabaseState =
  | 'db_unreachable'
  | 'db_empty'
  | 'db_ready'
  | 'migration_optional'
  | 'migration_required'
  | 'db_ahead'
  | 'db_invalid';

export interface CSRFContract {
  headerName: string;
  token: string;
  expiresAt: string;
}

export interface CSRFResponse extends ApiEnvelope {
  csrf: CSRFContract;
}

export interface Project {
  ID?: string;
  id?: string;
  Key?: string;
  key?: string;
  Name?: string;
  name?: string;
  Description?: string | null;
  description?: string | null;
}

export interface EnvironmentRow {
  ID?: string;
  id?: string;
  ProjectID?: string;
  projectId?: string;
  Key?: string;
  key?: string;
  Name?: string;
  name?: string;
  SortOrder?: number;
  sortOrder?: number;
}

export interface Flag {
  ID?: string;
  id?: string;
  Key?: string;
  key?: string;
  Name?: string;
  name?: string;
  Description?: string;
  description?: string;
  Enabled?: boolean;
  enabled?: boolean;
  ClientSide?: boolean;
  clientSide?: boolean;
  Version?: number;
  version?: number;
  Value?: FlagValue;
  value?: FlagValue;
  Rules?: FlagRule[];
  rules?: FlagRule[];
  Rollout?: RolloutConfig | null;
  rollout?: RolloutConfig | null;
}

export interface FlagValue {
  Type?: string;
  type?: string;
  Default?: unknown;
  default?: unknown;
}

export interface FlagRule {
  id?: string;
  ID?: string;
  attribute?: string;
  Attribute?: string;
  operator?: string;
  Operator?: string;
  value?: unknown;
  Value?: unknown;
  serve?: unknown;
  Serve?: unknown;
  priority?: number;
  Priority?: number;
}

export interface RolloutConfig {
  Percentage?: number;
  percentage?: number;
  Stickiness?: string;
  stickiness?: string;
}

export interface Segment {
  ID?: string;
  id?: string;
  Key?: string;
  key?: string;
  Name?: string;
  name?: string;
  RulesJSON?: unknown;
  rules?: unknown;
  Version?: number;
  version?: number;
}

export interface AuditEntry {
  ID?: string;
  id?: string;
  EntityType?: string;
  entityType?: string;
  EntityID?: string;
  entityId?: string;
  Action?: string;
  action?: string;
  CorrelationID?: string | null;
  correlationId?: string | null;
  CreatedAt?: string;
  createdAt?: string;
  EntityKey?: string | null;
  entityKey?: string | null;
  EntityName?: string | null;
  entityName?: string | null;
}

export interface FlagVersion {
  ID?: string;
  id?: string;
  Version?: number;
  version?: number;
  ChangeType?: string;
  changeType?: string;
  CreatedAt?: string;
  createdAt?: string;
}

export interface EvaluationResult {
  value: unknown;
  version: number;
  reason: string;
  matchedRuleId?: string | null;
  source: string;
}

export interface DashboardBootstrap extends ApiEnvelope {
  status: SystemStatus | null;
  csrf: CSRFContract | null;
  app: {
    name: string;
    basePath: string;
  };
}

export interface Role {
  id?: string;
  key?: string;
  name?: string;
  description?: string;
  permissions?: string[];
  systemRole?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface DashboardUser {
  id?: string;
  email?: string;
  name?: string;
  roleId?: string;
  roleKey?: string;
  role?: string;
  status?: string;
  createdAt?: string;
}

export function rowID(row: { ID?: string; id?: string }): string {
  return row.id ?? row.ID ?? '';
}

export function rowKey(row: { Key?: string; key?: string }): string {
  return row.key ?? row.Key ?? '';
}

export function rowName(row: { Name?: string; name?: string; Key?: string; key?: string }): string {
  return row.name ?? row.Name ?? rowKey(row);
}
