import { html, css } from 'lit';
import { customElement } from 'lit/decorators.js';
import { LitroPage, pageReset } from '@beatzball/litro/runtime';
import { definePageData } from '@beatzball/litro';
import { getGlobalData } from 'litro:content';
import { siteConfig } from '../server/starlight.config.js';
import { starlightHead } from '../src/route-meta.js';
import { buildSeoHead, buildSeoTitle } from '../src/seo.js';

// Register the components used in render(). The landing page's own first...
import '../src/components/litro-hero-nova.js';
import '../src/components/litro-status-line.js';
import '../src/components/litro-pane.js';
import '../src/components/litro-pane-grid.js';
import '../src/components/litro-site-footer.js';
import '../src/components/litro-install-command.js';
import '../src/components/litro-feature-row.js';
import '../src/components/litro-steps.js';
import '../src/components/litro-key-hints.js';
import '../src/components/litro-term-window.js';

// ...then the one shared with the docs half of this site.
import '../src/components/starlight-header.js';

import type { StepItem } from '../src/components/litro-steps.js';
import type { KeyHint } from '../src/components/litro-key-hints.js';
import type { StatusCell } from '../src/components/litro-status-line.js';
import type { FooterColumn } from '../src/components/litro-site-footer.js';

/**
 * The landing page, on litro's supernova recipe.
 *
 * The page renders fully on the server, so all of this copy is readable with
 * JavaScript turned off. The copy lives in the lists at the top of the file,
 * so a change of wording is an edit here and not a hunt through the markup.
 *
 * THE README AND THIS PAGE SAY THE SAME THINGS. Install, the hotkey and the
 * keys are kept in step with the README and with
 * content/docs/getting-started.md; change one and change the others.
 */

/**
 * The command a reader copies. Homebrew, because it is one line on a Mac and
 * on Linux; the curl line is in the note under it and in Getting Started.
 */
const INSTALL_COMMAND = 'brew install beatzball/tap/swoop';

/**
 * The cells in the status line at the foot of the window.
 *
 * EVERY CELL MUST BE A FACT. A status line is read as live state, so nothing
 * in it may be invented. The version is deliberately NOT here: the Docker
 * build's context is site/, so ../VERSION is not there to read, and a
 * hand-written number would be wrong the day after the next release.
 */
const STATUS_CELLS: StatusCell[] = [
  { state: 'working', value: 'docs', trailing: '/getting-started', href: '/docs/getting-started' },
  { label: 'license', value: 'MIT', href: 'https://github.com/beatzball/swoop/blob/main/LICENSE', optional: true },
  { label: 'built with', value: 'litro', href: 'https://litro.dev', right: true },
];

/** What the line is, for a reader who cannot see it. */
const STATUS_LABEL = 'Project status';

/**
 * The "what it does" rows. Every command chip is something a reader can type
 * into the launcher, or run in a shell, and see work.
 *
 * The one picture is the output of the clock extension from
 * content/docs/writing-an-extension.md, step 1. That script is tested; this is
 * what it prints, at 09:41 in London. `cut -f4,5` keeps the title and the
 * subtitle, the two fields a reader sees, because the whole line does not fit
 * the picture's width; the picture lines them up with spaces rather than the
 * real tab so they read as columns.
 */
const HIGHLIGHTS: Array<{
  title: string;
  description: string;
  commands: string[];
  href: string;
  figure?: { label: string; shell: string };
}> = [
  {
    title: 'Everything in one list',
    description:
      'Apps, files, quicklinks, emoji, snippets, notes, tasks and window moves, ' +
      'filtered as you type. Enter does the obvious thing; ctrl-k on any row ' +
      'opens what else it can do.',
    commands: ['saf', '2+2', 'ctrl-k'],
    href: '/docs/extensions',
  },
  {
    title: 'Ask AI, with any model',
    description:
      'Tab asks the AI you name: a local model, a hosted one, or any command ' +
      'that reads a question and prints an answer. The answer stays in the panel.',
    commands: ['Tab'],
    href: '/docs/ask-ai',
  },
  {
    title: 'Extensions are programs',
    description:
      'An extension is a program that prints lines: an id, a kind, an icon, a ' +
      'title and a subtitle, separated by tabs. A shell script is enough. No ' +
      'manifest, no registration, no restart.',
    commands: ['clock list'],
    href: '/docs/writing-an-extension',
    figure: {
      label:
        'A shell session: the clock extension lists three rows, the time in ' +
        'London, New York and Tokyo.',
      shell: `$ clock list | cut -f4,5
Time in London    09:41
Time in New York  04:41
Time in Tokyo     17:41`,
    },
  },
];

