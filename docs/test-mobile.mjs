import { chromium, devices } from 'playwright';

(async () => {
  // Use the system chromium browser
  const browser = await chromium.launch({
    executablePath: '/usr/bin/chromium-browser',
    headless: true,
  });

  // iPhone 16 has dimensions 393x852 (portrait)
  const iPhone16 = {
    name: 'iPhone 16',
    viewport: { width: 393, height: 852 },
    deviceScaleFactor: 3,
    isMobile: true,
    hasTouch: true,
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1'
  };

  const context = await browser.newContext({
    ...iPhone16,
  });

  const page = await context.newPage();

  // Test homepage
  console.log('Loading homepage...');
  await page.goto('http://localhost:4321/gh-aw/', { waitUntil: 'networkidle' });
  await page.screenshot({ path: '/tmp/mobile-homepage.png', fullPage: true });
  console.log('Homepage screenshot saved to /tmp/mobile-homepage.png');

  // Test a content page
  console.log('Loading get started page...');
  await page.goto('http://localhost:4321/gh-aw/get-started/quickstart/', { waitUntil: 'networkidle' });
  await page.screenshot({ path: '/tmp/mobile-content.png', fullPage: true });
  console.log('Content page screenshot saved to /tmp/mobile-content.png');

  await browser.close();
  console.log('Done!');
})();
