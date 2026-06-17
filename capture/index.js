const { chromium } = require('playwright');
const path = require('path');

const target = process.env.TARGET_DESTINATION || 'localhost:3000';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(target);
  const capture_filename = "screenshot.png";
  const result_path = path.join("/", "out", capture_filename);
  await page.screenshot({ path: result_path });
  await browser.close();
  console.log(`capture stored at: ${result_path}`);
})();

