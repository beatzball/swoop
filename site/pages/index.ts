import { css, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { LitroPage } from '@beatzball/litro/runtime';
import { definePageData } from '@beatzball/litro';
import { getGlobalData } from 'litro:content';
import { siteConfig } from '../server/starlight.config.js';
import { starlightHead } from '../src/route-meta.js';
import { buildSeoHead, buildSeoTitle } from '../src/seo.js';

// Register components used in render()
import '../src/components/starlight-header.js';
import '../src/components/litro-card.js';
import '../src/components/litro-card-grid.js';
import '../src/components/litro-footer.js';

/**
 * Install, then the first thing to run. Kept in step with the Install section
 * of content/docs/getting-started.md and of the README.
 */
const INSTALL_STEPS = [
  { note: 'with Homebrew, on a Mac or on Linux', cmd: 'brew install beatzball/tap/swoop' },
  { note: 'Mac: the panel at login, on alt+shift+space', cmd: 'brew services start swoop' },
  {
    note: 'or with nothing but curl',
    cmd: 'curl -fsSL https://raw.githubusercontent.com/beatzball/swoop/main/scripts/get | bash',
  },
] as const;

/**
 * What to type. One line each, in the order someone meets them. Every row
 * links to the page that covers it, so this doubles as the table of contents.
 */
const MODES = [
  { cmd: 'saf', what: 'Apps, filtered as you type. Enter opens.', href: '/docs/getting-started' },
  { cmd: '2+2', what: 'The answer is a row. Enter copies it.', href: '/docs/extensions#calculator' },
  { cmd: 'wiki', what: 'Quicklinks: a name, a link, and what opens it.', href: '/docs/extensions#quicklinks' },
  { cmd: 'notes', what: 'Markdown notes, edited inside the panel.', href: '/docs/extensions#notes' },
  { cmd: 'Tab', what: 'Ask AI, with any model you like.', href: '/docs/ask-ai' },
  { cmd: 'ctrl-k', what: 'Everything else a row can do.', href: '/docs/extensions' },
] as const;

/** The two tools that do the heavy lifting. swoop is the wiring. */
const TOOLS = [
  { name: 'fzf', job: 'does the finding', href: 'https://github.com/junegunn/fzf' },
  { name: 'libghostty', job: 'does the drawing', href: 'https://github.com/ghostty-org/ghostty' },
  { name: 'your programs', job: 'print the lines', href: '/docs/writing-an-extension' },
] as const;

export interface SplashData {
  siteTitle: string;
  description: string;
  nav: Array<{ label: string; href: string }>;
  features: Array<{ title: string; description: string; icon?: string }>;
  /**
   * Raw <head> HTML. Litro injects this and strips it from the JSON payload
   * before serializing — it contains no </script>, but the framework treats
   * the key specially regardless. See src/seo.ts.
   */
  seoHead: string;
  /** Overrides routeMeta.title, which cannot vary per request. */
  seoTitle: string;
}

export const pageData = definePageData(async (_event) => {
  const metadata = await getGlobalData();
  const siteTitle = String(metadata.title ?? siteConfig.title);
  const description = String(metadata.description ?? siteConfig.description);

  return {
    siteTitle,
    description,
    seoTitle: buildSeoTitle(siteTitle),
    seoHead: buildSeoHead({ title: siteTitle, description, path: '/' }),
    nav: siteConfig.nav,
    // Six, not five or seven. The grid is auto-fit at a 16rem minimum inside a
    // 56rem column, which resolves to three across at every width that fits
    // more than one — so only a multiple of three fills its last row.
    features: [
      {
        icon: '🐦',
        title: 'One key away',
        description: 'alt+shift+space shows a floating panel on a Mac. Esc, or a click elsewhere, and it is gone.',
      },
      {
        icon: '🧩',
        title: 'Extensions are programs',
        description: 'Anything that prints tab-separated lines. A shell script is enough, in any language that starts fast.',
      },
      {
        icon: '🤖',
        title: 'Any model',
        description: 'Tab asks the AI you name: a local model, a hosted one, or any command that reads a question and prints an answer.',
      },
      {
        icon: '⌨️',
        title: 'Terminal first',
        description: 'Notes and tasks open in your editor inside the panel. The same launcher runs in any terminal, on any OS.',
      },
      {
        icon: '📄',
        title: 'Plain files',
        description: 'Settings, snippets, quicklinks, notes and tasks are text files you can edit, grep and sync.',
      },
      {
        icon: '🔒',
        title: 'Yours alone',
        description: 'Clipboard history skips password managers. Usage stays on your machine. Nothing is sent anywhere you did not pick.',
      },
    ],
  } satisfies SplashData;
});

export const routeMeta = {
  head: starlightHead,
  title: 'swoop',
};

@customElement('page-home')
export class SplashPage extends LitroPage {
  /**
   * Local workaround for beatzball/litro#137 — remove once the recipe ships
   * its own reset.
   *
   * The `box-sizing: border-box` reset lives in public/styles/starlight.css,
   * which is a document stylesheet and so does not cross into this
   * component's shadow root. Without it the <main> below computes as
   * content-box: `width:100%` resolves to the full viewport and the 1.5rem
   * side padding is added on top, putting the page 48px wider than the screen
   * on every phone. Invisible above ~900px, where `max-width:56rem` caps
   * <main> before the padding can matter.
   *
   * litro.dev works around the same bug by dropping the horizontal padding
   * from <main> entirely. That trades the overflow for content sitting flush
   * against the screen edge; resetting box-sizing keeps the gutters.
   */
  static override styles = css`
    :host {
      display: block;
    }

    *,
    *::before,
    *::after {
      box-sizing: border-box;
    }
  `;

  override render() {
    const data = this.serverData as SplashData | null;
    const { siteTitle = 'swoop', description = '', nav = [], features = [] } = data ?? {};

    return html`
      <div style="min-height:100vh;display:flex;flex-direction:column;">
        <starlight-header
          siteTitle="${siteTitle}"
          .nav="${nav}"
          currentPath="/"
        ></starlight-header>
        <main style="
          flex:1;
          max-width:56rem;
          margin:0 auto;
          padding:4rem 1.5rem 3rem;
          width:100%;
        ">
          <section style="text-align:center;margin-bottom:3.5rem;">
            <img
              src="/logo.webp"
              alt=""
              width="160"
              height="160"
              style="
                display:block;
                margin:0 auto 1.5rem;
                width:clamp(96px,18vw,160px);
                height:auto;
              "
            />
            <h1 style="
              font-size:clamp(2rem,5vw,3.5rem);
              font-weight:800;
              color:var(--sl-color-text);
              margin:0 0 1rem;
              line-height:1.1;
            ">${siteTitle}</h1>
            ${description ? html`
              <p style="
                font-size:var(--sl-text-xl);
                color:var(--sl-color-gray-4);
                max-width:36rem;
                margin:0 auto 2.5rem;
                line-height:1.6;
              ">${description}</p>
            ` : ''}
            <div style="display:flex;gap:1rem;justify-content:center;flex-wrap:wrap;">
              <a href="/docs/getting-started" style="
                display:inline-block;
                padding:0.6rem 1.5rem;
                background:var(--sl-color-accent);
                color:var(--sl-color-text-invert,#fff);
                border-radius:var(--sl-border-radius);
                font-weight:600;
                text-decoration:none;
                font-size:var(--sl-text-base);
              ">Get Started</a>
              <a href="https://github.com/beatzball/swoop" style="
                display:inline-block;
                padding:0.6rem 1.5rem;
                border:1px solid var(--sl-color-border);
                color:var(--sl-color-text);
                border-radius:var(--sl-border-radius);
                font-weight:600;
                text-decoration:none;
                font-size:var(--sl-text-base);
              ">GitHub</a>
            </div>
            <!-- The name is the joke, so it lands after the reader has already
                 decided whether to click. The README opens with it. -->
            <p style="
              font-size:var(--sl-text-base);
              color:var(--sl-color-gray-4);
              max-width:36rem;
              margin:1.75rem auto 0;
              line-height:1.6;
              font-style:italic;
            ">Say it like a bird does: swoop down, grab the thing, gone.</p>
          </section>

          <!-- The recording carries the whole pitch faster than any paragraph.
               Kept in step with demo/swoop.tape — re-check the alt text when
               that tape changes what it records, and rebuild public/swoop.webp
               from demo/swoop.gif (see site/AGENTS.md). -->
          <section style="margin-bottom:3.5rem;">
            <img
              src="/swoop.webp"
              alt="swoop: filtering apps, the action menu, the calculator, and Ask AI answering a question"
              width="1000"
              height="560"
              loading="lazy"
              decoding="async"
              style="
                display:block;
                width:100%;
                /* Required, because of the width/height attributes above.
                   Those reserve the right box before the image loads, but the
                   width:100% here overrides only the width half of the pair.
                   Without this line the height attribute stands and the
                   recording is squashed to 560px tall at every viewport. */
                height:auto;
                max-width:48rem;
                margin:0 auto;
                border:1px solid var(--sl-color-border);
                border-radius:var(--sl-border-radius);
              "
            />
          </section>

          <!-- swoop finds and draws nothing itself. Saying so up front is the
               honest pitch, and it tells a reader what they are installing. -->
          <section style="margin-bottom:3.5rem;">
            <h2 style="
              font-size:var(--sl-text-xl);
              font-weight:700;
              color:var(--sl-color-text);
              margin:0 0 0.5rem;
              text-align:center;
            ">Built the Unix way</h2>
            <p style="
              text-align:center;
              color:var(--sl-color-gray-4);
              font-size:var(--sl-text-sm);
              margin:0 0 1.5rem;
            ">Small programs, one job each. swoop is the wiring.</p>
            <div style="
              display:grid;
              grid-template-columns:repeat(auto-fit,minmax(13rem,1fr));
              gap:0.75rem;
              max-width:44rem;
              margin:0 auto;
            ">
              ${TOOLS.map(
                (t) => html`
                  <a href="${t.href}" style="
                    padding:1rem;
                    border:1px solid var(--sl-color-border);
                    border-radius:var(--sl-border-radius);
                    display:flex;
                    flex-direction:column;
                    gap:0.35rem;
                    text-decoration:none;
                  ">
                    <span style="
                      font-family:var(--sl-font-mono,ui-monospace,monospace);
                      font-weight:700;
                      color:var(--sl-color-text-accent,var(--sl-color-accent));
                      font-size:var(--sl-text-base);
                    ">${t.name}</span>
                    <span style="
                      color:var(--sl-color-gray-4);
                      font-size:var(--sl-text-sm);
                      line-height:1.5;
                    ">${t.job}</span>
                  </a>
                `,
              )}
            </div>
          </section>

          <!-- Install and first run, so the landing page answers "how do I
               start" without a click. -->
          <section style="margin-bottom:3.5rem;">
            <h2 style="
              font-size:var(--sl-text-xl);
              font-weight:700;
              color:var(--sl-color-text);
              margin:0 0 1rem;
              text-align:center;
            ">Get running</h2>
            <div style="
              background:var(--sl-color-bg-inline-code,#f6f6f6);
              border:1px solid var(--sl-color-border);
              border-radius:var(--sl-border-radius);
              padding:1.25rem 1.5rem;
              overflow-x:auto;
              max-width:44rem;
              margin:0 auto;
            ">
              <pre style="margin:0;font-size:var(--sl-text-sm);line-height:1.9;"><code>${INSTALL_STEPS.map(
                (step) => html`<span style="color:var(--sl-color-gray-4);"># ${step.note}</span>
<span style="color:var(--sl-color-text);">${step.cmd}</span>
`,
              )}</code></pre>
            </div>
          </section>

          <!-- What you can point it at. Every row links to the page that
               covers it, so this doubles as the table of contents. -->
          <section style="margin-bottom:3.5rem;">
            <h2 style="
              font-size:var(--sl-text-xl);
              font-weight:700;
              color:var(--sl-color-text);
              margin:0 0 1.5rem;
              text-align:center;
            ">Six things to type</h2>
            <div style="display:grid;gap:0.75rem;max-width:44rem;margin:0 auto;">
              ${MODES.map(
                (m) => html`
                  <a href="${m.href}" style="
                    display:flex;
                    align-items:baseline;
                    gap:1rem;
                    flex-wrap:wrap;
                    padding:0.75rem 1rem;
                    border:1px solid var(--sl-color-border);
                    border-radius:var(--sl-border-radius);
                    text-decoration:none;
                  ">
                    <code style="
                      flex-shrink:0;
                      font-family:var(--sl-font-mono,ui-monospace,monospace);
                      font-size:var(--sl-text-sm);
                      background:var(--sl-color-bg-inline-code,#f6f6f6);
                      border:1px solid var(--sl-color-border);
                      border-radius:0.25rem;
                      padding:0.15rem 0.5rem;
                      white-space:nowrap;
                      color:var(--sl-color-text-accent,var(--sl-color-accent));
                    ">${m.cmd}</code>
                    <span style="color:var(--sl-color-text);font-size:var(--sl-text-base);">
                      ${m.what}
                    </span>
                  </a>
                `,
              )}
            </div>
          </section>

          <section>
            <litro-card-grid>
              ${features.map(f => html`
                <litro-card
                  icon="${f.icon ?? ''}"
                  title="${f.title}"
                  description="${f.description}"
                ></litro-card>
              `)}
            </litro-card-grid>
          </section>
        </main>
        <litro-footer recipe="starlight"></litro-footer>
      </div>
    `;
  }
}

export default SplashPage;
