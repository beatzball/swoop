import { html } from 'lit';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';
import { customElement } from 'lit/decorators.js';
import { LitroPage, pageReset } from '@beatzball/litro/runtime';
import { definePageData } from '@beatzball/litro';
import { createError } from 'h3';
import { getPosts } from 'litro:content';
import { siteConfig } from '../../server/starlight.config.js';
import { addHeadingIds } from '../../src/extract-headings.js';
import { starlightHead } from '../../src/route-meta.js';
import { buildSeoHead, buildSeoTitle } from '../../src/seo.js';
import { formatDate, isoDate } from '../../src/date-utils.js';
import { contentStyles } from '../../src/content-styles.js';
import { annotateImages } from '../../src/annotate-images.js';

// Register components used in render()
import '../../src/components/starlight-page.js';

/**
 * One blog post, at /blog/<slug>, from content/blog/<slug>.md.
 *
 * It is the doc page with no sidebar and no table of contents: the same
 * header, the same column, the same typography (src/content-styles.ts), the
 * same image sizing (src/annotate-images.ts), so a post reads like the rest
 * of the site. What differs is a date under the title and a way back to the
 * list.
 */
export interface BlogPostData {
  /** Only what render() reads, for the same reason as the doc page. */
  post: { title: string; date: string };
  body: string;
  siteTitle: string;
  nav: typeof siteConfig.nav;
  currentSlug: string;
  /** Raw <head> HTML for this post. See src/seo.ts. */
  seoHead: string;
  /** Overrides routeMeta.title, which is static per file. */
  seoTitle: string;
}

export const pageData = definePageData(async (event) => {
  const slug = event.context.params?.slug ?? '';

  // Content URLs are /content/blog/<slug> (contentDir = 'content').
  const posts = await getPosts();
  const post = posts.find(p => p.url === `/content/blog/${slug}`);

  if (!post) {
    throw createError({ statusCode: 404, message: `Post not found: ${slug}` });
  }

  // Dynamic imports for the same reason as in pages/docs/[slug].ts: the
  // grammars and node:fs must not reach the client bundle.
  const { applyHighlighting } = await import('../../src/highlight.js');
  const { imageSize } = await import('../../src/image-size.js');
  const body = annotateImages(
    applyHighlighting(addHeadingIds(post.body)),
    `content/blog/${slug}.md`,
    imageSize,
  );
  const title = post.title || slug;
  const description = post.description || siteConfig.description;

  return {
    post: { title, date: isoDate(post.date) },
    body,
    siteTitle: siteConfig.title,
    nav: siteConfig.nav,
    currentSlug: slug,
    seoTitle: buildSeoTitle(title),
    seoHead: buildSeoHead({
      title,
      description,
      path: `/blog/${slug}`,
      type: 'article',
    }),
  } satisfies BlogPostData;
});

export async function generateRoutes(): Promise<string[]> {
  const posts = await getPosts();
  return posts
    .filter(p => p.url.startsWith('/content/blog/'))
    .map(p => '/blog' + p.url.slice('/content/blog'.length));
}

export const routeMeta = {
  head: starlightHead,
  title: 'Blog — swoop',
};

@customElement('page-blog-slug')
export class BlogPostPage extends LitroPage {
  static override styles = [pageReset, contentStyles];

  override render() {
    const data = this.serverData as BlogPostData | null;
    if (!data?.post) return html`<p>Loading&hellip;</p>`;

    // ${unsafeHTML(data.body)} renders the Markdown-generated HTML directly.
    // content/blog is trusted-author-only, like content/docs.
    return html`
      <starlight-page
        siteTitle="${data.siteTitle}"
        pageTitle="${data.post.title}"
        .nav="${data.nav}"
        currentPath="/blog/${data.currentSlug}"
        nosidebar
      >
        <div slot="content">
          <p style="margin-top:-0.5rem;font-size:var(--sl-text-sm);color:var(--sl-color-gray-4);">
            <time datetime="${data.post.date}">${formatDate(data.post.date)}</time>
          </p>

          ${unsafeHTML(data.body)}

          <nav style="
            padding-top:2rem;
            margin-top:2rem;
            border-top:1px solid var(--sl-color-border);
            font-size:var(--sl-text-sm);
          " aria-label="Back to the blog">
            <a href="/blog" style="color:var(--sl-color-accent);text-decoration:none;">
              ← All posts
            </a>
          </nav>
        </div>
      </starlight-page>
    `;
  }
}

export default BlogPostPage;