/** From nothing to the panel, in order. Kept in step with Getting Started. */
const STEPS: StepItem[] = [
  {
    title: 'Install it',
    description:
      'brew install beatzball/tap/swoop, on a Mac or on Linux. No Homebrew: the curl line in Getting Started needs nothing else.',
  },
  {
    title: 'Start the panel',
    description: 'brew services start swoop, on a Mac. It comes back at every login.',
  },
  {
    title: 'Press alt+shift+space',
    description:
      'Type a few letters, press Enter. On Linux, and in any terminal, run swoop.',
  },
];

/** The keys worth knowing on day one. */
const KEY_HINTS: KeyHint[] = [
  { keys: ['alt', 'shift', 'space'], meaning: 'Show the panel, on a Mac' },
  { keys: 'Enter', meaning: 'Do the obvious thing with the row' },
  { keys: ['ctrl', 'k'], meaning: 'Everything else the row can do' },
  { keys: 'Tab', meaning: 'Ask AI' },
  { keys: 'Esc', meaning: 'Back out of a pane, then close' },
];

/**
 * What swoop stands on. It finds and draws nothing itself: saying so up front
 * is the honest pitch, and it tells a reader what they are installing.
 */
const FOUNDATIONS: Array<{ name: string; meta: string; description: string; href: string }> = [
  {
    name: 'fzf',
    meta: 'does the finding',
    description: 'Every list, every filter and every key. swoop hands it lines and asks it for one back.',
    href: 'https://github.com/junegunn/fzf',
  },
  {
    name: 'libghostty',
    meta: 'does the drawing',
    description: 'The terminal inside the panel on a Mac, so the panel is a real terminal and your tools run in it.',
    href: 'https://github.com/ghostty-org/ghostty',
  },
  {
    name: 'your programs',
    meta: 'print the lines',
    description: 'Every extension, the bundled ones too. Small programs, one job each; swoop is the wiring.',
    href: '/docs/writing-an-extension',
  },
];

/** The map of everything the page did not cover. */
const FOOTER_COLUMNS: FooterColumn[] = [
  {
    heading: 'Docs',
    links: [
      { label: 'Getting Started', href: '/docs/getting-started' },
      { label: 'Ask AI', href: '/docs/ask-ai' },
      { label: 'Extensions', href: '/docs/extensions' },
      { label: 'Settings', href: '/docs/settings' },
    ],
  },
  {
    heading: 'Build',
    links: [
      { label: 'Writing an Extension', href: '/docs/writing-an-extension' },
      { label: 'The extension contract', href: 'https://github.com/beatzball/swoop/issues/2' },
    ],
  },
  {
    heading: 'Project',
    links: [
      { label: 'GitHub', href: 'https://github.com/beatzball/swoop' },
      { label: 'Changelog', href: 'https://github.com/beatzball/swoop/blob/main/CHANGELOG.md' },
      { label: 'Releases', href: 'https://github.com/beatzball/swoop/releases' },
    ],
  },
];

