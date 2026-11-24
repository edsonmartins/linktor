import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Linktor',
  tagline: 'Conversas omnichannel com IA — uma API e um painel para todos os canais',
  favicon: 'img/favicon.png',

  future: {
    v4: true,
  },

  // DOCS_URL vem do build da imagem (web/docs/Dockerfile): o endereço em que o
  // site é publicado.
  url: process.env.DOCS_URL || 'https://docs.linktor.dev',
  baseUrl: '/',

  organizationName: 'edsonmartins',
  projectName: 'linktor',

  // Link ou âncora quebrados falham o build — a navegação é parte do contrato da documentação.
  onBrokenLinks: 'throw',
  onBrokenAnchors: 'throw',
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  i18n: {
    // pt-BR é a fonte (web/docs/docs). Espanhol fica em i18n/es; página ainda
    // não traduzida aparece em português, em vez de sumir.
    defaultLocale: 'pt-BR',
    locales: ['pt-BR', 'es'],
    localeConfigs: {
      'pt-BR': {label: 'Português', htmlLang: 'pt-BR'},
      es: {label: 'Español', htmlLang: 'es'},
    },
  },

  themes: [
    '@docusaurus/theme-mermaid',
    [
      // Busca local (índice gerado no build; não depende de serviço externo).
      '@easyops-cn/docusaurus-search-local',
      {
        hashed: true,
        language: ['pt', 'es', 'en'],
        docsRouteBasePath: '/',
        indexBlog: false,
        highlightSearchTermsOnTargetPage: true,
        searchResultLimits: 10,
      },
    ],
  ],

  // The Scalar plugin is temporarily disabled until an OpenAPI spec is published
  // under static/openapi.json or another stable URL.
  plugins: [
    // [
    //   '@scalar/docusaurus',
    //   {
    //     label: 'API Reference',
    //     route: '/api-reference',
    //     configuration: {
    //       spec: {
    //         url: '/openapi.json',
    //       },
    //       theme: 'purple',
    //       hideModels: false,
    //       hideDownloadButton: false,
    //       hiddenClients: [],
    //       defaultHttpClient: {
    //         targetKey: 'javascript',
    //         clientKey: 'fetch',
    //       },
    //     },
    //   },
    // ],
  ],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl: 'https://github.com/edsonmartins/linktor/tree/main/web/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: false,
    },
    navbar: {
      // Símbolo + nome em texto: um arquivo só serve ao tema claro e ao escuro.
      title: 'Linktor',
      logo: {
        alt: 'Linktor',
        src: 'img/logo.png',
        width: 32,
        height: 32,
      },
      items: [
        // Navegação por público: quem atende, quem conecta canais, quem integra, quem opera.
        {type: 'docSidebar', sidebarId: 'usar', position: 'left', label: 'Usar'},
        {type: 'docSidebar', sidebarId: 'canais', position: 'left', label: 'Canais'},
        {type: 'docSidebar', sidebarId: 'desenvolver', position: 'left', label: 'Desenvolver'},
        {type: 'docSidebar', sidebarId: 'sdks', position: 'left', label: 'SDKs'},
        {type: 'docSidebar', sidebarId: 'implantar', position: 'left', label: 'Implantar'},
        {type: 'localeDropdown', position: 'right'},
        {
          href: 'https://github.com/edsonmartins/linktor',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Usar',
          items: [
            {label: 'Primeiros passos', to: '/guias/primeiros-passos'},
            {label: 'Conceitos', to: '/conceitos/visao-geral'},
            {label: 'Canais', to: '/canais/visao-geral'},
          ],
        },
        {
          title: 'Desenvolver',
          items: [
            {label: 'API', to: '/api/visao-geral'},
            {label: 'SDKs', to: '/sdks/visao-geral'},
            {label: 'MCP', to: '/mcp/visao-geral'},
          ],
        },
        {
          title: 'Implantar',
          items: [
            {label: 'Instalação', to: '/implantar/instalacao'},
            {label: 'GitHub', href: 'https://github.com/edsonmartins/linktor'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Linktor`,
    },
    mermaid: {
      theme: {light: 'neutral', dark: 'dark'},
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'typescript', 'python', 'go', 'java', 'rust', 'csharp', 'php', 'yaml', 'toml', 'nginx', 'docker', 'http', 'diff'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
