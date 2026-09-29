import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";
import { pageReset } from '@beatzball/litro/runtime/page-reset.js';

export interface NavItem {
  label: string;
  href: string;
}

/**
 * A real browser, not a server DOM shim.
 *
 * `typeof document !== "undefined"` is NOT enough. A server-side DOM shim can
 * define `document` and still leave `documentElement` undefined, which is how
 * the FAST copy of this header threw
 * "Cannot read properties of undefined (reading 'getAttribute')" during SSR
 * and took down every page that carried it. Lit's SSR never calls
 * firstUpdated(), so this copy never reached the bug — but the adapter copies
 * are read against each other, and the shape has to be the same in all of
 * them. See `.agents/rules/adapters-ssr.md` in the litro repository.
 *
 * Guard on the thing you are about to touch, not on a global a shim provides.
 */
function themeRoot(): HTMLElement | undefined {
  if (typeof document === "undefined") return undefined;
  return document.documentElement ?? undefined;
}

/**
 * <starlight-header siteTitle="My Docs" .nav=${nav} currentPath="/docs/getting-started">
 *   Top navigation bar with site title, nav links, and dark/light theme toggle.
 */
@customElement("starlight-header")
export class StarlightHeader extends LitElement {
  static override properties = {
    siteTitle: { type: String },
    nav: { type: Array },
    currentPath: { type: String },
    navOpen: { type: Boolean },
    hasSidebar: { type: Boolean },
    _theme: { type: String, state: true },
  };

