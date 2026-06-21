import path from 'path';
import { chromium } from 'playwright';

const result = path.join(process.cwd(), "out", process.env.OUTPUT_FILENAME || 'card.png');
(async () => {
  const browser = await chromium.launch({
    headless: true,
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-web-security',
      '--disable-features=IsolateOrigins,site-per-process',
      '--disable-gpu',
      '--use-gl=angle',
      '--use-angle=swiftshader',
      '--font-render-hinting=none',
      '--force-color-profile=srgb',
      '--disable-features=Translate',
      '--enable-font-antialiasing',
      '--hide-scrollbars',
      '--disable-lcd-text',
    ]
  });
  const context = await browser.newContext({
    viewport: {
      width: width,
      height: height,
    },
    deviceScaleFactor: 2,
    colorScheme: 'dark',
  });
  const page = await context.newPage();
  await page.goto(`http://localhost:${process.env.PORT}`, {
    waitUntil: 'networkidle',
  });
  await page.waitForTimeout(1000);
  await page.evaluate(async () => {
    document.fonts.ready
  });
  await page.screenshot({
    path: result,
    type: 'png',
  });
  await browser.close();
  console.log(`exported card to: ${result}`);
})();
