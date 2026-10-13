import { html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { LitroPage, pageReset } from '@beatzball/litro/runtime';
import { definePageData } from '@beatzball/litro';
import { getPosts } from 'litro:content';
import { siteConfig } from '../../server/starlight.config.js';
import { starlightHead } from '../../src/route-meta.js';
import { buildSeoHead, buildSeoTitle } from '../../src/seo.js';
import { formatDate, isoDate } from '../../src/date-utils.js';
import { contentStyles } from '../../src/content-styles.js';

// Register components used in render()
import '../../src/components/starlight-page.js';

/** The list of posts at /blog, newest first. */
export interface BlogIndexData {
  posts: Array<{ slug: string; title: string; date: string; description: string }>;
  siteTitle: string;
  nav: typeof siteConfig.nav;
  seoHead: string;
  seoTitle: string;
}

const TITLE = 'Blog';
const DESCRIPTION = 'Notes from building swoop: why it is made the way it is, and what changed.';

export const pageData = definePageData(async (_event) => {
  const all = await getPosts();
  // getPosts() is already newest first. Only the fields the list shows go to
  // the client: a body here would ship every post on the index page too.
  const posts = all
    .filter(p => p.url.startsWith('/content/blog/'))
    .map(p => ({
      slug: p.url.slice('/content/blog/'.length),
      title: p.title,
      date: isoDate(p.date),
      description: p.description ?? '',
    }));

  return {
    posts,
    siteTitle: siteConfig.title,
    nav: siteConfig.nav,
    seoTitle: buildSeoTitle(TITLE),
    seoHead: buildSeoHead({ title: TITLE, description: DESCRIPTION, path: '/blog' }),
  } satisfies BlogIndexData;
});

export const routeMeta = {
  head: starlightHead,
  title: 'Blog — swoop',
};

@customElement('page-blog')
export class BlogIndexPage extends LitroPage {
  static override styles = [pageReset, contentStyles];

  override render() {
    const data = this.serverData as BlogIndexData | null;
    const { posts = [], siteTitle = siteConfig.title, nav = [] } = data ?? {};

    // Each post is an <h2>, so the index paints a content heading the way
    // every other page does (e2e/index.spec.ts looks for one), and a reader
    // with a screen reader gets one heading per post.
    return html`
      <starlight-page
        siteTitle="${siteTitle}"
        pageTitle="${TITLE}"
        .nav="${nav}"
        currentPath="/blog"
        nosidebar
      >
        <div slot="content">
          ${posts.length === 0 ? html`<p>No posts yet.</p>` : posts.map(post => html`
            <h2 style="border:none;padding-bottom:0;margin-bottom:0.25rem;">
              <a href="/blog/${post.slug}">${post.title}</a>
            </h2>
            <p style="margin-bottom:0.5rem;font-size:var(--sl-text-sm);color:var(--sl-color-gray-4);">
              <time datetime="${post.date}">${formatDate(post.date)}</time>
            </p>
            ${post.description ? html`<p>${post.description}</p>` : ''}
          `)}
        </div>
      </starlight-page>
    `;
  }
}

export default BlogIndexPage;
