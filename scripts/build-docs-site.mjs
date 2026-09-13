#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

import { createMarkdownRenderer } from "./docs-markdown.mjs";
import { escapeHtml, escapeAttr } from "./docs-html.mjs";

import { css, faviconSvg, js, preThemeScript, themeToggleHtml } from "./docs-site-assets.mjs";

const root = process.cwd();
const docsDir = path.join(root, "docs");
const outDir = docsDir;
const repoBase = "https://github.com/steipete/gifgrep";
const repoEditBase = `${repoBase}/edit/main/docs`;
const cname = readCname();
const siteBase = cname ? `https://${cname}` : "";

const productName = "gifgrep";
const productTagline = "GIF search for terminals";
const productDescription =
  "A tiny Go CLI/TUI for searching animated GIFs from GIPHY or KLIPY, piping URLs or JSON, and previewing results inline in modern terminals.";
const brewInstall = "brew install steipete/tap/gifgrep";

const sections = [
  ["Start", ["index.md", "install.md", "quickstart.md", "commands.md"]],
  ["Use", ["search.md", "tui.md", "still.md", "sheet.md", "json.md"]],
  ["Previews", ["previews.md", "kitty.md", "iterm.md", "sixel.md"]],
  ["Providers", ["providers/index.md", "providers/auto.md", "providers/giphy.md", "providers/klipy.md"]],
  ["Reference", ["gif-sources.md"]],
];

