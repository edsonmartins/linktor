import fs from 'node:fs';
import path from 'node:path';
import type {LoadContext, Plugin} from '@docusaurus/types';

/**
 * Documentação em texto puro para uso com LLMs, gerada no build a partir das
 * mesmas páginas-fonte do site — nunca fica para trás da documentação:
 *
 *   /llms.txt       índice: uma linha por página, com link e resumo (llmstxt.org)
 *   /llms-full.txt  todas as páginas, em Markdown, num arquivo só
 *
 * Sai um par por idioma (o do espanhol em /es/). Página sem tradução entra em
 * português, como no site.
 */

/** Ordem de leitura; as pastas que não estiverem aqui entram depois, em ordem alfabética. */
const SECTION_ORDER = ['guias', 'conceitos', 'painel', 'canais', 'api', 'mcp', 'sdks', 'implantar', 'contribuir'];

interface Page {
  /** Caminho relativo à pasta de docs, com barras normais: canais/sms.md */
  rel: string;
  title: string;
  description: string;
  position: number;
  url: string;
  body: string;
}

function walk(dir: string, base = dir): string[] {
  return fs.readdirSync(dir, {withFileTypes: true}).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full, base);
    return /\.mdx?$/.test(entry.name) ? [path.relative(base, full).split(path.sep).join('/')] : [];
  });
}

function parseFrontMatter(source: string): {data: Record<string, string>; body: string} {
  const match = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/.exec(source);
  if (!match) return {data: {}, body: source};
  const data: Record<string, string> = {};
  for (const line of match[1].split(/\r?\n/)) {
    const pair = /^([A-Za-z_]+):\s*(.*)$/.exec(line);
    if (pair) data[pair[1]] = pair[2].replace(/^(['"])(.*)\1$/, '$2');
  }
  return {data, body: source.slice(match[0].length)};
}

/** Rota de uma página: o `slug` do front matter, ou o caminho sem a extensão. */
function routeOf(rel: string, slug?: string): string {
  if (slug) return slug.startsWith('/') ? slug : `/${slug}`;
  return `/${rel.replace(/\.mdx?$/, '')}`;
}

/** Aplica `transform` só fora dos blocos de código, que precisam sair intactos. */
function outsideCodeFences(body: string, transform: (text: string) => string): string {
  return body
    .split(/(^```[\s\S]*?^```[^\n]*$)/m)
    .map((part) => (part.startsWith('```') ? part : transform(part)))
    .join('');
}

export default function llmsTxtPlugin(context: LoadContext): Plugin {
  const {siteDir, siteConfig, i18n, baseUrl} = context;
  const sourceDir = path.join(siteDir, 'docs');
  const translatedDir = path.join(siteDir, 'i18n', i18n.currentLocale, 'docusaurus-plugin-content-docs', 'current');
  const siteRoot = siteConfig.url.replace(/\/$/, '') + baseUrl.replace(/\/$/, '');

  /** A tradução da página, quando existe; senão a fonte em pt-BR. */
  const read = (rel: string) => {
    const translated = path.join(translatedDir, rel);
    return fs.readFileSync(fs.existsSync(translated) ? translated : path.join(sourceDir, rel), 'utf8');
  };

  return {
    name: 'linktor-llms-txt',

    async postBuild({outDir}) {
      const files = walk(sourceDir);
      const routes = new Map(files.map((rel) => [rel, routeOf(rel, parseFrontMatter(read(rel)).data.slug)]));

      const pages: Page[] = files.map((rel) => {
        const {data, body} = parseFrontMatter(read(rel));
        const cleaned = outsideCodeFences(body, (text) =>
          text
            // MDX: imports e componentes não significam nada fora do site
            .replace(/^import .*$/gm, '')
            .replace(/^<[A-Z][^>]*\/>\s*$/gm, '')
            // links relativos entre páginas viram URLs completas
            .replace(/\]\((?!https?:|#|\/|mailto:|pathname:)([^)\s#]+\.mdx?)(#[^)\s]*)?\)/g, (whole, target, hash = '') => {
              const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(rel), target));
              const route = routes.get(resolved);
              return route ? `](${siteRoot}${route}${hash})` : whole;
            }),
        );
        return {
          rel,
          title: data.title || rel,
          description: data.description || '',
          position: Number(data.sidebar_position ?? 999),
          url: siteRoot + routes.get(rel),
          body: cleaned.replace(/\n{3,}/g, '\n\n').trim(),
        };
      });

      const sectionOf = (page: Page) => (page.rel.includes('/') ? page.rel.split('/')[0] : '');
      const rank = (section: string) => {
        if (section === '') return -1;
        const index = SECTION_ORDER.indexOf(section);
        return index === -1 ? SECTION_ORDER.length : index;
      };
      pages.sort(
        (a, b) =>
          rank(sectionOf(a)) - rank(sectionOf(b)) ||
          sectionOf(a).localeCompare(sectionOf(b)) ||
          a.position - b.position ||
          a.rel.localeCompare(b.rel),
      );

      const sectionLabel = (section: string) => {
        const category = path.join(sourceDir, section, '_category_.json');
        if (section && fs.existsSync(category)) return JSON.parse(fs.readFileSync(category, 'utf8')).label as string;
        return section || siteConfig.title;
      };

      const index: string[] = [`# ${siteConfig.title}`, '', `> ${siteConfig.tagline}`, ''];
      const full: string[] = [
        `# ${siteConfig.title} — documentação completa`,
        '',
        `> ${siteConfig.tagline}`,
        '',
        `Fonte: ${siteRoot}/ · ${pages.length} páginas · índice em ${siteRoot}/llms.txt`,
        '',
      ];

      let current: string | null = null;
      for (const page of pages) {
        const section = sectionOf(page);
        if (section !== current) {
          current = section;
          index.push(`## ${sectionLabel(section)}`, '');
        }
        index.push(`- [${page.title}](${page.url})${page.description ? `: ${page.description}` : ''}`);
        full.push('---', '', `Página: ${page.title}`, `URL: ${page.url}`, '', page.body, '');
      }
      index.push('', '## Opcional', '', `- [Documentação completa em um arquivo](${siteRoot}/llms-full.txt)`, '');

      fs.writeFileSync(path.join(outDir, 'llms.txt'), index.join('\n').replace(/\n(## )/g, '\n\n$1').replace(/\n{3,}/g, '\n\n'));
      fs.writeFileSync(path.join(outDir, 'llms-full.txt'), full.join('\n'));
    },
  };
}
