#!/usr/bin/env node
import { spawn } from 'child_process';
import fs from 'fs';
import http from 'http';
import path from 'path';
import process from 'process';
import { fileURLToPath } from 'url';

const repoRoot = path.resolve(fileURLToPath(new URL('.', import.meta.url)), '..');
const preferredPort = process.env.GHOSTTY_WEB_PORT
  ? Number(process.env.GHOSTTY_WEB_PORT)
  : 0;
const query = process.env.GIFGREP_QUERY || 'cats';
const outputPath =
  process.env.GIFGREP_SNAPSHOT || path.join(repoRoot, 'ghostty-web-snap.png');
const demoCmd = process.env.GHOSTTY_WEB_CMD || 'npx';
const demoArgs = (process.env.GHOSTTY_WEB_ARGS || '')
  .split(' ')
  .filter(Boolean);

function waitForServer(url, timeoutMs = 20000) {
  const deadline = Date.now() + timeoutMs;
  return new Promise((resolve, reject) => {
    const attempt = () => {
      const req = http.get(url, (res) => {
        res.resume();
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 400) {
          resolve();
        } else if (Date.now() > deadline) {
          reject(new Error(`timeout waiting for ${url}`));
        } else {
          setTimeout(attempt, 250);
        }
      });
      req.on('error', () => {
        if (Date.now() > deadline) {
          reject(new Error(`timeout waiting for ${url}`));
        } else {
          setTimeout(attempt, 250);
        }
      });
    };
    attempt();
  });
}

function probeUrl(url) {
  return new Promise((resolve) => {
    const req = http.get(url, (res) => {
      res.resume();
      resolve(res.statusCode || 0);
    });
    req.on('error', () => resolve(0));
  });
}

async function pickDemoUrl(port) {
  const demoUrl = `http://localhost:${port}/demo/`;
  const demoStatus = await probeUrl(demoUrl);
  if (demoStatus >= 200 && demoStatus < 400) {
    return demoUrl;
  }
  return `http://localhost:${port}/`;
}

function randomPort() {
  return 62000 + Math.floor(Math.random() * 3000);
}

async function startDemo() {
  const attempts = [];
  for (let i = 0; i < 5; i++) {
    const port = preferredPort || randomPort();
    const demo = spawn(
      demoCmd,
      ['-y', '@ghostty-web/demo@next', ...demoArgs],
      {
        env: { ...process.env, PORT: String(port) },
        stdio: ['ignore', 'pipe', 'pipe'],
      }
    );

    demo.stdout.on('data', (chunk) => process.stdout.write(chunk));
    demo.stderr.on('data', (chunk) => process.stderr.write(chunk));

    const demoExited = new Promise((_, reject) => {
      demo.once('exit', (code) => reject(new Error(`demo exited (${code})`)));
    });

    try {
      await Promise.race([waitForServer(`http://localhost:${port}/`), demoExited]);
      const demoUrl = await pickDemoUrl(port);
      return { demo, demoUrl };
    } catch (err) {
      attempts.push(err);
      demo.kill('SIGINT');
    }
  }

  throw new Error(`failed to start demo (${attempts.length} attempts)`);
}

async function main() {
  const { demo, demoUrl } = await startDemo();
  let browser;

  try {
    let playwright;
    try {
      playwright = await import('playwright');
    } catch (err) {
      throw new Error('playwright not installed. Run: npm install && npx playwright install chromium');
    }

    browser = await playwright.chromium.launch();
    const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
    await page.goto(demoUrl, { waitUntil: 'networkidle' });

    await page.waitForFunction(() => {
      const text =
        document.querySelector('#status-text')?.textContent ||
        document.querySelector('#connection-text')?.textContent ||
        '';
      return text.includes('Connected');
    });

    await page.addStyleTag({
      content: `
      .terminal-window { max-width: 1200px; }
      #terminal, #terminal-container { height: 700px !important; }
    `,
    });
    await page.evaluate(() => window.dispatchEvent(new Event('resize')));

    const container = (await page.$('#terminal')) || (await page.$('#terminal-container'));
    if (!container) {
      throw new Error('terminal container not found');
    }
    await container.click();
    await page.keyboard.type(`cd ${repoRoot}\r`, { delay: 5 });
    await page.keyboard.type(
      `source ~/.profile >/dev/null 2>&1; go run ./cmd/gifgrep tui --source klipy ${query}\r`,
      { delay: 5 }
    );
    await page.waitForTimeout(4000);

    const terminalText = await page.evaluate(() => document.body?.textContent || '');
    if (/unknown flag|missing KLIPY_API_KEY|exit status/i.test(terminalText)) {
      throw new Error('gifgrep command failed in ghostty-web terminal');
    }

    const terminal = await page.$('.terminal-window');
    if (!terminal) {
      throw new Error('terminal window not found');
    }

    await terminal.screenshot({ path: outputPath });

    if (fs.existsSync(outputPath)) {
      console.log(`Saved ${outputPath}`);
    } else {
      throw new Error('failed to save screenshot');
    }
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    demo.kill('SIGINT');
  }

}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
