import { escapeHtml, escapeAttr } from "./docs-html.mjs";
import { highlightCode } from "./docs-highlight.mjs";

export function createMarkdownRenderer(rewriteHref) {
  function markdownToHtml(markdown, currentRel) {
    const lines = markdown.replace(/\r\n/g, "\n").split("\n");
    const html = [];
    let paragraph = [];
    let list = null;
    let fence = null;
    let blockquote = [];

    const flushParagraph = () => {
      if (!paragraph.length) return;
      html.push(`<p>${inline(paragraph.join(" "), currentRel)}</p>`);
      paragraph = [];
    };
    const closeList = () => {
      if (!list) return;
      html.push(`</${list}>`);
      list = null;
    };
    const flushBlockquote = () => {
      if (!blockquote.length) return;
      const inner = markdownToHtml(blockquote.join("\n"), currentRel);
      html.push(`<blockquote>${inner}</blockquote>`);
      blockquote = [];
    };
    const splitRow = (line) => {
      let trimmed = line.trim();
      if (trimmed.startsWith("|")) trimmed = trimmed.slice(1);
      if (trimmed.endsWith("|") && !trimmed.endsWith("\\|")) trimmed = trimmed.slice(0, -1);
      const cells = [];
      let current = "";
      for (let idx = 0; idx < trimmed.length; idx++) {
        const char = trimmed[idx];
        if (char === "\\" && trimmed[idx + 1] === "|") {
          current += "\\|";
          idx += 1;
          continue;
        }
        if (char === "|") {
          cells.push(current.trim().replace(/\\\|/g, "|"));
          current = "";
          continue;
        }
        current += char;
      }
      cells.push(current.trim().replace(/\\\|/g, "|"));
      return cells;
    };
    const isDivider = (line) => /^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)+\|?\s*$/.test(line);

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const fenceMatch = line.match(/^```([\w+-]+)?\s*$/);
      if (fenceMatch) {
        flushParagraph();
        closeList();
        flushBlockquote();
        if (fence) {
          const body = highlightCode(fence.lines.join("\n"), fence.lang);
          html.push(`<pre><code class="language-${escapeAttr(fence.lang)}">${body}</code></pre>`);
          fence = null;
        } else {
          fence = { lang: fenceMatch[1] || "text", lines: [] };
        }
        continue;
      }
      if (fence) {
        fence.lines.push(line);
        continue;
      }
      if (/^>\s?/.test(line)) {
        flushParagraph();
        closeList();
        blockquote.push(line.replace(/^>\s?/, ""));
        continue;
      }
      flushBlockquote();
      if (!line.trim()) {
        flushParagraph();
        closeList();
        continue;
      }
      if (/^\s*---+\s*$/.test(line)) {
        flushParagraph();
        closeList();
        html.push("<hr>");
        continue;
      }
      const heading = line.match(/^(#{1,4})\s+(.+)$/);
      if (heading) {
        flushParagraph();
        closeList();
        const level = heading[1].length;
        const text = heading[2].trim();
        const id = slug(text);
        const inner = inline(text, currentRel);
        if (level === 1) {
          html.push(`<h1 id="${id}">${inner}</h1>`);
        } else {
          html.push(`<h${level} id="${id}"><a class="anchor" href="#${id}" aria-label="Anchor link">#</a>${inner}</h${level}>`);
        }
        continue;
      }
      if (line.trimStart().startsWith("|") && line.includes("|", line.indexOf("|") + 1) && isDivider(lines[i + 1] || "")) {
        flushParagraph();
        closeList();
        const header = splitRow(line);
        const aligns = splitRow(lines[i + 1]).map((cell) => {
          const left = cell.startsWith(":");
          const right = cell.endsWith(":");
          return right && left ? "center" : right ? "right" : left ? "left" : "";
        });
        i += 1;
        const rows = [];
        while (i + 1 < lines.length && lines[i + 1].trimStart().startsWith("|")) {
          i += 1;
          rows.push(splitRow(lines[i]));
        }
        const th = header.map((c, idx) => `<th${aligns[idx] ? ` style="text-align:${aligns[idx]}"` : ""}>${inline(c, currentRel)}</th>`).join("");
        const tb = rows.map((r) => `<tr>${r.map((c, idx) => `<td${aligns[idx] ? ` style="text-align:${aligns[idx]}"` : ""}>${inline(c, currentRel)}</td>`).join("")}</tr>`).join("");
        html.push(`<table><thead><tr>${th}</tr></thead><tbody>${tb}</tbody></table>`);
        continue;
      }
      const bullet = line.match(/^\s*-\s+(.+)$/);
      const numbered = line.match(/^\s*\d+\.\s+(.+)$/);
      if (bullet || numbered) {
        flushParagraph();
        const tag = bullet ? "ul" : "ol";
        if (list && list !== tag) closeList();
        if (!list) {
          list = tag;
          html.push(`<${tag}>`);
        }
        html.push(`<li>${inline((bullet || numbered)[1], currentRel)}</li>`);
        continue;
      }
      paragraph.push(line.trim());
    }
    flushParagraph();
    closeList();
    flushBlockquote();
    return html.join("\n");
  }

  function inline(text, currentRel) {
    const stash = [];
    let out = text.replace(/`([^`]+)`/g, (_, code) => {
      stash.push(`<code>${escapeHtml(code)}</code>`);
      return `\u0000${stash.length - 1}\u0000`;
    });
    out = escapeHtml(out)
      .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, href) => mediaHtml(alt, href, currentRel))
      .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
      .replace(/(^|[^*])\*([^*\s][^*]*?)\*(?!\*)/g, "$1<em>$2</em>")
      .replace(/(^|[^_])_([^_\s][^_]*?)_(?!_)/g, "$1<em>$2</em>")
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, href) => `<a href="${escapeAttr(rewriteHref(href, currentRel))}">${label}</a>`)
      .replace(/&lt;(https?:\/\/[^\s<>]+)&gt;/g, '<a href="$1">$1</a>');
    out = out.replace(/\\\|/g, "|");
    out = out.replace(/&lt;br&gt;/g, "<br>");
    return out.replace(/\u0000(\d+)\u0000/g, (_, i) => stash[Number(i)]);
  }

  function mediaHtml(alt, href, currentRel) {
    const src = rewriteHref(href, currentRel);
    if (href.split(/[?#]/)[0].toLowerCase().endsWith(".mp4")) {
      const poster = src.replace(/\.mp4$/i, ".png");
      return `<video src="${escapeAttr(src)}" poster="${escapeAttr(poster)}" aria-label="${escapeAttr(alt)}" autoplay muted loop playsinline></video>`;
    }
    return `<img src="${escapeAttr(src)}" alt="${escapeAttr(alt)}">`;
  }


  return markdownToHtml;
}

function slug(text) {
  return text.toLowerCase().replace(/`/g, "").replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