export interface HomeData {
  siteTitle: string;
  description: string;
  nav: Array<{ label: string; href: string }>;
  features: Array<{ title: string; description: string }>;
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
    // Six, drawn two to a row on a wide screen (span 3 of the grid's six
    // columns), so the block closes square.
    features: [
      {
        title: 'One key away',
        description: 'alt+shift+space shows a floating panel on a Mac. Esc, or a click elsewhere, and it is gone.',
      },
      {
        title: 'Terminal first',
        description: 'Notes and tasks open in your editor inside the panel. The same launcher runs in any terminal, on any OS.',
      },
      {
        title: 'Plain files',
        description: 'Settings, snippets, quicklinks, notes and tasks are text files you can edit, grep and sync.',
      },
      {
        title: 'Yours alone',
        description: 'Clipboard history skips password managers. Usage stays on your machine. Nothing is sent anywhere you did not pick.',
      },
      {
        title: 'Any model',
        description: 'A local model, a hosted one, or any command. Presets switch between them, and each says whether it can search the web.',
      },
      {
        title: 'Open source',
        description: 'MIT licensed, on GitHub. Every tool is its own small program with its own tests.',
      },
    ],
  } satisfies HomeData;
});

export const routeMeta = {
  head: starlightHead,
  title: 'swoop',
};

/**
 * The bird from public/logo.svg, drawn inline as the hero's mark.
 *
 * Inline rather than an <img>, so the hero costs no extra request, and filled
 * from the accent token rather than the file's own color, so it follows the
 * theme. It is atmosphere: the headline and the header already say the name,
 * so it is hidden from assistive tech. Change the logo, and change this too.
 */
const MARK = html`
  <svg slot="mark" viewBox="0 0 512 512" aria-hidden="true">
    <path
      d="M48 96 C 90 380, 250 440, 350 318"
      fill="none"
      stroke="var(--nova-accent)"
      stroke-opacity="0.45"
      stroke-width="34"
      stroke-linecap="round"
    />
    <path
      transform="translate(372 256) rotate(-32) scale(1.15)"
      d="M-110 -40 C -50 -72 -10 -44 0 8 C 10 -44 50 -72 110 -40 C 60 -40 26 -10 12 42 L -12 42 C -26 -10 -60 -40 -110 -40 Z"
      fill="var(--nova-accent)"
    />
  </svg>
`;

