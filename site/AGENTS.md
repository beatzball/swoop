# swoop docs site — agent instructions

This directory is the source for **https://swoop.sh**. It is a Litro
`supernova` site (the lit adapter) in SSG mode: Markdown in, static HTML out.
The docs half is the starlight layout; the landing page is supernova's.

Read this before changing anything in `site/`.

## Where things live

| Path | What it is |
|------|-----------|
| `content/docs/*.md` | Every documentation page. One file = one page. |
| `server/starlight.config.js` | Site title, top nav, and the sidebar tree. |
| `_data/metadata.js` | Site title, canonical URL, description (used for SEO and OG images). |
| `pages/index.ts` | The landing page (a Lit component, not Markdown). Its copy is in the lists at the top of the file. |
| `pages/docs/[slug].ts` | The doc page template. Do not edit to add a page. |
| `src/seo.ts` | Per-page description, canonical, Open Graph and Twitter tags. |
| `src/components/` | Shared UI, from the recipe. `starlight-header.ts` carries swoop's logo and GitHub button. |
| `public/` | The logo, the icons, and the demo recording as WebP. See below. |
| `scripts/cut-logo.mjs` | Cuts the icons from `public/logo.png`. |
| `Dockerfile`, `nginx.conf` | Deploy. Coolify builds these on push to `main`. |
| `../.github/workflows/site.yml` | Builds, tests, and calls the deploy hook. |

## Add a documentation page

1. Create `content/docs/<slug>.md`. The filename becomes the URL:
   `content/docs/foo.md` → `/docs/foo`.

2. Give it frontmatter. `title` and `description` are required:

   ```markdown
   ---
   title: Your Page Title
   description: One sentence. It is used for SEO and the OG image.
   sidebar:
     order: 8
   ---

   ## First heading

   Body starts here.
   ```

3. Add it to the sidebar in `server/starlight.config.js`. A page not listed
   there is still reachable by URL but invisible in the nav:

   ```js
   { label: 'Your Page Title', slug: 'your-page-slug' },
   ```

4. Add the route to `DOC_ROUTES` in `e2e/index.spec.ts`.

5. Build to verify (see below).

## Rules

- **Start the body at `##`, not `#`.** The `title` from frontmatter is already
  rendered as the page's `<h1>`. A `#` in the body makes a second one.
- **Slugs must be unique across the whole `content/` directory.** The build
  throws on a collision rather than silently dropping a page.
- **Internal links are absolute paths**: `/docs/ask-ai`, not `ask-ai.md`.
- **Do not edit `routes.generated.ts` or `server/stubs/page-manifest.ts`.**
  Both are regenerated on every build and are gitignored.
- **Never name a competing launcher**, anywhere on the site. The repository's
  `AGENTS.md` says why and what to say instead.
- **No home paths, names or emails.** Use `~/.config/...` or a placeholder like
  `/absolute/path/to/swoop`.
- **The README and the site say the same things.** The README is what GitHub
  shows; the site is the long form. When a README section changes, change the
  page that carries it (Getting Started, Ask AI, Extensions, Settings).
- **The extension guide is tested code.** Every block in
  `content/docs/writing-an-extension.md` was run through `bin/swoop`. If you
  change one, run the new script the same way before you commit it.

## The pictures

- `public/swoop.webp` is `demo/swoop.gif`, the README's recording, as animated
  WebP. Coolify builds with `site/` as the Docker context, so `../demo` is not
  there at build time: the file has to be committed here. After `vhs`
  re-records the tape:

  ```sh
  gif2webp -mixed -q 80 demo/swoop.gif -o site/public/swoop.webp   # brew install webp
  ```

- `public/logo.png` is the logo: the owl drawing, cut from the supplied art
  with the background and shadow removed, on a transparent ground.
  `logo.webp` (the pages), `apple-touch-icon.png`, `favicon-32.png` and
  `favicon.ico` are cut from it. Change the PNG, and cut all four again:

  ```sh
  node scripts/cut-logo.mjs && cwebp -q 92 public/logo.png -o public/logo.webp
  ```

