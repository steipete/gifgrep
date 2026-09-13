import { escapeHtml } from "./docs-html.mjs";

export function highlightCode(code, lang) {
  const language = (lang || "text").toLowerCase();
  if (language === "bash" || language === "sh" || language === "shell" || language === "zsh" || language === "console") {
    return highlightShell(code);
  }
  if (language === "json" || language === "json5") return highlightJson(code);
  if (language === "ts" || language === "typescript" || language === "js" || language === "javascript" || language === "tsx" || language === "jsx") {
    return highlightJs(code);
  }
  if (language === "go" || language === "golang") return highlightGo(code);
  if (language === "yaml" || language === "yml") return highlightYaml(code);
  return escapeHtml(code);
}

function stashToken(idx) {
  return String.fromCharCode(0xe000 + idx);
}

function restoreStashTokens(value, stash) {
  return value.replace(/[\ue000-\uf8ff]/g, (token) => {
    const idx = token.charCodeAt(0) - 0xe000;
    return stash[idx] ?? "";
  });
}

function withStash(code, patterns) {
  const stash = [];
  let working = code;
  for (const [re, cls] of patterns) {
    working = working.replace(re, (match) => {
      const idx = stash.length;
      stash.push(`<span class="${cls}">${escapeHtml(match)}</span>`);
      return stashToken(idx);
    });
  }
  return restoreStashTokens(escapeHtml(working), stash);
}

function highlightShell(code) {
  return code
    .split("\n")
    .map((line) => {
      if (/^\s*#/.test(line)) return `<span class="hl-c">${escapeHtml(line)}</span>`;
      const promptMatch = line.match(/^(\s*)([$#>])(\s+)(.*)$/);
      if (promptMatch) {
        const [, lead, sym, gap, rest] = promptMatch;
        return `${escapeHtml(lead)}<span class="hl-p">${escapeHtml(sym)}</span>${escapeHtml(gap)}${highlightShellLine(rest)}`;
      }
      return highlightShellLine(line);
    })
    .join("\n");
}

function highlightShellLine(line) {
  const stash = [];
  const stashAdd = (match, cls) => {
    const idx = stash.length;
    stash.push(`<span class="${cls}">${escapeHtml(match)}</span>`);
    return stashToken(idx);
  };
  let working = line;
  working = working.replace(/(?:'[^']*'|"[^"]*")/g, (m) => stashAdd(m, "hl-s"));
  working = working.replace(/\s#.*$/g, (m) => stashAdd(m, "hl-c"));
  working = working.replace(/https?:\/\/[^\s]+/g, (m) => stashAdd(m, "hl-url"));
  working = working.replace(/\[[^\]]+\]/g, (m) => stashAdd(m, "hl-m"));
  working = working.replace(/(^|\s)(--?[A-Za-z][A-Za-z0-9-]*)/g, (_, lead, flag) => `${escapeHtml(lead)}${stashAdd(flag, "hl-f")}`);
  working = working.replace(/\b(gifgrep|brew|go|git|gh|make|sudo|cd|export|cat|curl|jq|ls|mv|cp|rm|mkdir|docker|tail|node|npm|pnpm|yarn)\b/g, (m) => stashAdd(m, "hl-cmd"));
  working = working.replace(/\b(\d+(?:\.\d+)?)\b/g, (m) => stashAdd(m, "hl-n"));
  return restoreStashTokens(escapeHtml(working), stash);
}

function highlightJson(code) {
  return withStash(code, [
    [/"(?:\\.|[^"\\])*"\s*:/g, "hl-k"],
    [/"(?:\\.|[^"\\])*"/g, "hl-s"],
    [/\b(true|false|null)\b/g, "hl-m"],
    [/-?\b\d+(?:\.\d+)?(?:e[+-]?\d+)?\b/gi, "hl-n"],
  ]);
}

function highlightJs(code) {
  return withStash(code, [
    [/\/\/[^\n]*/g, "hl-c"],
    [/\/\*[\s\S]*?\*\//g, "hl-c"],
    [/`(?:\\.|[^`\\])*`/g, "hl-s"],
    [/"(?:\\.|[^"\\])*"/g, "hl-s"],
    [/'(?:\\.|[^'\\])*'/g, "hl-s"],
    [/\b(const|let|var|function|return|if|else|for|while|switch|case|break|continue|class|extends|new|import|from|export|default|async|await|try|catch|finally|throw|typeof|instanceof|interface|type|enum|as|of|in|null|undefined|true|false|this)\b/g, "hl-k"],
    [/\b(\d+(?:\.\d+)?)\b/g, "hl-n"],
  ]);
}

function highlightGo(code) {
  return withStash(code, [
    [/\/\/[^\n]*/g, "hl-c"],
    [/\/\*[\s\S]*?\*\//g, "hl-c"],
    [/`[^`]*`/g, "hl-s"],
    [/"(?:\\.|[^"\\])*"/g, "hl-s"],
    [/\b(package|import|func|return|if|else|for|range|switch|case|break|continue|default|type|struct|interface|map|chan|go|defer|select|var|const|nil|true|false|iota)\b/g, "hl-k"],
    [/\b(\d+(?:\.\d+)?)\b/g, "hl-n"],
  ]);
}

function highlightYaml(code) {
  return code
    .split("\n")
    .map((line) => {
      if (/^\s*#/.test(line)) return `<span class="hl-c">${escapeHtml(line)}</span>`;
      const m = line.match(/^(\s*-?\s*)([A-Za-z0-9_.-]+)(\s*:)(.*)$/);
      if (m) {
        const [, lead, key, colon, rest] = m;
        return `${escapeHtml(lead)}<span class="hl-k">${escapeHtml(key)}</span>${escapeHtml(colon)}${highlightYamlValue(rest)}`;
      }
      return escapeHtml(line);
    })
    .join("\n");
}

function highlightYamlValue(rest) {
  if (!rest.trim()) return escapeHtml(rest);
  const trimmed = rest.trim();
  if (/^["'].*["']$/.test(trimmed)) {
    return escapeHtml(rest.replace(trimmed, "")) + `<span class="hl-s">${escapeHtml(trimmed)}</span>`;
  }
  if (/^(true|false|null|~)$/i.test(trimmed)) {
    return escapeHtml(rest.replace(trimmed, "")) + `<span class="hl-m">${escapeHtml(trimmed)}</span>`;
  }
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    return escapeHtml(rest.replace(trimmed, "")) + `<span class="hl-n">${escapeHtml(trimmed)}</span>`;
  }
  return escapeHtml(rest);
}

