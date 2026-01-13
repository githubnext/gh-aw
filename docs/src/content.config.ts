import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { blogSchema } from 'starlight-blog/schema';
// import { changelogsLoader } from 'starlight-changelogs/loader';

export const collections = {
	docs: defineCollection({ 
		loader: docsLoader(), 
		schema: docsSchema({
			extend: (context) => blogSchema(context)
		})
	}),
	// changelogs: defineCollection({
	// 	loader: changelogsLoader([
	// 		{
	// 			provider: 'github',       // use GitHub releases as changelog source
	// 			base: 'changelog',        // base path for changelog pages
	// 			owner: 'githubnext',      // GitHub org/user
	// 			repo: 'gh-aw',            // GitHub repo
	// 			// Use GitHub token if available in environment, otherwise rely on public API
	// 			...(process.env.GITHUB_TOKEN && { token: process.env.GITHUB_TOKEN }),
	// 			// No process filter: include all releases
	// 		},
	// 	]),
	// }),
};
