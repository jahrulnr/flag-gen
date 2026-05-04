export type WikiId = 'lifecycle' | 'projects' | 'environments' | 'segments' | 'flags' | 'flag-history' | 'audit';

export interface WikiSection {
  title: string;
  bullets: string[];
}

export interface WikiContent {
  id: WikiId;
  eyebrow: string;
  title: string;
  summary: string;
  sections: WikiSection[];
}

export const WIKI_CONTENT: Record<WikiId, WikiContent> = {
  lifecycle: {
    id: 'lifecycle',
    eyebrow: 'Before you enter',
    title: 'How to prepare the database',
    summary: 'Use this page to check whether the service is installed, ready, or waiting for migrations.',
    sections: [
      {
        title: 'Recommended flow',
        bullets: [
          'Check the current status first; do not install or migrate blindly.',
          'Run install only for a fresh database that has not been bootstrapped.',
          'Run migrate during a planned window when the status says migrations are available.',
        ],
      },
      {
        title: 'What to watch',
        bullets: [
          'If the status shows a hard failure, fix database connectivity before trying again.',
          'Keep this page open after install or migrate so you can confirm the resulting state.',
        ],
      },
    ],
  },
  projects: {
    id: 'projects',
    eyebrow: 'Workspace setup',
    title: 'How to organize projects',
    summary: 'Projects group environments, segments, and flags for one product, service, or team-owned surface.',
    sections: [
      {
        title: 'Create a useful project',
        bullets: [
          'Use a stable key such as checkout or mobile_app; avoid temporary campaign names.',
          'Pick a readable name so operators can recognize ownership quickly.',
          'Open Environments after creating a project to define production, staging, or local contexts.',
        ],
      },
      {
        title: 'Cleanup guidance',
        bullets: [
          'Delete a project only after its environments, segments, and flags have been removed.',
          'Prefer archiving in process notes before deleting anything that may still be referenced by clients.',
        ],
      },
    ],
  },
  environments: {
    id: 'environments',
    eyebrow: 'Release contexts',
    title: 'How to manage environments',
    summary: 'Environments let the same flag behave differently in production, staging, QA, or local setups.',
    sections: [
      {
        title: 'Common setup',
        bullets: [
          'Create production first, then add staging or QA environments as needed.',
          'Use short lowercase keys like production, staging, or qa because they appear in URLs and SDK paths.',
          'Use sort order to keep the most important environments at the top.',
        ],
      },
      {
        title: 'Before deleting',
        bullets: [
          'Remove flags from the environment first.',
          'Confirm no SDK or backend job still requests that environment key.',
        ],
      },
    ],
  },
  segments: {
    id: 'segments',
    eyebrow: 'Targeting',
    title: 'How to define segments',
    summary: 'Segments describe reusable audience rules so flags do not duplicate targeting logic.',
    sections: [
      {
        title: 'Good segment rules',
        bullets: [
          'Name segments by audience intent, such as beta_users or internal_staff.',
          'Keep Rules JSON small and reviewable; complicated targeting is harder to debug.',
          'Edit a segment when the audience changes instead of creating several near-duplicates.',
        ],
      },
      {
        title: 'Operational tips',
        bullets: [
          'Check every flag that depends on a segment before deleting it.',
          'Use consistent attribute names with your SDK context payloads.',
        ],
      },
    ],
  },
  flags: {
    id: 'flags',
    eyebrow: 'Feature control',
    title: 'How to work with flags',
    summary: 'Flags control behavior per environment with defaults, rollout settings, and targeting rules.',
    sections: [
      {
        title: 'Safe rollout',
        bullets: [
          'Start with the default value that preserves current behavior.',
          'Enable client-side only for flags safe to expose to browsers or mobile clients.',
          'Increase rollout percentage gradually and keep stickiness stable, usually userId.',
        ],
      },
      {
        title: 'When editing',
        bullets: [
          'Use History after changes to confirm version and change type.',
          'Keep Rules JSON valid and leave it as [] when no targeting is needed.',
        ],
      },
    ],
  },
  'flag-history': {
    id: 'flag-history',
    eyebrow: 'Audit trail',
    title: 'How to read flag history',
    summary: 'Flag history shows versioned changes so operators can understand what changed and when.',
    sections: [
      {
        title: 'Reading versions',
        bullets: [
          'Use the newest version to confirm the current shape of the flag.',
          'Compare change types when investigating unexpected behavior.',
          'Match timestamps with deployment or incident timelines.',
        ],
      },
      {
        title: 'Rollback thinking',
        bullets: [
          'History is evidence; decide rollback by restoring the intended flag settings.',
          'Record the reason for manual fixes in the related incident or release note.',
        ],
      },
    ],
  },
  audit: {
    id: 'audit',
    eyebrow: 'Read model',
    title: 'How to search audit events',
    summary: 'Audit helps you inspect changes by project, entity type, and entity id.',
    sections: [
      {
        title: 'Search effectively',
        bullets: [
          'Project ID is required because audit events are scoped to a project.',
          'Filter by entity type such as flag, environment, or segment to narrow the result.',
          'Paste the exact entity id when investigating one object.',
        ],
      },
      {
        title: 'Use the result',
        bullets: [
          'Correlation IDs help connect API calls, logs, and user actions.',
          'If results are empty, confirm you are using the project UUID, not the project key.',
        ],
      },
    ],
  },
};
