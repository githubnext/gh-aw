import type { APIRoute } from 'astro';
import fs from 'node:fs/promises';
import path from 'node:path';

export const GET: APIRoute = async () => {
  // The dictation instructions file is in the repository root's .github/instructions/ directory
  const file = path.resolve(process.cwd(), '../.github/instructions/dictation.instructions.md');
  
  try {
    const content = await fs.readFile(file, 'utf8');
    return new Response(content, {
      status: 200,
      headers: { 
        'Content-Type': 'text/plain; charset=utf-8',
        'Cache-Control': 'public, max-age=3600'
      },
    });
  } catch (e) {
    console.error('Error reading dictation instructions:', e);
    return new Response('Not found', { status: 404 });
  }
};
