#!/usr/bin/env node
const fs = require("fs");
const path = require("path");

function argValue(name, fallback) {
  const index = process.argv.indexOf(name);
  if (index === -1 || index + 1 >= process.argv.length) return fallback;
  return process.argv[index + 1];
}

function stdinInputs() {
  if (process.stdin.isTTY) return {};
  const body = fs.readFileSync(0, "utf8").trim();
  if (!body) return {};
  return JSON.parse(body);
}

async function main() {
  const inputs = stdinInputs();
  const url = argValue("--url", inputs.url || "https://example.com");
  const output = argValue("--output", inputs.output || ".agentd-work/screenshot.png");
  const result = { url, screenshot: output, title: "", status: "fixture" };

  try {
    const puppeteer = require("puppeteer");
    const browser = await puppeteer.launch({ headless: "new" });
    const page = await browser.newPage();
    await page.setViewport({ width: 1365, height: 900 });
    await page.goto(url, { waitUntil: "networkidle2", timeout: 30000 });
    fs.mkdirSync(path.dirname(output), { recursive: true });
    await page.screenshot({ path: output, fullPage: true });
    result.title = await page.title();
    result.status = "captured";
    await browser.close();
  } catch (error) {
    const fixturePath = path.join(__dirname, "..", "fixtures", "website_metadata.json");
    Object.assign(result, JSON.parse(fs.readFileSync(fixturePath, "utf8")));
    result.error = String(error.message || error);
  }

  console.log(JSON.stringify(result, null, 2));
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
