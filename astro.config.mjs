import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
  site: 'https://githubnext.github.io',
  base: '/gh-aw',
  integrations: [
    starlight({
      title: 'GitHub Agentic Workflows',
      description: 'Write agentic workflows in natural language markdown, and run them safely in GitHub Actions.',
      social: {
        github: 'https://github.com/githubnext/gh-aw',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Quick Start', link: '/quick-start/' },
            { label: 'Concepts', link: '/concepts/' },
          ],
        },
        {
          label: 'Workflows Reference',
          items: [
            { label: 'Workflow Structure', link: '/workflow-structure/' },
            { label: 'Frontmatter Options', link: '/frontmatter/' },
            { label: 'Safe Output Processing', link: '/safe-outputs/' },
            { label: 'Include Directives', link: '/include-directives/' },
            { label: 'Command Triggers', link: '/command-triggers/' },
            { label: 'Tools Configuration', link: '/tools/' },
            { label: 'MCPs', link: '/mcps/' },
          ],
        },
        {
          label: 'Development Experience',
          items: [
            { label: 'Commands', link: '/commands/' },
            { label: 'Visual Studio Code', link: '/vscode/' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Samples', link: '/samples/' },
            { label: 'Secrets Management', link: '/secrets/' },
            { label: 'Security Notes', link: '/security-notes/' },
          ],
        },
      ],
    }),
  ],
  outDir: './docs-dist',
});