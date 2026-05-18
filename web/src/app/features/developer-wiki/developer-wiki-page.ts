import { CommonModule } from '@angular/common';
import { Component, signal } from '@angular/core';

type GuideId = 'angular' | 'go';

interface Guide {
  id: GuideId;
  label: string;
  icon: string;
  summary: string;
  steps: string[];
  code: string;
}

const GUIDES: Guide[] = [
  {
    id: 'angular',
    label: 'Angular / Browser',
    icon: 'fa-brands fa-angular',
    summary: 'Use client-side HTTP reads only for flags marked clientSide. Payloads must be safe to expose in browser code.',
    steps: [
      'Create a public API key or leave PUBLIC_READ_API_KEY empty only in trusted local development.',
      'Read flags from /v1/client/projects/{projectId}/environments/{environmentKey}/flags.',
      'Use the single evaluate endpoint when you need targeting context such as userId.',
      'Keep secrets out of flag values because browser clients can inspect every response.',
    ],
    code: `const baseUrl = 'https://flags.example.com/v1';
const projectId = 'project-uuid';
const environmentKey = 'production';

async function evaluateCheckout(userId: string) {
  const res = await fetch(
    \`\${baseUrl}/client/projects/\${projectId}/environments/\${environmentKey}/flags/checkout_v2/evaluate?userId=\${encodeURIComponent(userId)}\`,
    { headers: { 'X-Public-Api-Key': 'public-key' } },
  );
  if (!res.ok) throw new Error(\`Flag evaluate failed: \${res.status}\`);
  const body = await res.json();
  return body.result.value === true;
}`,
  },
  {
    id: 'go',
    label: 'Go / Backend',
    icon: 'fa-brands fa-golang',
    summary: 'Backend services should subscribe over WebSocket for local read state and can use HTTP evaluation as fallback.',
    steps: [
      'Generate a scoped server or broadcast API key for the target project and environment.',
      'Use WebSocket /v1/broadcast/flags for initial flags.snapshot and later flag.updated or flag.deleted events.',
      'Keep local decisions version-aware and ignore stale updates.',
      'Use HTTP evaluate only for remote fallback or explicit debug checks.',
    ],
    code: `client := flagclient.New("https://flags.example.com/v1")
client.RuntimeKey = os.Getenv("FLAG_GEN_API_KEY")

sdk := flagclient.NewServerSDK(client, "project-uuid", "production", 30*time.Second)
sdk.StartWebSocket(ctx)
sdk.StartPolling(ctx) // optional fallback

enabled := sdk.Variation("checkout_v2", false).(bool)
if enabled {
    // run new path
}`,
  },
];

@Component({
  selector: 'app-developer-wiki-page',
  imports: [CommonModule],
  templateUrl: './developer-wiki-page.html',
  styleUrl: './developer-wiki-page.scss',
})
export class DeveloperWikiPage {
  readonly guides = GUIDES;
  readonly active = signal<GuideId>('angular');

  activeGuide(): Guide {
    return this.guides.find((guide) => guide.id === this.active()) ?? this.guides[0];
  }
}