@customElement('page-home')
export class HomePage extends LitroPage {
  static override styles = [
    pageReset,
    css`
    /* ── The token block ───────────────────────────────────────────────
     *
     * EDIT THIS BLOCK TO RETHEME THE WHOLE LANDING PAGE. The page and all of
     * its components read these tokens and define no colors of their own, so
     * nothing else has to change.
     *
     * Every value is var(--brand-…, fallback). Set a --brand-… property
     * anywhere above this element — :root in public/styles/starlight.css
     * is the usual place — and the page follows it. Leave it unset and the
     * fallback here applies.
     *
     * THE DOCS HALF OF THIS SITE HAS ITS OWN TOKENS, and they are not these.
     * They are named --sl-* and they are defined in
     * public/styles/starlight.css: --sl-color-bg, --sl-color-text,
     * --sl-color-accent, --sl-color-border, --sl-color-gray-1 to
     * --sl-color-gray-6, --sl-color-note, --sl-color-tip,
     * --sl-color-caution, --sl-color-danger, --sl-font-sans,
     * --sl-font-mono, --sl-text-xs to --sl-text-4xl, --sl-nav-height,
     * --sl-sidebar-width, --sl-toc-width, --sl-content-width,
     * --sl-shadow-sm, --sl-shadow-md, --sl-border-radius and
     * --sl-border-radius-sm. Do not redefine any of those here — the docs
     * pages read them too, and the header on this page is a docs component.
     *
     * THE PAGE FOLLOWS THE READER'S LIGHT OR DARK CHOICE, exactly as the docs
     * half does, because the --brand-* values it reads are defined in the
     * light and dark blocks of public/styles/starlight.css. Change them there
     * and both themes move together; there is no color-scheme declaration
     * here and no dark-only palette, which is what used to leave this page
     * dark when a reader switched to light.
     */
    :host {
      /* surfaces — the dark ground is TINTED, not flat black; the light one
         is a warm off-white rather than pure white, because the hero is a
         large field of one color and pure white under a wash reads as a
         blown-out photograph. */
      --nova-bg: var(--brand-bg, #0d0e1a);
      --nova-surface: var(--brand-surface, #171a2b);
      --nova-border: var(--brand-border, #2a2e45);

      /* text */
      --nova-text: var(--brand-text, #e9ecfa);
      --nova-text-dim: var(--brand-text-dim, #979db8);

      /* accent — TWO of them, and the reason is contrast. --nova-accent is
         the one that reads as text against the page. It is not dark enough
         to carry WHITE text on it, so anything that puts words on an accent
         FIELD — the status line's mode segment, the primary button — takes
         --nova-accent-high, which the stylesheet derives from the same
         accent and which clears 4.5:1 in both themes. */
      --nova-accent: var(--brand-accent, #7c3aed);
      --nova-accent-high: var(--brand-accent-high, #4c1d95);
      /* And a third: the accent as small TEXT on the page. On a light ground
         a brand color usually reads under 4.5:1, so the stylesheet darkens it
         there and leaves it alone on a dark one. */
      --nova-accent-text: var(--brand-accent-text, #7c3aed);
      --nova-accent-2: var(--brand-accent-2, #22d3ee);

      /* states */
      --nova-error: var(--brand-error, #f87171);
      --nova-blocked: var(--brand-blocked, #fbbf24);
      --nova-working: var(--brand-working, #38bdf8);
      --nova-done: var(--brand-done, #4ade80);
      --nova-idle: var(--brand-idle, #64748b);

      /* layout */
      --nova-measure: var(--brand-measure, 64rem);
      --nova-gutter: var(--brand-gutter, 1.5rem);
      --nova-radius: var(--brand-radius, 0.375rem);
      /* How tall the hero pane is before its content makes it taller. The
         3rem is the status bar above it, so the hero fills exactly what is
         left of the first screen. */
      /* The header above and the status line below both come out of the
         first screen, so the hero fills exactly what is left of it. */
      --nova-hero-min: var(--brand-hero-min, calc(100svh - 3.5rem - 1.75rem));
      /* How solid the hero's mark is. A light ground shows far less of a
         faint shape than a dark one, so the value is per theme. */
      --nova-mark-opacity: var(--brand-mark-opacity, 0.11);
      --nova-font-mono: var(
        --brand-font-mono,
        ui-monospace,
        'Cascadia Code',
        'Fira Code',
        monospace
      );
    }

    /* ── Page frame ────────────────────────────────────────────────────── */

    :host {
      display: block;
      background: var(--nova-bg);
      color: var(--nova-text);
    }

    /* The status line at the foot is FIXED, so the page has to leave room for
       it by hand or the credit line sits under it — and the screen it fills
       is that much shorter. */
    .page {
      --status-height: var(--nova-status-height, 1.75rem);
      min-height: calc(100vh - var(--status-height));
      min-height: calc(100svh - var(--status-height));
      padding-bottom: var(--status-height);
      display: flex;
      flex-direction: column;
    }

    main {
      flex: 1;
      width: 100%;
    }

    .shell {
      max-width: var(--nova-measure);
      margin: 0 auto;
      padding: 0 var(--nova-gutter);
      width: 100%;
    }

    /* ── The header ────────────────────────────────────────────────────
     *
     * starlight-header is the DOCS half of this site's header, and it reads
     * the --sl-* tokens, which follow the reader's light or dark choice while
     * this page is dark either way. So the set it needs is given to it here,
     * on the element, exactly as litro-card is dressed above.
     *
     * --sl-font-brand is the one token that is not a dressing. It is the
     * header's brand face, and setting it to the mono is what carries the
     * terminal character into a header that is otherwise the docs header,
     * unchanged. Delete this one line and the header matches the docs pages'
     * exactly.
     */
    starlight-header {
      --sl-font-brand: var(--nova-font-mono);
      --sl-color-bg: var(--nova-bg);
      --sl-color-bg-nav: var(--nova-bg);
      --sl-color-text: var(--nova-text);
      --sl-color-gray-2: var(--nova-surface);
      --sl-color-gray-4: var(--nova-text-dim);
      --sl-color-gray-5: var(--nova-text-dim);
      --sl-color-border: var(--nova-border);
      --sl-color-accent: var(--nova-accent);
      --sl-color-accent-low: color-mix(
        in srgb,
        var(--nova-accent) 18%,
        transparent
      );
    }

    /* ── Hero ──────────────────────────────────────────────────────────
     *
     * LEFT, ON A COLUMN. Everything in the hero starts on one line down the
     * left, so the eye has a spine to run down: headline, lede, slab, small
     * print, buttons. A centered stack has no spine — every line starts
     * somewhere different and nothing leads anywhere.
     *
     * The column stops well short of the measure, which is what leaves the
     * right of the pane to the mark.
     */

    /* padding-top and padding-bottom, never the two-value padding shorthand:
       every section below also carries .shell, whose HORIZONTAL padding is
       the page's gutter. A rule like "padding: 4rem 0" is later in this sheet
       at the same specificity, so it would quietly set that gutter to zero
       and let the section run to the edge of a phone. */

    .hero {
      padding-top: 4rem;
      padding-bottom: 4rem;
    }

    .hero-copy {
      max-width: 46rem;
    }

    /* THE HEADLINE IS SET IN THE MONO FACE — the same one the status bar, the
       badges, the terminal pictures and the command below are set in. That is
       the whole type idea of this page: it speaks in one voice, and the voice
       is the terminal's. The sans is kept for prose, where it is easier to
       read, and for nothing else. */
    .hero h1 {
      font-family: var(--nova-font-mono);
      /* The floor is what a phone gets, and a mono face is wide: at 2rem the
         word "product" alone is most of a 390px column and the headline runs
         off the side. 1.75rem is the largest floor that still lets this
         sentence wrap. */
      font-size: clamp(1.75rem, 5.6vw, 4.25rem);
      font-weight: 700;
      line-height: 1.04;
      letter-spacing: -0.02em;
      margin: 0 0 1.5rem;
      text-wrap: balance;
    }

    /* The lede keeps its own, narrower measure. The slab below it does not:
       a command has to be read in one piece, so it takes the whole column. */
    .lede {
      font-size: clamp(1.1rem, 1.5vw, 1.3rem);
      color: var(--nova-text-dim);
      max-width: 34rem;
      margin: 0 0 2.5rem;
      line-height: 1.75;
    }

    .hero litro-install-command {
      display: block;
      margin: 0 0 2.5rem;
    }

    /* ── Buttons ───────────────────────────────────────────────────────── */

    .actions {
      display: flex;
      gap: 1rem;
      flex-wrap: wrap;
    }

    .button {
      display: inline-block;
      padding: 0.6rem 1.5rem;
      border-radius: var(--nova-radius);
      font-weight: 600;
      text-decoration: none;
      border: 1px solid transparent;
    }

    .button.primary {
      background: var(--nova-accent);
      color: var(--nova-text);
    }

    .button.ghost {
      border-color: var(--nova-border);
      color: var(--nova-text);
    }

    .button:focus-visible {
      outline: 2px solid var(--nova-accent);
      outline-offset: 2px;
    }

    /* ── Feature rows ──────────────────────────────────────────────────── */

    .rows {
      display: flex;
      flex-direction: column;
      gap: 3.5rem;
      padding-top: 4rem;
      padding-bottom: 4rem;
    }

    /* Shadow DOM styles stop at a slot, so the paragraph handed to a row is
       styled here, by the page, and not inside litro-feature-row. */
    .rows p {
      color: var(--nova-text-dim);
      line-height: 1.7;
      margin: 0;
    }

    /* ── Get running ───────────────────────────────────────────────────── */

    .start {
      display: grid;
      /* minmax(0, 1fr), not 1fr. A bare 1fr is minmax(auto, 1fr), so the
         track never shrinks below its widest item's content — a long
         transcript line then pushes the column past a phone's screen and the
         page scrolls sideways at the 320px WCAG 1.4.10 reflow width. The 0
         lets the track shrink and the item scroll inside itself instead. */
      grid-template-columns: minmax(0, 1fr);
      gap: 2.5rem;      padding-top: 1rem;
      padding-bottom: 4rem;
    }

    @media (min-width: 48rem) {
      .start {
        grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        gap: 4rem;
      }
    }

    /* ── SECTIONS 3, 4, 6 · The pane blocks ───────────────────────────── */

    .panes {
      padding-top: 3rem;
      padding-bottom: 1rem;
    }

    /* Shadow DOM styles stop at a slot, so a pane's body text is styled by
       the pane and the page needs nothing here. */

    /* ── SECTION 9 · The footer ────────────────────────────────────────── */

    litro-site-footer {
      margin-top: 4rem;
    }

    /* ── Section headings and closing ──────────────────────────────────── */

    .section-title {
      font-size: 1.5rem;
      font-weight: 700;
      margin: 0 0 1.25rem;
    }

    .closing {
      text-align: center;
      padding-top: 4rem;
      padding-bottom: 5rem;
      border-top: 1px solid var(--nova-border);
    }

    .closing h2 {
      font-size: clamp(1.6rem, 4vw, 2.5rem);
      font-weight: 800;
      margin: 0 0 1.5rem;
    }

    .closing litro-install-command {
      display: block;
      max-width: 34rem;
      margin: 0 auto 2rem;
    }

    .closing .actions {
      justify-content: center;
    }
    /* ── swoop's own additions ────────────────────────────────────────── */

    /* The name is the joke, so it lands after the reader has already decided
       whether to click. The README opens with it. */
    .tagline {
      margin: 1.75rem 0 0;
      color: var(--nova-text-dim);
      font-style: italic;
      line-height: 1.6;
    }

    /* The recording carries the whole pitch faster than any paragraph. */
    .demo {
      padding-top: 4rem;
    }

    .demo img {
      display: block;
      width: 100%;
      /* Required, because of the width/height attributes on the tag. Those
         reserve the right box before the image loads, but width:100% overrides
         only the width half of the pair. Without this the height attribute
         stands and the recording is squashed to 560px tall at every width. */
      height: auto;
      max-width: 48rem;
      margin: 0 auto;
      border: 1px solid var(--nova-border);
      border-radius: var(--nova-radius);
    }

    /* The row's heading says what; this says where to read more. Underlined,
       not just colored: a link in running text has to be told apart from the
       words around it by something other than color. */
    .rows a {
      color: var(--nova-accent-text);
      text-decoration: underline;
      text-underline-offset: 0.2em;
    }

    /* The transcript is a fixed-width picture: its columns only line up if
       it never wraps, so it scrolls inside the window on a phone instead. */
    .rows litro-term-window pre {
      margin: 0;
      white-space: pre;
      overflow-x: auto;
      font-family: var(--nova-font-mono);
      font-size: 0.8125rem;
      line-height: 1.7;
    }
  `,
  ];

