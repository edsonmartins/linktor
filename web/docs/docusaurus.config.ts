import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Linktor Documentation',
  tagline: 'Omnichannel conversations powered by AI',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://docs.linktor.io',
  baseUrl: '/',

  organizationName: 'linktor',
  projectName: 'linktor',

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl: 'https://github.com/linktor/linktor/tree/main/web/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/linktor-social-card.jpg',
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: false,
    },
    navbar: {
      title: '',
      logo: {
        alt: 'Linktor Logo',
        src: 'img/logo_fundo_claro.png',
        srcDark: 'img/logo_fundo_escuro.png',
        width: 120,
        height: 32,
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          to: '/sdks/overview',
          label: 'SDKs',
          position: 'left',
        },
        {
          to: '/api/overview',
          label: 'API',
          position: 'left',
        },
        {
          href: 'https://linktor.io',
          label: 'Home',
          position: 'right',
        },
        {
          href: 'https://app.linktor.io',
          label: 'Dashboard',
          position: 'right',
        },
        {
          href: 'https://github.com/linktor/linktor',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/getting-started/installation',
            },
            {
              label: 'Channels',
              to: '/channels/overview',
            },
            {
              label: 'SDKs',
              to: '/sdks/overview',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/linktor/linktor',
            },
            {
              label: 'Discord',
              href: 'https://discord.gg/linktor',
            },
            {
              label: 'Twitter',
              href: 'https://twitter.com/linktorhq',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'Landing Page',
              href: 'https://linktor.io',
            },
            {
              label: 'Dashboard',
              href: 'https://app.linktor.io',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Linktor. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'typescript', 'python', 'go', 'java', 'rust', 'csharp', 'php'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
