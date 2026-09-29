import { test, expect, type Page } from '@playwright/test';

const DOC_ROUTES = [
  '/docs/getting-started',
  '/docs/ask-ai',
  '/docs/extensions',
  '/docs/settings',
  '/docs/writing-an-extension',
];

const PRERENDERED_ROUTES = ['/', ...DOC_ROUTES];

/**
 * Everything the browser itself reports: an uncaught exception, a
 * `console.error`, or a subresource that never arrived.
 *
 * This is the only place a broken client bundle shows up. The route still
 * answers 200 and the document still carries the prerendered markup, so
 * neither an HTTP probe nor a markup grep notices; the chunk throws in the
 * browser, the custom element never upgrades, and the page paints nothing.
 *
 * Every route reports zero of these on a clean build, in both targets, so an
 * empty list is the real baseline and not an accident of what we listen for.
 */
function collectBrowserErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(`uncaught: ${e.message}`));
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(`console.error: ${m.text()}`);
  });
  page.on('requestfailed', (r) => errors.push(`request failed: ${r.url()}`));
  return errors;
}

test('home renders page-home component', async ({ page }) => {
  await page.goto('/');
  await page.waitForSelector('page-home');
  await expect(page.locator('page-home')).toBeVisible();
});

/**
 * The landing page is the supernova recipe's: a hero, feature rows, steps and
 * keys, panes, a footer and a status line. Every one of them expanded on the
 * server, rather than reaching the reader as an empty tag.
 */
test('home renders the landing page sections', async ({ page }) => {
  await page.goto('/');
  await page.waitForSelector('page-home');
  const root = page.locator('page-home');
  await expect(root.locator('starlight-header')).toHaveCount(1);
  await expect(root.locator('litro-hero-nova')).toHaveCount(1);
  await expect(root.locator('.hero h1')).toBeVisible();
  await expect(root.locator('litro-feature-row')).toHaveCount(3);
  await expect(root.locator('litro-steps')).toHaveCount(1);
  await expect(root.locator('litro-key-hints')).toHaveCount(1);
  await expect(root.locator('litro-pane-grid')).toHaveCount(2);
  await expect(root.locator('litro-site-footer')).toHaveCount(1);
  await expect(root.locator('litro-status-line')).toHaveCount(1);
});

/**
 * The landing page must read with JavaScript off. This reads the HTML the
 * server sends directly, so nothing a client script does can make it pass.
 * Each line names content that only that component renders, so a bare tag
 * cannot satisfy it.
 */
test('landing page copy is in the server HTML', async ({ request }) => {
  const response = await request.get('/');
  expect(response.status()).toBe(200);
  const body = await response.text();

  expect(body).toContain('A launcher one key away, built the Unix way.');
  expect(body).toContain('brew install beatzball/tap/swoop');
  expect(body).toContain('/docs/getting-started');
  expect(body).toContain('Start the panel');             // litro-steps
  expect(body).toContain('<kbd>');                       // litro-key-hints
  expect(body).toContain('Time in Tokyo');               // litro-term-window
  expect(body).toContain('does the finding');            // litro-pane
  expect(body).toContain('class="site-title"');          // starlight-header
  expect(body).toContain('class="cell mode"');           // litro-status-line
  // The bird is slotted as the hero's mark. The recipe's own drawing stays in
  // the markup as the slot's fallback, which a slotted mark hides, so this
  // looks for the bird rather than for the fallback's absence.
  expect(body).toMatch(/<svg slot="mark"[^>]*>[\s\S]*?rotate\(-32\)/);
});

/**
 * With clipboard permission the copy button says "Copied". Without it, it
 * selects the command instead and says "Selected": a page must not claim a
 * copy it did not make.
 */
test('the install command copies, and says so', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('/');
  await page.waitForSelector('litro-outlet[data-litro-settled]');

  const button = page.locator('litro-install-command').first().locator('button');
  await button.click();
  await expect(button).toHaveText('Copied');
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    'brew install beatzball/tap/swoop',
  );
});

test('the install command selects the text when the clipboard is refused', async ({ page }) => {
  await page.goto('/');
  await page.waitForSelector('litro-outlet[data-litro-settled]');
  await page.evaluate(() => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
  });

  const button = page.locator('litro-install-command').first().locator('button');
  await button.click();
  await expect(button).toHaveText('Selected');
});

