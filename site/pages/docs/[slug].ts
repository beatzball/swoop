import { html } from 'lit';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';
import { customElement } from 'lit/decorators.js';
import { LitroPage, pageReset } from '@beatzball/litro/runtime';
import { definePageData } from '@beatzball/litro';
import { createError } from 'h3';
import type { Post } from 'litro:content';
import { getPosts } from 'litro:content';
import { siteConfig } from '../../server/starlight.config.js';
import { extractHeadings, addHeadingIds } from '../../src/extract-headings.js';
import { starlightHead } from '../../src/route-meta.js';
import { buildSeoHead, buildSeoTitle } from '../../src/seo.js';
import { contentStyles } from '../../src/content-styles.js';
import { annotateImages } from '../../src/annotate-images.js';

// Register components used in render()
import '../../src/components/starlight-page.js';

export interface DocPageData {
  /** Only what render() reads. See the note where this is built. */
  doc: { title: string };
  body: string;
  toc: Array<{ depth: number; text: string; slug: string }>;
  sidebar: typeof siteConfig.sidebar;
  siteTitle: string;
  currentSlug: string;
  prevDoc: { label: string; href: string } | null;
  nextDoc: { label: string; href: string } | null;
  nav: typeof siteConfig.nav;
  editUrl: string | null;
  /** Raw <head> HTML for this doc. See src/seo.ts. */
  seoHead: string;
  /**
   * Overrides routeMeta.title. routeMeta is static per FILE, and this file
   * serves every doc, so without this every page would share one title.
   */
  seoTitle: string;
}

function computePrevNext(
  sidebar: typeof siteConfig.sidebar,
  currentSlug: string,
): { prevDoc: DocPageData['prevDoc']; nextDoc: DocPageData['nextDoc'] } {
  const flat = sidebar.flatMap(g => g.items);
  const idx = flat.findIndex(item => item.slug === currentSlug);
  return {
    prevDoc: idx > 0
      ? { label: flat[idx - 1].label, href: `/docs/${flat[idx - 1].slug}` }
      : null,
    nextDoc: idx < flat.length - 1
      ? { label: flat[idx + 1].label, href: `/docs/${flat[idx + 1].slug}` }
      : null,
  };
}

export const pageData = definePageData(async (event) => {
  const slug = event.context.params?.slug ?? '';

  // Content URLs are /content/docs/<slug> (contentDir = 'content', so
  // dirname is project root, and paths include the 'content/' prefix in the URL).
  const posts = await getPosts();
  const doc = posts.find(p => p.url === `/content/docs/${slug}`);

  if (!doc) {
    throw createError({ statusCode: 404, message: `Doc not found: ${slug}` });
  }

  const toc = extractHeadings(doc.rawBody);

  // Imported here rather than at the top of the file, and this is load-bearing.
  //
  // `highlight.js` pulls in every one of its ~190 language grammars. A static
  // top-level import put all of it in the CLIENT bundle — this file is the
  // page component as well as its data fetcher — making the chunk for every
  // doc route 940KB, of which 920KB was grammars the browser never runs.
  //
  // It is only needed while this fetcher runs, which under SSG is at build
  // time: the highlighted markup is baked into the static HTML. As a dynamic
  // import it becomes its own chunk that the client build emits but never
  // requests, and the route chunk drops to ~19KB.
  //
  // Deliberately not guarded with `import.meta.env.SSR` — Nitro does not
  // define it, so the guard reads as undefined on the server and silently
  // skips highlighting altogether. Every code block renders unstyled and
  // nothing errors.
  const { applyHighlighting } = await import('../../src/highlight.js');
  // src/image-size.ts reads node:fs. Imported at the top of this file those
  // builtins land in the CLIENT bundle, where Vite externalizes them with a
  // warning rather than an error — the build passes and the doc page renders
  // blank in the browser. Same dynamic-import treatment as highlighting.
  const { imageSize } = await import('../../src/image-size.js');
  const body = annotateImages(
    applyHighlighting(addHeadingIds(doc.body)),
    `content/docs/${slug}.md`,
    imageSize,
  );
  const { prevDoc, nextDoc } = computePrevNext(siteConfig.sidebar, slug);
  const title = doc.title || slug;
  const description = doc.description || siteConfig.description;
  const editUrl = siteConfig.editUrlBase
    ? `${siteConfig.editUrlBase}/${slug}.md`
    : null;

  return {
    // Only the title, not the whole doc. Returning `doc` sent doc.body (the
    // un-highlighted HTML) and doc.rawBody (the Markdown source) into the
    // page's JSON payload as well, so every doc page shipped its own content
    // three times over — 11.5KB of a 20.4KB payload on getting-started.
    doc: { title: doc.title || slug },
    body,
    toc,
    sidebar: siteConfig.sidebar,
    siteTitle: siteConfig.title,
    currentSlug: slug,
    prevDoc,
    nextDoc,
    nav: siteConfig.nav,
    editUrl,
    seoTitle: buildSeoTitle(title),
    seoHead: buildSeoHead({
      title,
      description,
      path: `/docs/${slug}`,
      // A doc page is a document, not a landing page; og:type drives how
      // some readers (and Google) treat it.
      type: 'article',
    }),
  } satisfies DocPageData;
});

