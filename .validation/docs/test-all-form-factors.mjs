import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({
    executablePath: '/usr/bin/chromium-browser',
    headless: true,
  });

  // Define form factors to test
  const formFactors = [
    {
      name: 'iPhone 16 (Mobile)',
      viewport: { width: 393, height: 852 },
      deviceScaleFactor: 3,
      isMobile: true,
      hasTouch: true,
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1'
    },
    {
      name: 'Desktop Portrait',
      viewport: { width: 1080, height: 1920 }, // 9:16 aspect ratio
      deviceScaleFactor: 1,
      isMobile: false,
      hasTouch: false,
    },
    {
      name: 'Desktop Landscape',
      viewport: { width: 1920, height: 1080 }, // 16:9 aspect ratio
      deviceScaleFactor: 1,
      isMobile: false,
      hasTouch: false,
    },
    {
      name: 'Tablet 4:3',
      viewport: { width: 1024, height: 768 }, // 4:3 aspect ratio (iPad)
      deviceScaleFactor: 2,
      isMobile: true,
      hasTouch: true,
      userAgent: 'Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1'
    },
  ];

  const pages = [
    { url: 'http://localhost:4321/gh-aw/', name: 'home' },
    { url: 'http://localhost:4321/gh-aw/introduction/overview/', name: 'content' },
  ];

  for (const formFactor of formFactors) {
    console.log(`\n=== Testing: ${formFactor.name} ===`);
    
    const context = await browser.newContext({
      viewport: formFactor.viewport,
      deviceScaleFactor: formFactor.deviceScaleFactor,
      isMobile: formFactor.isMobile,
      hasTouch: formFactor.hasTouch,
      userAgent: formFactor.userAgent,
    });

    const page = await context.newPage();

    for (const testPage of pages) {
      console.log(`Loading ${testPage.name}...`);
      await page.goto(testPage.url, { waitUntil: 'networkidle' });
      
      // Calculate crop height to maintain 9:16 aspect ratio max
      const { width, height } = formFactor.viewport;
      const maxAspectRatio = 9 / 16; // Height/Width
      const currentAspectRatio = height / width;
      
      let cropHeight = height;
      if (currentAspectRatio > maxAspectRatio) {
        // Crop to 9:16
        cropHeight = Math.floor(width * maxAspectRatio);
      }

      const filename = `/tmp/${formFactor.name.toLowerCase().replace(/\s+/g, '-')}-${testPage.name}.png`;
      
      // Take screenshot with crop if needed
      if (cropHeight < height) {
        await page.screenshot({ 
          path: filename,
          clip: { x: 0, y: 0, width: width, height: cropHeight }
        });
        console.log(`  Cropped screenshot (${width}x${cropHeight}) saved to ${filename}`);
      } else {
        await page.screenshot({ 
          path: filename,
          fullPage: false
        });
        console.log(`  Screenshot (${width}x${height}) saved to ${filename}`);
      }
    }

    await context.close();
  }

  await browser.close();
  console.log('\n✓ All form factors tested!');
})();
