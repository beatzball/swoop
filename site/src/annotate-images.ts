/** Reports already printed by this build. See where it is read. */
const reported = new Set<string>();

/**
 * Give every image in the rendered Markdown the attributes it cannot carry
 * itself: an intrinsic size, and lazy loading below the first one.
 *
 * **width and height** are what stop the page jumping. Without them a
 * Markdown image reserves no space, so every paragraph below it moves once
 * the bytes arrive — /docs/pipes measured CLS 0.174 against Google's 0.1
 * limit. The pair is only a ratio here, because the stylesheet sets
 * `max-width:100%; height:auto`, so the browser reserves the right box at
 * whatever width the column happens to be.
 *
 * **loading** is deferred on every image except the first. The recordings are
 * the heaviest thing the site serves and a page often has one near the top
 * and others most readers never scroll to. The first stays eager: it is
 * usually above the fold and is the thing the reader is waiting for.
 *
 * **An image that cannot be sized stops the build.** sizeOf reads PNG and
 * WebP headers and returns null for everything else, which is the right
 * fallback — a wrong number is worse than none — but on its own it is silent:
 * add a .jpg and that page quietly reserves nothing again, the build passes,
 * the probe passes, and nothing measures CLS. Nobody noticing is the failure
 * mode, so the omission is raised here instead of shipped. The opt-out is
 * writing BOTH width and height yourself; see the message below.
 *
 * Reported and `process.exitCode`, deliberately NOT `throw`. litro's page
 * handler wraps every pageData fetcher in a try/catch that calls console.warn
 * and renders the page without data — "Data fetch failure is non-fatal" — so a
 * throw here is caught, `pnpm build` still exits 0, AND the page ships with
 * the Loading placeholder instead of its content. Measured both ways on the
 * fixture: throw gave exit 0 and a blank /docs/pipes; this gives exit 1 and a
 * complete page. Turning this back into a throw would silently restore the
 * bug it was written to close.
 *
 * Every image is checked before anything is reported, so one build names every
 * problem rather than one per round trip.
 */
export function annotateImages(
  html: string,
  where: string,
  sizeOf: (src: string) => { w: number; h: number } | null,
): string {
  let seen = 0;
  const unsized: string[] = [];
  const attr = (name: string, attrs: string) =>
    // Anchored to a quote or whitespace so `data-width=` cannot match.
    new RegExp(`[\\s"']${name}\\s*=`).test(' ' + attrs);

  const out = html.replace(/<img\b([^>]*)>/g, (tag, attrs: string) => {
    // Count every image, including ones skipped below. Counting only the ones
    // that get rewritten would make a hand-written eager <img> promote the
    // NEXT image to "first" as well, and neither would be deferred.
    const isFirst = ++seen === 1;
    const add: string[] = [];

    if (!isFirst && !attr('loading', attrs)) {
      add.push('loading="lazy"', 'decoding="async"');
    }

    const hasW = attr('width', attrs);
    const hasH = attr('height', attrs);
    const src = /\ssrc\s*=\s*["']([^"']+)["']/.exec(attrs)?.[1] ?? '(no src)';

    if (hasW && hasH) {
      // Both by hand: the author has taken the size on themselves. This is
      // the opt-out, and it is the only one.
    } else if (hasW || hasH) {
      // Exactly one. The stylesheet sets `max-width:100%; height:auto`, so a
      // lone dimension gives the browser no ratio and reserves no box — the
      // original CLS bug, on a tag that looks as though it were handled.
      //
      // Deliberately not completed from the header either: the author's
      // `width="100"` beside a measured `height="620"` is a ratio nobody
      // chose, which is exactly the wrong number src/image-size.ts refuses to
      // invent. Reported instead, so the author writes the other half.
      unsized.push(`${src}  (has ${hasW ? 'width' : 'height'}, needs both)`);
    } else {
      // Site-root paths only: anything remote has no file to read at build
      // time. It is collected rather than skipped, because an unsized remote
      // image shifts the layout exactly as much as an unsized local one.
      const local = src.startsWith('/') && !src.startsWith('//');
      const size = local ? sizeOf(src) : null;
      if (size) add.push(`width="${size.w}"`, `height="${size.h}"`);
      else unsized.push(src);
    }

    return add.length ? `<img${attrs} ${add.join(' ')}>` : tag;
  });

  if (unsized.length) {
    const report =
      `${where}: ${unsized.length} image(s) have no intrinsic size, so the page would ` +
      `reserve no space for them and the layout would shift as they load:\n` +
      unsized.map((s) => `  ${s}`).join('\n') +
      `\nsrc/image-size.ts reads PNG and WebP headers only, and nothing remote can be ` +
      `measured at build time. Convert the image to WebP, or write the tag in the ` +
      `Markdown with BOTH dimensions, which is the opt-out, e.g.\n` +
      `  <img src="..." alt="..." width="1400" height="620">`;

    // Printed once per distinct report. This fetcher runs twice for every doc
    // — litro's og-handler calls the same pageData to build /__og/<slug>.png —
    // so without this every list appears twice and a build with a few bad
    // pages buries each real one in its own duplicate.
    if (!reported.has(report)) {
      reported.add(report);
      console.error(report);
    }
    // Guarded because this function is defined in the page component's module
    // and so is emitted into the client bundle. Nothing calls it there today
    // (pageData is serialized into the HTML server-side, and vite.config.ts
    // stubs imageSize away), but `process` does not exist in a browser, and an
    // unguarded assignment would be a ReferenceError inside litro's catch —
    // which renders the Loading placeholder, the exact outcome the paragraph
    // above says this design avoids.
    if (typeof process !== 'undefined') process.exitCode = 1;
  }

  return out;
}