export async function generateRoutes(): Promise<string[]> {
  const posts = await getPosts();
  return posts
    .filter(p => p.url.startsWith('/content/docs/'))
    .map(p => '/docs' + p.url.slice('/content/docs'.length));
}

export const routeMeta = {
  head: starlightHead,
  title: 'Docs — swoop',
};

@customElement('page-docs-slug')
export class DocPage extends LitroPage {
  /**
   * Styles injected into page-docs-slug's shadow root so they reach the
   * <div slot="content"> subtree. Defined in src/content-styles.ts, which
   * says why, and shared with the blog post page.
   */
  static override styles = [pageReset, contentStyles];

  override render() {
    const data = this.serverData as DocPageData | null;
    if (!data?.doc) return html`<p>Loading&hellip;</p>`;

    // TEMPLATE NOTE — the note below was an HTML comment in the template. An
    // HTML comment is served to every reader, so the prose lives here, named by
    // the element it belongs to.
    //
    // ${unsafeHTML(data.body)}
    //     unsafeHTML renders the Markdown-generated HTML directly.
    //     The content/docs directory is trusted-author-only; do not place
    //     user-submitted or untrusted content here without sanitizing.

    return html`
      <starlight-page
        siteTitle="${data.siteTitle}"
        pageTitle="${data.doc.title}"
        .nav="${data.nav}"
        .sidebar="${data.sidebar}"
        .toc="${data.toc}"
        currentSlug="${data.currentSlug}"
        currentPath="/docs/${data.currentSlug}"
      >
        <div slot="content">
          ${unsafeHTML(data.body)}

          ${data.prevDoc || data.nextDoc ? html`
            <nav style="
              display:flex;
              justify-content:space-between;
              padding-top:2rem;
              margin-top:2rem;
              border-top:1px solid var(--sl-color-border);
              font-size:var(--sl-text-sm);
            " aria-label="Previous and next pages">
              ${data.prevDoc ? html`
                <a href="${data.prevDoc.href}" style="color:var(--sl-color-accent);text-decoration:none;">
                  ← ${data.prevDoc.label}
                </a>
              ` : html`<span></span>`}
              ${data.nextDoc ? html`
                <a href="${data.nextDoc.href}" style="color:var(--sl-color-accent);text-decoration:none;">
                  ${data.nextDoc.label} →
                </a>
              ` : ''}
            </nav>
          ` : ''}

          ${data.editUrl ? html`
            <p style="margin-top:1.5rem;font-size:var(--sl-text-xs);color:var(--sl-color-gray-4);">
              <a href="${data.editUrl}" style="color:var(--sl-color-accent);" target="_blank" rel="noopener">
                Edit this page
              </a>
            </p>
          ` : ''}
        </div>
      </starlight-page>
    `;
  }
}

export default DocPage;