// Keep internal notes and generated command aggregates out of the public site.
const buildExcludes = [/^refactor\//, /^commands\.generated\.md$/];

fs.mkdirSync(outDir, { recursive: true });

const allPages = allMarkdown(docsDir).map((file) => {
  const rel = path.relative(docsDir, file).replaceAll(path.sep, "/");
  const raw = fs.readFileSync(file, "utf8");
  const { frontmatter, body } = parseFrontmatter(raw);
  const cleaned = stripStrayDirectives(body);
  const title = frontmatter.title || firstHeading(cleaned) || titleize(path.basename(rel, ".md"));
  return { file, rel, title, outRel: outPath(rel, frontmatter), markdown: cleaned, frontmatter };
});

const pages = allPages.filter((page) => !buildExcludes.some((re) => re.test(page.rel)));
const pageMap = new Map(pages.map((page) => [page.rel, page]));
const permalinkMap = new Map();
for (const page of pages) {
  if (page.frontmatter.permalink) {
    permalinkMap.set(normalizePermalink(page.frontmatter.permalink), page);
  }
}

const nav = sections
  .map(([name, rels]) => ({
    name,
    pages: rels.map((rel) => pageMap.get(rel)).filter(Boolean),
  }))
  .filter((section) => section.pages.length);

const sectionByRel = new Map();
for (const section of nav) for (const page of section.pages) sectionByRel.set(page.rel, section.name);
const orderedPages = nav.flatMap((s) => s.pages);

const markdownToHtml = createMarkdownRenderer(rewriteHref);

for (const page of pages) {
  const html = markdownToHtml(page.markdown, page.rel);
  const toc = tocFromHtml(html);
  const idx = orderedPages.findIndex((p) => p.rel === page.rel);
  const prev = idx > 0 ? orderedPages[idx - 1] : null;
  const next = idx >= 0 && idx < orderedPages.length - 1 ? orderedPages[idx + 1] : null;
  const sectionName = sectionByRel.get(page.rel) || "Reference";
  const pageOut = path.join(outDir, page.outRel);
  fs.mkdirSync(path.dirname(pageOut), { recursive: true });
  fs.writeFileSync(pageOut, layout({ page, html, toc, prev, next, sectionName }), "utf8");
}

fs.writeFileSync(path.join(outDir, "favicon.svg"), faviconSvg(), "utf8");
fs.writeFileSync(path.join(outDir, ".nojekyll"), "\n", "utf8");
if (cname) fs.writeFileSync(path.join(outDir, "CNAME"), `${cname}\n`, "utf8");
validateLinks(outDir);
fs.writeFileSync(path.join(outDir, "llms.txt"), llmsTxt(), "utf8");
console.log(`built docs site: ${path.relative(root, outDir)}`);

function llmsTxt() {
  const origin = siteBase.replace(/\/$/, "");
  const source = repoBase;
  const name = productName;
  const description = productDescription;
  const install = brewInstall;
  const docPages = docsLlmsPages().map((page) => `- ${page.title}: ${pageUrl(origin, page.outRel)}`);
  const lines = [
    `# ${name}`,
    "",
    description,
    "",
    "Canonical documentation:",
    ...docPages,
  ];
  if (install) {
    lines.push("", "Install:", `- ${install}`);
  }
  if (source) {
    lines.push("", `Source: ${source}`);
  }
  lines.push("", "Guidance for agents:", "- Prefer the canonical documentation URLs above over README excerpts or package metadata.", "- Fetch only the pages needed for the current task; this is an index, not a full-site corpus.");
  return `${lines.join("\n")}\n`;
}

function docsLlmsPages() {
  const seen = new Set();
  return [...orderedPages, ...pages].filter((page) => page.outRel && !seen.has(page.outRel) && seen.add(page.outRel));
}

function pageUrl(origin, outRel) {
  const normalized = outRel === "index.html" ? "" : outRel.replace(/(?:^|\/)index\.html$/, (match) => match === "index.html" ? "" : "/");
  if (!origin) return normalized || "index.html";
  return normalized ? `${origin}/${normalized}` : `${origin}/`;
}

function readCname() {
  for (const candidate of [path.join(docsDir, "CNAME"), path.join(root, "CNAME")]) {
    if (fs.existsSync(candidate)) return fs.readFileSync(candidate, "utf8").trim();
  }
  return "";
}

function parseFrontmatter(raw) {
  const match = raw.match(/^---\n([\s\S]*?)\n---\n?/);
  if (!match) return { frontmatter: {}, body: raw };
  const fm = {};
  for (const line of match[1].split("\n")) {
    const m = line.match(/^([A-Za-z0-9_-]+):\s*(.*?)\s*$/);
    if (!m) continue;
    let value = m[2];
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    fm[m[1]] = value;
  }
  return { frontmatter: fm, body: raw.slice(match[0].length) };
}

function stripStrayDirectives(body) {
  return body
    .replace(/\r\n/g, "\n")
    .split("\n")
    .filter((line) => !/^\s*\{:\s*[^}]*\}\s*$/.test(line))
    .map((line) => line.replace(/\s*\{:\s*[^}]*\}\s*$/, ""))
    .join("\n");
}

function normalizePermalink(value) {
  let v = value.trim();
  if (!v) return "/";
  if (!v.startsWith("/")) v = `/${v}`;
  if (v.length > 1 && v.endsWith("/")) v = v.slice(0, -1);
  return v;
}

function allMarkdown(dir) {
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .flatMap((entry) => {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) return allMarkdown(full);
      return entry.name.endsWith(".md") ? [full] : [];
    })
    .sort();
}

function outPath(rel, frontmatter = {}) {
  if (frontmatter.permalink) {
    const permalink = normalizePermalink(frontmatter.permalink);
    if (permalink === "/") return "index.html";
    return `${permalink.slice(1)}/index.html`;
  }
  if (rel === "index.md") return "index.html";
  if (rel === "README.md") return "index.html";
  if (rel.endsWith("/README.md")) return rel.replace(/README\.md$/, "index.html");
  return rel.replace(/\.md$/, ".html");
}