/** The header carries the logo, the site links and GitHub as an icon. */
test('the header carries the logo, the links and GitHub', async ({ page }) => {
  await page.goto('/');
  await page.waitForSelector('page-home');
  const header = page.locator('page-home starlight-header');

  await expect(header.locator('.site-title')).toHaveAttribute('href', '/');
  await expect(header.locator('.site-logo')).toHaveAttribute('src', '/logo.webp');
  await expect(header.locator('nav a[href="/docs/getting-started"]')).toHaveText('Docs');
  await expect(header.locator('a.github-link')).toHaveAttribute(
    'href',
    'https://github.com/beatzball/swoop',
  );
});

// `.first()` here and not on `page-home`: against the BUILT output
// `litro-outlet` briefly holds a second, hidden `page-docs-slug` alongside the
// prerendered one while it hydrates, so a bare locator trips Playwright's
// strict mode with "resolved to 2 elements". Checked with `--repeat-each=5`:
// only the docs route shows the duplicate, home never does. `litro dev` never
// shows it either, which is one more thing the preview target sees first.
test('/docs/getting-started renders', async ({ page }) => {
  await page.goto('/docs/getting-started');
  await page.waitForSelector('page-docs-slug');
  await expect(page.locator('page-docs-slug').first()).toBeVisible();
});

test('all prerendered routes return 200', async ({ request }) => {
  for (const route of PRERENDERED_ROUTES) {
    const response = await request.get(route);
    expect(response.status(), `Expected 200 for ${route}`).toBe(200);
  }
});

/**
 * Where a route's OWN content lives, so an assertion cannot be satisfied by
 * page furniture.
 *
 * A bare `h2` is not good enough. Playwright's CSS engine pierces open shadow
 * roots, and `starlight-toc` renders a fixed `<h2>On this page</h2>`
 * (src/components/starlight-toc.ts) on every doc page. That heading is
 * there whether or not the page has any content, so a bare `h2` check passes a
 * completely empty doc page. Doc content is slotted into a
 * `<div slot="content">` (pages/docs/[slug].ts); the home page has no such slot and no TOC, so its
 * own headings are the only ones inside `page-home`.
 *
 * Every doc page is required to start its body at `##` (see site/AGENTS.md), so
 * "the content slot contains a visible, non-empty <h2>" holds for all of them.
 */
function contentHeadingSelector(route: string): string {
  return route === '/' ? 'page-home h2' : '[slot="content"] h2';
}

test('every prerendered route paints its content', async ({ page }) => {
  for (const route of PRERENDERED_ROUTES) {
    await page.goto(route);
    // `:visible`, not merely present: an element that never upgraded is still
    // in the document, and so is every <h2> the prerender wrote. Visibility is
    // what separates a page that rendered from one that shipped its markup and
    // then painted nothing.
    const heading = page.locator(contentHeadingSelector(route)).first();
    await expect(heading, `${route} painted no visible content <h2>`).toBeVisible();
    await expect(heading, `${route} painted an empty content <h2>`).not.toHaveText('');
  }
});

test('every prerendered route hydrates without a browser error', async ({ page }) => {
  const errors = collectBrowserErrors(page);
  for (const route of PRERENDERED_ROUTES) {
    await page.goto(route, { waitUntil: 'networkidle' });
    // Hydration proper: the page component has to be defined. When the client
    // chunk throws on the way in, the registration never runs.
    const tag = route === '/' ? 'page-home' : 'page-docs-slug';
    await expect
      .poll(
        () => page.evaluate((t) => !!customElements.get(t), tag),
        { message: `${route} never defined <${tag}>, so it did not hydrate` },
      )
      .toBe(true);
  }
  expect(errors, 'the browser reported errors').toEqual([]);
});

// A docs URL ending in `/` used to paint nothing on the sibling sites: 200,
// every asset loaded, no console error, and the page element never defined,
// because the client router matches `/docs/:slug` exactly. app.ts and
// server/routes/[...].ts now canonicalize the path, and nginx.conf redirects
// it. So this asserts on what a reader would see, the page's own h1, which
// getByRole skips when the page is hidden.
//
// It waits for the outlet to settle first. Against the built output the outlet
// briefly holds the prerendered page and the hydrated one side by side, both
// visible, so an h1 lookup made in that moment finds two and trips strict
// mode. Once settled there is exactly one. A blank page still fails: settling
// does not make a hidden page visible.
test('/docs/getting-started/ with a trailing slash shows the page', async ({ page }) => {
  await page.goto('/docs/getting-started/');
  await page.waitForSelector('litro-outlet[data-litro-settled]');
  await expect(page.getByRole('heading', { level: 1, name: 'Getting Started' })).toBeVisible({
    timeout: 30_000,
  });
});