  static override styles = [
    pageReset,
    css`
    :host {
      display: block;
      position: sticky;
      top: 0;
      z-index: 100;
    }

    header {
      height: var(--sl-nav-height, 3.5rem);
      background-color: var(--sl-color-bg-nav, #fff);
      border-bottom: 1px solid var(--sl-color-border, #e8e8e8);
      display: flex;
      align-items: center;
      padding: 0 var(--sl-content-pad-x, 1.5rem);
      gap: 1rem;
    }

    /* At 320px the row runs 13px past the viewport once the menu button is
       there too: 48 padding + 36 button + 86 title + 58 nav + 82 actions + 48
       gaps. Nothing here can wrap, so the whole page scrolls sideways. Buy the
       room back from the padding and the gaps, which are the two parts nobody
       misses at that width. */
    @media (max-width: 22.5rem) {
      header {
        padding: 0 0.75rem;
        gap: 0.5rem;
      }
    }

    .menu-btn {
      display: none;
      appearance: none;
      background: none;
      border: 1px solid var(--sl-color-border, #e8e8e8);
      border-radius: var(--sl-border-radius, 0.375rem);
      width: 2.25rem;
      height: 2.25rem;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      color: var(--sl-color-text, #23262f);
      transition: background-color 0.15s;
      flex-shrink: 0;
      padding: 0;
    }

    .menu-btn:hover {
      background-color: var(--sl-color-gray-2, #e8e8e8);
    }

    .menu-btn svg {
      width: 1.1rem;
      height: 1.1rem;
    }

    @media (max-width: 72rem) {
      .menu-btn {
        display: flex;
      }
    }

    /* THE BRAND FACE. --sl-font-brand is what the wordmark and the navigation
       are set in, and it falls back to the body sans, so a site that never
       sets it looks exactly as it did. A site that wants the terminal
       character in its header sets it once, to the mono, and both the name
       and the links follow.

       It is a token rather than a fork of this component because that is the
       whole of the difference: two declarations, not a second header. */
    .site-title {
      font-family: var(--sl-font-brand, var(--sl-font-sans));
      font-size: var(--sl-text-lg, 1.125rem);
      font-weight: 700;
      color: var(--sl-color-text, #23262f);
      text-decoration: none;
      white-space: nowrap;
      /* A flex item will not shrink below its own text, so a long site name
         pushes the theme toggle past the right edge of a 320px screen. These
         three let the row give way and end the name in an ellipsis instead. */
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .site-logo {
      /* Fixed box so the header never reflows while the image loads. */
      width: 1.75rem;
      height: 1.75rem;
      object-fit: contain;
      flex-shrink: 0;
    }

    /* Actions sit together on the right; the toggle used to claim the space
       on its own with margin-left:auto, which left no room for GitHub. */
    .header-actions {
      margin-left: auto;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex-shrink: 0;
    }

    .github-link {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 2.25rem;
      height: 2.25rem;
      border: 1px solid var(--sl-color-border, #e8e8e8);
      border-radius: var(--sl-border-radius, 0.375rem);
      color: var(--sl-color-text, #23262f);
      text-decoration: none;
      transition: background-color 0.15s;
    }

    .github-link:hover {
      background: var(--sl-color-bg-nav, #f6f6f6);
    }

    .github-link svg {
      width: 1.125rem;
      height: 1.125rem;
      fill: currentColor;
    }

    .site-title:hover {
      opacity: 0.85;
    }

    nav {
      display: flex;
      align-items: center;
      gap: 0.25rem;
      flex: 1;
      /* min-width: 0, or a flex item refuses to shrink below its content and
         the whole header grows past a phone's screen. On the docs pages the
         nav is hidden behind the hamburger below 72rem and this never showed;
         the landing page has no sidebar, so it keeps its links and needs the
         row to be able to give way. */
      min-width: 0;
    }

    /* A phone cannot fit four links, a search control and two icon buttons.
       The links SCROLL rather than disappear: dropping one would take a
       destination away from exactly the reader with the least room to go
       looking for it. The bar is hidden because a scrollbar inside a header
       is noise, and the links are still reachable by keyboard and by swipe. */
    @media (max-width: 48rem) {
      nav {
        overflow-x: auto;
        scrollbar-width: none;
        -ms-overflow-style: none;
      }

      nav::-webkit-scrollbar {
        display: none;
      }

      nav a {
        flex-shrink: 0;
      }
    }

    nav a {
      font-family: var(--sl-font-brand, var(--sl-font-sans));
      padding: 0.35rem 0.75rem;
      font-size: var(--sl-text-sm, 0.875rem);
      font-weight: 500;
      color: var(--sl-color-gray-5, #4b4b4b);
      text-decoration: none;
      border-radius: var(--sl-border-radius, 0.375rem);
      transition:
        color 0.15s,
        background-color 0.15s;
    }

    nav a:hover {
      color: var(--sl-color-text, #23262f);
      background-color: var(--sl-color-gray-2, #e8e8e8);
    }

    nav a[aria-current="page"] {
      color: var(--sl-color-accent, #7c3aed);
      background-color: var(--sl-color-accent-low, #ede9fe);
    }

    .theme-toggle {
      appearance: none;
      background: none;
      border: 1px solid var(--sl-color-border, #e8e8e8);
      border-radius: var(--sl-border-radius, 0.375rem);
      width: 2.25rem;
      height: 2.25rem;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      font-size: 1rem;
      color: var(--sl-color-text, #23262f);
      transition: background-color 0.15s;
      flex-shrink: 0;
    }

    .theme-toggle:hover {
      background-color: var(--sl-color-gray-2, #e8e8e8);
    }

    .theme-toggle svg {
      width: 1.125rem;
      height: 1.125rem;
    }
  `,
  ];

  siteTitle = "";
  nav: NavItem[] = [];
  currentPath = "";
  navOpen = false;
  hasSidebar = false;

  // What the server renders the icon as. Dark, because the head script in
  // src/route-meta.ts makes dark the default, so the icon does not flip on
  // load for a reader who has chosen nothing.
  _theme = "dark";

  /**
   * Keep the toggle's icon on the theme the page is actually showing.
   *
   * THIS READS, IT DOES NOT DECIDE. The head script in route-meta.ts sets
   * data-theme before the first paint, from the reader's stored choice or,
   * when they have made none, from the system. Resolving it a second time
   * here would answer a moment later, and a different answer would overwrite
   * a correct value with a wrong one — which is what this component used to
   * do: it fell back to "light" with no look at prefers-color-scheme, so on
   * a dark system every page carrying this header flipped to light right
   * after it loaded.
   */
  private _readTheme = () => {
    if (!themeRoot()) return;
    this._theme =
      document.documentElement.getAttribute("data-theme") === "dark"
        ? "dark"
        : "light";
  };