function firstHeading(markdown) {
  return markdown.match(/^#\s+(.+)$/m)?.[1]?.trim();
}

function titleize(input) {
  return input.replaceAll("-", " ").replace(/\b\w/g, (m) => m.toUpperCase());
}

function rewriteHref(href, currentRel) {
  if (/^(https?:|mailto:|tel:|#)/.test(href)) return href;
  const [raw, hash = ""] = href.split("#");
  if (!raw) return hash ? `#${hash}` : "";
  if (raw.startsWith("/")) {
    const target = permalinkMap.get(normalizePermalink(raw));
    if (target) {
      const currentOut = pageMap.get(currentRel)?.outRel || outPath(currentRel);
      const out = hrefToOutRel(target.outRel, currentOut);
      return hash ? `${out}#${hash}` : out;
    }
    return href;
  }
  if (!raw.endsWith(".md")) return href;
  const from = path.posix.dirname(currentRel);
  const target = path.posix.normalize(path.posix.join(from, raw));
  let rewritten = pageMap.get(target)?.outRel || outPath(target);
  const currentOut = pageMap.get(currentRel)?.outRel || outPath(currentRel);
  rewritten = hrefToOutRel(rewritten, currentOut);
  return `${rewritten}${hash ? `#${hash}` : ""}`;
}

function tocFromHtml(html) {
  const items = [];
  const re = /<h([23]) id="([^"]+)">([\s\S]*?)<\/h[23]>/g;
  let m;
  while ((m = re.exec(html))) {
    const text = m[3]
      .replace(/<a class="anchor"[^>]*>.*?<\/a>/, "")
      .replace(/<[^>]+>/g, "")
      .trim();
    items.push({ level: Number(m[1]), id: m[2], text });
  }
  if (items.length < 2) return "";
  return `<nav class="toc" aria-label="On this page"><h2>On this page</h2>${items
    .map((i) => `<a class="toc-l${i.level}" href="#${i.id}">${escapeHtml(i.text)}</a>`)
    .join("")}</nav>`;
}

function isHomePage(page) {
  if (page.frontmatter.permalink && normalizePermalink(page.frontmatter.permalink) === "/") return true;
  return page.rel === "index.md" || page.rel === "README.md";
}

function homeHero(page) {
  const description = page.frontmatter.description || productDescription;
  const installRel = pageMap.get("install.md")?.outRel
    ? hrefToOutRel(pageMap.get("install.md").outRel, page.outRel)
    : "install.html";
  const quickstartRel = pageMap.get("quickstart.md")?.outRel
    ? hrefToOutRel(pageMap.get("quickstart.md").outRel, page.outRel)
    : "quickstart.html";
  const services = ["CLI", "TUI", "GIPHY", "KLIPY", "JSON", "Kitty", "Ghostty", "iTerm2", "Sixel", "ANSI"];
  return `<header class="home-hero">
        <div class="home-product" aria-label="${escapeAttr(productName)}">
          <span class="home-product-mark" aria-hidden="true">🧲</span>
          <span>${escapeHtml(productName)}</span>
        </div>
        <p class="eyebrow">GIF search · Terminal previews</p>
        <h1>${escapeHtml(productTagline)}</h1>
        <p class="lede">${escapeHtml(description)}</p>
        <div class="home-cta">
          <a class="btn btn-primary" href="${quickstartRel}">Quickstart</a>
          <a class="btn btn-ghost" href="${repoBase}" rel="noopener">GitHub</a>
          <div class="home-install" aria-label="Install with Homebrew">
            <span class="prompt" aria-hidden="true">$</span>
            <code>${escapeHtml(brewInstall)}</code>
          </div>
        </div>
        <div class="home-services" aria-label="Supported services">
          ${services.map((s) => `<span>${escapeHtml(s)}</span>`).join("")}
        </div>
        <p class="muted"><a href="${installRel}">Other install options →</a></p>
      </header>`;
}

function standardHero(page, sectionName, editUrl) {
  return `<header class="hero">
        <div class="hero-text">
          <p class="eyebrow">${escapeHtml(sectionName)}</p>
          <h1>${escapeHtml(page.title)}</h1>
        </div>
        <div class="hero-meta">
          <a class="repo" href="${repoBase}" rel="noopener">GitHub</a>
          <a class="edit" href="${escapeAttr(editUrl)}" rel="noopener">Edit page</a>
        </div>
      </header>`;
}

function layout({ page, html, toc, prev, next, sectionName }) {
  const depth = page.outRel.split("/").length - 1;
  const rootPrefix = depth ? "../".repeat(depth) : "";
  const editUrl = `${repoEditBase}/${page.rel}`;
  const home = isHomePage(page);
  const prevNext = !home && (prev || next) ? pageNavHtml(prev, next, page.outRel) : "";
  const heroBlock = home ? homeHero(page) : standardHero(page, sectionName, editUrl);
  const articleClass = home ? "doc doc-home" : "doc";
  const tocBlock = home ? "" : toc;
  const titleSuffix = home ? `${productName} — ${productTagline}` : `${page.title} — ${productName}`;
  const description = page.frontmatter.description || (home ? productDescription : `${page.title} — ${productName} CLI documentation.`);
  const canonicalUrl = pageCanonicalUrl(page);
  const socialImage = siteBase ? `${siteBase}/assets/gifgrep-tui.png` : `${rootPrefix}assets/gifgrep-tui.png`;
  const socialMeta = [
    ["link", "rel", "canonical", "href", canonicalUrl],
    ["meta", "property", "og:type", "content", "website"],
    ["meta", "property", "og:site_name", "content", productName],
    ["meta", "property", "og:title", "content", titleSuffix],
    ["meta", "property", "og:description", "content", description],
    ["meta", "property", "og:url", "content", canonicalUrl],
    ["meta", "property", "og:image", "content", socialImage],
    ["meta", "property", "og:image:width", "content", "1824"],
    ["meta", "property", "og:image:height", "content", "1209"],
    ["meta", "name", "twitter:card", "content", "summary_large_image"],
    ["meta", "name", "twitter:title", "content", titleSuffix],
    ["meta", "name", "twitter:description", "content", description],
    ["meta", "name", "twitter:image", "content", socialImage],
  ].map(tagHtml).join("\n  ");
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(titleSuffix)}</title>
  <meta name="description" content="${escapeAttr(description)}">
  ${socialMeta}
  <link rel="icon" href="${rootPrefix}favicon.svg" type="image/svg+xml">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script>${preThemeScript()}</script>
  <style>${css()}</style>
</head>
<body${home ? ' class="home"' : ""}>
  <button class="nav-toggle" type="button" aria-label="Toggle navigation" aria-expanded="false">
    <span aria-hidden="true"></span><span aria-hidden="true"></span><span aria-hidden="true"></span>
  </button>
  <div class="shell">
    <aside class="sidebar">
      <div class="sidebar-head">
        <a class="brand" href="${hrefToOutRel("index.html", page.outRel)}" aria-label="${productName} docs home">
          <span class="mark" aria-hidden="true"><i></i><i></i><i></i><i></i></span>
          <span><strong>${escapeHtml(productName)}</strong><small>terminal GIF docs</small></span>
        </a>
        ${themeToggleHtml()}
      </div>
      <label class="search"><span>Search</span><input id="doc-search" type="search" placeholder="search, tui, previews"></label>
      <nav>${navHtml(page)}</nav>
    </aside>
    <main>
      ${heroBlock}
      <div class="doc-grid${home ? " doc-grid-home" : ""}">
        <article class="${articleClass}">${html}${prevNext}</article>
        ${tocBlock}
      </div>
    </main>
  </div>
  <script>${js()}</script>
</body>
</html>`;
}

function pageCanonicalUrl(page) {
  if (!siteBase) return page.outRel;
  if (page.outRel === "index.html") return `${siteBase}/`;
  const rel = page.outRel.endsWith("/index.html") ? page.outRel.slice(0, -"index.html".length) : page.outRel;
  return `${siteBase}/${rel}`;
}

function tagHtml([tag, k1, v1, k2, v2]) {
  return tag === "link" ? `<link ${k1}="${v1}" ${k2}="${escapeAttr(v2)}">` : `<meta ${k1}="${v1}" ${k2}="${escapeAttr(v2)}">`;
}

function pageNavHtml(prev, next, currentOutRel) {
  const cell = (page, dir) => {
    if (!page) return "";
    return `<a class="page-nav-${dir}" href="${hrefToOutRel(page.outRel, currentOutRel)}"><small>${dir === "prev" ? "Previous" : "Next"}</small><span>${escapeHtml(page.title)}</span></a>`;
  };
  return `<nav class="page-nav" aria-label="Pager">${cell(prev, "prev")}${cell(next, "next")}</nav>`;
}

function navHtml(currentPage) {
  return nav
    .map((section) => `<section><h2>${escapeHtml(section.name)}</h2>${section.pages.map((page) => {
      const href = hrefToOutRel(page.outRel, currentPage.outRel);
      const active = page.rel === currentPage.rel ? " active" : "";
      return `<a class="nav-link${active}" href="${href}">${escapeHtml(navTitle(page))}</a>`;
    }).join("")}</section>`)
    .join("");
}

function navTitle(page) {
  if (page.rel === "index.md") return "Overview";
  if (page.rel === "commands/README.md") return "Command Index";
  return page.title.replace(/^`gifgrep\s*/, "").replace(/^`--source\s*/, "").replace(/`$/, "");
}