### Images are sized at build time, and lazy below the first one

`pages/docs/[slug].ts` rewrites the rendered Markdown so every image except
the first carries `loading="lazy" decoding="async"`, and so every image
carries the `width` and `height` read out of its own header. Markdown image
syntax cannot carry an attribute, and the first image is usually above the
fold, so deferring it would delay the one thing the reader is waiting for.
Without the size pair the page reserves no space and every paragraph below
the image moves once the bytes arrive.

**Use a `.webp` or a `.png` and there is nothing to do.** Both are read
straight out of the file, so `![alt](/swoop.webp)` is enough. A `?v=2` cache
buster, a space or an accent in the filename are all handled.

**Anything else fails the build, by design.** A `.jpg`, a `.gif`, an `.svg`, a
remote URL or a path that is not there cannot be measured, and a page that
ships without the pair brings the layout shift back silently — so the build
stops and names the file rather than letting it through:

```
content/docs/extensions.md: 1 image(s) have no intrinsic size, ...
  /screenshot.jpg
```

Two ways out, and the message says both. Convert the image to WebP, which is
what the rest of the site uses. Or write the tag yourself with **both**
dimensions, which is the only opt-out:

```html
<img src="/screenshot.jpg" alt="..." width="1400" height="620">
```

One dimension is not enough and is reported too: the stylesheet sets
`height: auto`, so a lone `width` gives the browser no ratio and reserves no
box. A hand-written `<img>` that sets its own `loading` is left alone.

## Verify your change

```sh
cd site
pnpm install        # first time only
pnpm build          # must exit 0; prints every prerendered route
```

`pnpm build` is the real check. It fails on a duplicate slug, a missing
`title`, a broken component, or an image it cannot measure (see above), and it
prints the full route list so you can confirm your page is there.

For a live-reload loop while writing:

```sh
pnpm dev            # http://localhost:3000
```

End-to-end checks:

```sh
pnpm test:e2e            # both targets, in order
pnpm test:e2e:dev        # just `litro dev`
pnpm test:e2e:preview    # just the built output (rebuilds dist/ first)
```

The same specs run twice, against two different renderers. `dev` is Vite
serving modules from source; `preview` is the prerendered `dist/static` that
nginx ships in production. `preview` is the only check that opens what actually
ships, and what it catches on its own is client code that behaves differently
in the two builds — anything behind `import.meta.env.PROD`, anything the
minifier or tree-shaker rewrites. A build exits 0 and the route answers 200
either way, so nothing else in CI notices.

They run one after the other, never together, because `litro dev` deletes
`dist/` on startup and that is the directory `litro preview` serves.

**`dist/` is current only after the `preview` half has run**, because building
it is the first half of that target's server command. If you ran
`pnpm test:e2e:dev` on its own, or the `pnpm test:e2e` chain stopped when the
dev half failed, then `dist/static` is **empty** — and `litro preview` prints
its usual `Previewing static build at …` banner over it and serves 404 for
every route. Run `pnpm build` before you trust a preview.

**Do not check the built output with `python3 -m http.server`.** It serves
`/_litro/app.js` with a MIME type Chrome rejects for a module script, so every
page comes up blank with nothing in the console. That is the server, not the
site. Use `pnpm preview`, or the Docker image below.

## Deploy

Push to `main`. `.github/workflows/site.yml` builds the site, runs the e2e
suite, builds the image and probes it, then calls the Coolify deploy hook and
waits until https://swoop.sh/version.json reports the commit. Coolify rebuilds
from `site/Dockerfile` (Base Directory `/site`) and serves `dist/static` behind
nginx. The hook needs two repository secrets, `COOLIFY_WEBHOOK_URL` and
`COOLIFY_API_TOKEN`; without them the deploy step warns and does nothing.

To check the deploy locally exactly as production runs it:

```sh
cd site
docker build -t swoop-docs .
docker run --rm -p 8099:80 swoop-docs
# then open http://localhost:8099
```
