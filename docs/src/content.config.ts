import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { changelogsLoader } from 'starlight-changelogs/loader';

export const collections = {
	docs: defineCollection({ loader: docsLoader(), schema: docsSchema() }),
	changelogs: defineCollection({
		loader: changelogsLoader([
			{
				enabled: !(process.env.npm_lifecycle_event === 'dev' || process.env.npm_lifecycle_event === 'start'), // disable in dev mode (detects npm run dev/start)
				provider: 'github',       // use GitHub releases as changelog source
				base: 'changelog',        // base path for changelog pages
				owner: 'githubnext',      // GitHub org/user
				repo: 'gh-aw',            // GitHub repo
				// Use GitHub token if available in environment, otherwise rely on public API
				...(process.env.GITHUB_TOKEN && { token: process.env.GITHUB_TOKEN }),
				// No process filter: include all releases
			},
		]),
	}),
};
