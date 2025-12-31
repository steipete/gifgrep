import { spawnSync } from "node:child_process";

let args = process.argv.slice(2);
if (args.length > 0 && args[0] === "--") args = args.slice(1);

const result = spawnSync("go", ["run", "./cmd/gifgrep", ...args], {
  stdio: "inherit",
});

process.exit(result.status ?? 1);