function hrefToOutRel(targetOutRel, currentOutRel) {
  const currentDir = path.posix.dirname(currentOutRel);
  if (targetOutRel.endsWith("/index.html")) {
    const targetDir = targetOutRel.slice(0, -"index.html".length);
    const rel = path.posix.relative(currentDir, targetDir || ".") || ".";
    return rel.endsWith("/") ? rel : `${rel}/`;
  }
  if (targetOutRel === "index.html") {
    const rel = path.posix.relative(currentDir, ".") || ".";
    return rel.endsWith("/") ? rel : `${rel}/`;
  }
  return path.posix.relative(currentDir, targetOutRel) || path.posix.basename(targetOutRel);
}

function validateLinks(outputDir) {
  const failures = [];
  // Generated command help can contain literal placeholder links.
  const placeholderHrefs = /^(url|path|file|dir|name)$/i;
  for (const file of allHtml(outputDir)) {
    const html = fs.readFileSync(file, "utf8");
    for (const match of html.matchAll(/href="([^"]+)"/g)) {
      const href = match[1];
      if (/^(#|https?:|mailto:|tel:|javascript:)/.test(href)) continue;
      if (placeholderHrefs.test(href)) continue;
      const [rawPath, anchor = ""] = href.split("#");
      const targetPath = rawPath
        ? path.resolve(path.dirname(file), rawPath)
        : file;
      const target = fs.existsSync(targetPath) && fs.statSync(targetPath).isDirectory()
        ? path.join(targetPath, "index.html")
        : targetPath;
      if (!fs.existsSync(target)) {
        failures.push(`${path.relative(outputDir, file)}: ${href} -> missing ${path.relative(outputDir, target)}`);
        continue;
      }
      if (anchor) {
        const targetHtml = fs.readFileSync(target, "utf8");
        if (!targetHtml.includes(`id="${anchor}"`) && !targetHtml.includes(`name="${anchor}"`)) {
          failures.push(`${path.relative(outputDir, file)}: ${href} -> missing anchor`);
        }
      }
    }
  }
  if (failures.length) {
    throw new Error(`broken docs links:\n${failures.join("\n")}`);
  }
}

function allHtml(dir) {
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .flatMap((entry) => {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) return allHtml(full);
      return entry.name.endsWith(".html") ? [full] : [];
    })
    .sort();
}