  override render() {
    const data = this.serverData as HomeData | null;
    const {
      siteTitle = 'swoop',
      description = '',
      nav = [],
      features = [],
    } = data ?? {};

    // TEMPLATE NOTES — why the markup below is shaped the way it is. They are
    // here and not in the template because an HTML comment is served to every
    // reader.
    //
    // <litro-hero-nova>
    //     MARK is the bird from public/logo.svg, slotted as the hero's mark;
    //     a slotted mark replaces the recipe's own drawing by construction.
    //
    // <section class="demo shell">
    //     The recording is demo/swoop.gif as animated WebP. Kept in step with
    //     demo/swoop.tape: re-check the alt text when that tape changes what
    //     it records, and rebuild public/swoop.webp (see site/AGENTS.md).
    //     Lazy, because on most screens the hero fills the first view.
    //
    // <litro-status-line
    //     Fixed to the foot of the window, so it is the last thing in the page
    //     and .page carries the padding that keeps the footer clear of it.
    return html`
      <div class="page">
        <starlight-header
          siteTitle="${siteTitle}"
          .nav="${nav}"
          currentPath="/"
        ></starlight-header>

        <main>
          <litro-hero-nova>
            ${MARK}
            <section class="hero shell">
              <div class="hero-copy">
                <h1>A launcher one key away, built the Unix way.</h1>
                <p class="lede">${description}</p>
                <litro-install-command command="${INSTALL_COMMAND}">
                  <span slot="note"
                    >On a Mac or on Linux. No Homebrew? Getting Started has a
                    curl line that needs nothing else.</span
                  >
                </litro-install-command>
                <div class="actions">
                  <a href="/docs/getting-started" class="button primary">Get Started</a>
                  <a href="https://github.com/beatzball/swoop" class="button ghost">GitHub</a>
                </div>
                <p class="tagline">
                  Say it like a bird does: swoop down, grab the thing, gone.
                </p>
              </div>
            </section>
          </litro-hero-nova>

          <section class="demo shell" aria-label="swoop in use">
            <img
              src="/swoop.webp"
              alt="swoop: filtering apps, the action menu, the calculator, and Ask AI answering a question"
              width="1000"
              height="560"
              loading="lazy"
              decoding="async"
            />
          </section>

          <section class="rows shell" aria-label="What it does">
            ${HIGHLIGHTS.map(
              (row) => html`
                <litro-feature-row
                  class="row"
                  heading="${row.title}"
                  .commands="${row.commands}"
                >
                  <p>${row.description} <a href="${row.href}">Read more</a></p>
                  ${row.figure
                    ? html`
                        <litro-term-window slot="figure" label="${row.figure.label}">
                          <pre>${row.figure.shell}</pre>
                        </litro-term-window>
                      `
                    : ''}
                </litro-feature-row>
              `,
            )}
          </section>

          <section class="start shell" aria-label="Get running">
            <div>
              <h2 class="section-title">Get running</h2>
              <litro-steps .steps="${STEPS}"></litro-steps>
            </div>
            <div>
              <h2 class="section-title">Keys worth knowing</h2>
              <litro-key-hints .hints="${KEY_HINTS}"></litro-key-hints>
            </div>
          </section>

          <section class="panes shell" aria-label="What you get">
            <h2 class="section-title">What ${siteTitle} is</h2>
            <litro-pane-grid>
              ${features.map(
                (f) => html`
                  <litro-pane name="${f.title}" state="done" span="3">
                    ${f.description}
                  </litro-pane>
                `,
              )}
            </litro-pane-grid>
          </section>

          <section class="panes shell" aria-label="What it is built on">
            <h2 class="section-title">Built on</h2>
            <litro-pane-grid>
              ${FOUNDATIONS.map(
                (item) => html`
                  <litro-pane
                    name="${item.name}"
                    meta="${item.meta}"
                    span="2"
                    href="${item.href}"
                    >${item.description}</litro-pane
                  >
                `,
              )}
            </litro-pane-grid>
          </section>

          <section class="closing shell">
            <h2>Swoop down, grab the thing, gone.</h2>
            <litro-install-command command="${INSTALL_COMMAND}"></litro-install-command>
            <div class="actions">
              <a href="/docs/getting-started" class="button primary">Read the docs</a>
            </div>
          </section>
        </main>

        <litro-site-footer
          siteTitle="${siteTitle}"
          .columns="${FOOTER_COLUMNS}"
          credit="Created using Litro"
          creditHref="https://litro.dev"
          note="— the supernova recipe"
        ></litro-site-footer>

        <litro-status-line
          siteTitle="${siteTitle}"
          .cells="${STATUS_CELLS}"
          label="${STATUS_LABEL}"
        ></litro-status-line>
      </div>
    `;
  }
}

export default HomePage;
