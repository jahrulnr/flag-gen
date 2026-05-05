import {
  AngularNodeAppEngine,
  createNodeRequestHandler,
  isMainModule,
  writeResponseToNodeResponse,
} from '@angular/ssr/node';
import express from 'express';
import { join } from 'node:path';

const browserDistFolder = join(import.meta.dirname, '../browser');
const apiOrigin = process.env['API_ORIGIN'] || 'http://127.0.0.1:6123';

const app = express();
const angularApp = new AngularNodeAppEngine();

/**
 * Example Express Rest API endpoints can be defined here.
 * Uncomment and define endpoints as necessary.
 *
 * Example:
 * ```ts
 * app.get('/api/{*splat}', (req, res) => {
 *   // Handle API request
 * });
 * ```
 */

// Do not redirect GET / → /app/ here: `ng serve --serve-path=/app/` forwards the SPA entry as
// path `/` internally, and a 302 to `/app/` repeats forever (NS_ERROR_REDIRECT_LOOP).
// Open the app at `/app/` directly; standalone SSR uses the `/app` routes below.

app.use('/api', async (req, res, next) => {
  try {
    const upstream = await fetch(`${apiOrigin}${req.originalUrl}`, {
      method: req.method,
      headers: req.headers as HeadersInit,
      body: req.method === 'GET' || req.method === 'HEAD' ? undefined : req,
      duplex: req.method === 'GET' || req.method === 'HEAD' ? undefined : 'half',
    } as RequestInit & { duplex?: 'half' });

    res.status(upstream.status);
    // fetch() decompresses gzip/br bodies but may still expose upstream length/encoding headers;
    // forwarding those with the decoded body causes "Content Encoding Error" in browsers.
    const skipHeader = new Set(
      [
        'content-encoding',
        'content-length',
        'transfer-encoding',
        'connection',
        'keep-alive',
      ].map((k) => k.toLowerCase()),
    );
    upstream.headers.forEach((value, key) => {
      if (skipHeader.has(key.toLowerCase())) return;
      res.setHeader(key, value);
    });
    const body = Buffer.from(await upstream.arrayBuffer());
    res.send(body);
  } catch {
    res.status(502).json({
      error: 'backend_unavailable',
      message: 'Go API backend is unavailable',
    });
  }
});

function renderAngular(
  req: express.Request,
  res: express.Response,
  next: express.NextFunction,
): void {
  angularApp
    .handle(req)
    .then(async (response) => {
      if (!response) {
        next();
        return;
      }
      const withBootstrap = await injectBootstrap(req, res, response);
      writeResponseToNodeResponse(withBootstrap, res);
    })
    .catch(next);
}

async function injectBootstrap(
  req: express.Request,
  res: express.Response,
  response: Response,
): Promise<Response> {
  const contentType = response.headers.get('content-type') ?? '';
  if (!contentType.includes('text/html')) {
    return response;
  }
  const html = await response.text();
  if (html.includes('id="flag-gen-bootstrap"')) {
    return new Response(html, responseInit(response));
  }
  try {
    const csrfRes = await fetch(`${apiOrigin}/api/security/csrf`, {
      headers: {
        cookie: req.headers.cookie ?? '',
        'x-forwarded-proto': req.protocol,
      },
    });
    if (!csrfRes.ok) {
      return new Response(html, responseInit(response));
    }
    const setCookie = csrfRes.headers.get('set-cookie');
    if (setCookie) {
      res.setHeader('set-cookie', setCookie);
    }
    const bootstrap = await csrfRes.json();
    const script = `<script id="flag-gen-bootstrap" type="application/json">${JSON.stringify(bootstrap).replaceAll('</', '<\\/')}</script>`;
    return new Response(html.replace('</head>', `${script}</head>`), responseInit(response));
  } catch {
    return new Response(html, responseInit(response));
  }
}

function responseInit(response: Response): ResponseInit {
  return {
    status: response.status,
    statusText: response.statusText,
    headers: response.headers,
  };
}

/**
 * Serve static files from /browser under the same base path used by Go.
 */
app.use(
  '/app',
  express.static(browserDistFolder, {
    maxAge: '1y',
    index: false,
    redirect: false,
  }),
);

/**
 * SSR under /app (production Node + Go proxy).
 */
app.use('/app', renderAngular);

/**
 * `ng serve --serve-path=/app/` forwards the document URL as path `/` internally, while
 * public URLs stay under `/app/`. Mount SSR at `/` too so the entry page is not 404.
 * Registered after `/api` and `/app` so `/api/*` and `/app/*` stay specific.
 */
app.use('/', renderAngular);

/**
 * Start the server if this module is the main entry point, or it is ran via PM2.
 * The server listens on the port defined by the `PORT` environment variable, or defaults to 4000.
 */
if (isMainModule(import.meta.url) || process.env['pm_id']) {
  const port = process.env['PORT'] || 4000;
  app.listen(port, (error) => {
    if (error) {
      throw error;
    }

    console.log(`Node Express server listening on http://localhost:${port}`);
  });
}

/**
 * Request handler used by the Angular CLI (for dev-server and during build) or Firebase Cloud Functions.
 */
export const reqHandler = createNodeRequestHandler(app);
