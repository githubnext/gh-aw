import type { APIRoute } from 'astro';

export const prerender = true;

export const GET: APIRoute = () => {
	const lines = [
		'# GitHub Agentic Workflows',
		'',
		'> Documentation index for GitHub Agentic Workflows (gh-aw).',
		'> Canonical docs: https://github.github.com/gh-aw/',
		'',
		'## Documentation',
		'',
		'- [Home](https://github.github.com/gh-aw/)',
		'- [Quick Start](https://github.github.com/gh-aw/setup/quick-start/)',
		'- [CLI Commands](https://github.github.com/gh-aw/setup/cli/)',
		'- [Guides](https://github.github.com/gh-aw/guides/agentic-authoring/)',
		'- [Reference](https://github.github.com/gh-aw/reference/)',
		'- [Patterns](https://github.github.com/gh-aw/patterns/daily-ops/)',
		'- [Examples](https://github.github.com/gh-aw/examples/maintaining-repos/)',
		'- [Sitemap](https://github.github.com/gh-aw/sitemap-index.xml)',
		'',
		'## AI Discovery',
		'',
		'- [robots.txt](https://github.github.com/gh-aw/robots.txt)',
		'- [ai.txt](https://github.github.com/gh-aw/.well-known/ai.txt)',
		'- [agents.txt](https://github.github.com/gh-aw/agents.txt)',
		'- [llms-full.txt](https://github.github.com/gh-aw/llms-full.txt)',
	];

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
