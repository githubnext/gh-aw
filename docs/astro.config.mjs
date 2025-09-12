// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'GitHub Agentic Workflows',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/githubnext/gh-aw' }],
			sidebar: [
				{
					label: 'Reference',
					autogenerate: { directory: 'reference' },
				},
			],
		}),
	],
});
