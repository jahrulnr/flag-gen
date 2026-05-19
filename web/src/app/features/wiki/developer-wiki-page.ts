import { CommonModule } from '@angular/common';
import { Component, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { WikiButtonComponent } from '../../wiki/wiki-button.component';

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
    summary:
      'Gunakan route /v1/client/... hanya untuk flag clientSide. Payload harus aman jika dibaca di browser.',
    steps: [
      'Buat public API key atau kosongkan PUBLIC_READ_API_KEY hanya di local dev terpercaya.',
      'Baca flag: GET /v1/client/projects/{projectId}/environments/{environmentKey}/flags.',
      'Untuk targeting (userId, attributes): GET evaluate per flag atau POST evaluate batch.',
      'Jangan simpan secret di nilai flag — response bisa diinspeksi user.',
    ],
    code: `const baseUrl = 'https://flags.example.com/v1';
const projectId = 'project-uuid'; // dari halaman Projects
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
    summary:
      'Service backend subscribe WebSocket untuk definisi flag; evaluate HTTP untuk hasil per user.',
    steps: [
      'Salin Project ID (UUID) dari dashboard — bukan project key.',
      'Set FLAG_GEN_API_KEY (server) dan BROADCAST_DEV_TOKEN yang sama dengan server .env (local).',
      'StartWebSocket → terima flags.snapshot lalu flag.updated / flag.deleted.',
      'Untuk rules/rollout per user: EvaluateFlags — jangan andalkan Variation() saja.',
      'Contoh lengkap: examples/golang-service di repo.',
    ],
    code: `client := flagclient.New("http://localhost:6123/v1")
client.RuntimeKey = os.Getenv("FLAG_GEN_API_KEY")

sdk := flagclient.NewServerSDK(client, os.Getenv("FLAG_GEN_PROJECT_ID"), "development", 30*time.Second)
sdk.StartWebSocket(ctx) // butuh BROADCAST_DEV_TOKEN jika server mengaktifkannya

// Hanya untuk flag enabled + default — bukan targeting penuh:
enabled := sdk.Variation("checkout_v2", false).(bool)

// Targeting per user:
results, _ := client.EvaluateFlags(ctx, projectID, envKey, flagclient.EvaluateRequest{
  Context: flagclient.EvalContext{UserID: userID, Attributes: attrs},
  FlagKeys: []string{"checkout_v2"},
})`,
  },
];

@Component({
  selector: 'app-developer-wiki-page',
  imports: [CommonModule, RouterLink, WikiButtonComponent],
  templateUrl: './developer-wiki-page.html',
  styleUrl: './wiki-shared.scss',
})
export class DeveloperWikiPage {
  readonly guides = GUIDES;
  readonly active = signal<GuideId>('go');

  activeGuide(): Guide {
    return this.guides.find((guide) => guide.id === this.active()) ?? this.guides[0];
  }
}