  private _systemTheme?: MediaQueryList;

  override firstUpdated() {
    // One guard for the whole block: with no documentElement there is no
    // theme to read and no system preference worth listening to.
    if (!themeRoot()) return;
    this._readTheme();
    // The head script follows the system while the reader has stored no
    // choice, so the icon has to follow it too. The head script's own
    // listener was registered first, in <head>, so by the time this one runs
    // data-theme is already up to date.
    if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
      this._systemTheme = window.matchMedia("(prefers-color-scheme: dark)");
      this._systemTheme.addEventListener("change", this._readTheme);
    }
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this._systemTheme?.removeEventListener("change", this._readTheme);
  }

  private _toggleTheme() {
    const next = this._theme === "light" ? "dark" : "light";
    this._theme = next;
    // Writing the choice is what stops the head script's system listener
    // from overriding it later.
    if (typeof localStorage !== "undefined") {
      try {
        localStorage.setItem("sl-theme", next);
      } catch {
        // Site data blocked. The choice holds for this page either way.
      }
    }
    if (themeRoot()) {
      document.documentElement.setAttribute("data-theme", next);
    }
  }

  private _toggleNav() {
    this.dispatchEvent(
      new CustomEvent("sl-nav-toggle", { bubbles: true, composed: true }),
    );
  }

  override render() {
    // GitHub gets its own icon button rather than sitting in the text nav,
    // matching litro.dev. Matched on href so a relabeled entry still works.
    const isGithub = (item: { label: string; href: string }) =>
      /github\.com/i.test(item.href);
    const githubItem = this.nav.find(isGithub);
    const regularNav = this.nav.filter((item) => !isGithub(item));

    // Inline SVG rather than an emoji: emoji render at wildly different sizes
    // and weights per platform, so the button jumped around next to the
    // GitHub mark.
    const icon =
      this._theme === "dark"
        ? html`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
            </svg>`
        : html`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              aria-hidden="true">
              <path d="M21 12.8A9 9 0 1111.2 3a7 7 0 009.8 9.8z" />
            </svg>`;
    const label =
      this._theme === "dark" ? "Switch to light mode" : "Switch to dark mode";

    return html`
      <header>
        ${this.hasSidebar
          ? html`
              <button
                class="menu-btn"
                aria-label="${this.navOpen
                  ? "Close navigation"
                  : "Open navigation"}"
                aria-expanded="${this.navOpen}"
                @click="${this._toggleNav}"
              >
                ${this.navOpen
                  ? html`
                      <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        aria-hidden="true"
                      >
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    `
                  : html`
                      <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        aria-hidden="true"
                      >
                        <line x1="3" y1="6" x2="21" y2="6" />
                        <line x1="3" y1="12" x2="21" y2="12" />
                        <line x1="3" y1="18" x2="21" y2="18" />
                      </svg>
                    `}
              </button>
            `
          : ""}
        <a class="site-title" href="/">
          <img class="site-logo" src="/logo.webp" alt="" aria-hidden="true" />
          ${this.siteTitle}
        </a>
        <nav aria-label="Main navigation">
          ${regularNav.map(
            (item) => html`
              <a
                href="${item.href}"
                aria-current="${this.currentPath.startsWith(item.href)
                  ? "page"
                  : "false"}"
                >${item.label}</a
              >
            `,
          )}
        </nav>
        <div class="header-actions">
          ${githubItem
            ? html`
                <a
                  class="github-link"
                  href="${githubItem.href}"
                  target="_blank"
                  rel="noopener"
                  aria-label="GitHub (opens in new tab)"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"
                    />
                  </svg>
                </a>
              `
            : ""}
          <button
            class="theme-toggle"
            aria-label="${label}"
            @click="${this._toggleTheme}"
          >
            ${icon}
          </button>
        </div>
      </header>
    `;
  }
}

export default StarlightHeader;
