import { test, expect } from '@playwright/test';

test.describe('Author Avatar URLs', () => {
  const authorAvatars = [
    { name: 'githubnext', url: 'https://avatars.githubusercontent.com/githubnext' },
    { name: 'dsyme', url: 'https://avatars.githubusercontent.com/dsyme' },
    { name: 'pelikhan', url: 'https://avatars.githubusercontent.com/pelikhan' },
  ];

  for (const author of authorAvatars) {
    test(`should validate ${author.name} avatar URL is accessible`, async ({ request }) => {
      const response = await request.get(author.url);
      
      // Verify the URL returns a successful response
      expect(response.ok()).toBeTruthy();
      expect(response.status()).toBe(200);
      
      // Verify it returns an image content type
      const contentType = response.headers()['content-type'];
      expect(contentType).toMatch(/^image\//);
    });
  }

  test('should display author avatars on blog post', async ({ page }) => {
    // Navigate to a blog post
    await page.goto('/gh-aw/blog/2026-01-13-meet-the-workflows/');
    await page.waitForLoadState('networkidle');
    
    // Check that author images are present
    const authorImages = page.locator('img[alt="Don Syme"], img[alt="Peli de Halleux"]');
    await expect(authorImages.first()).toBeVisible();
    
    // Verify the images have the correct src attributes pointing to GitHub avatars
    const images = await authorImages.all();
    for (const img of images) {
      const src = await img.getAttribute('src');
      expect(src).toMatch(/avatars\.githubusercontent\.com/);
    }
  });

  test('should display author avatars on author page', async ({ page }) => {
    // Navigate to an author page
    await page.goto('/gh-aw/blog/authors/don-syme/');
    await page.waitForLoadState('networkidle');
    
    // Check that the author image is present
    const authorImage = page.locator('img[alt="Don Syme"]');
    await expect(authorImage.first()).toBeVisible();
    
    // Verify the image has the correct src attribute pointing to GitHub avatar
    const src = await authorImage.first().getAttribute('src');
    expect(src).toMatch(/avatars\.githubusercontent\.com\/dsyme/);
  });
});
