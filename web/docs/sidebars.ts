import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/installation',
        'getting-started/quick-start',
        'getting-started/authentication',
      ],
    },
    {
      type: 'category',
      label: 'Channels',
      items: [
        'channels/overview',
        'channels/whatsapp',
        'channels/telegram',
        'channels/sms',
        'channels/email',
        'channels/voice',
        'channels/webchat',
        'channels/facebook',
        'channels/instagram',
        'channels/rcs',
      ],
    },
    {
      type: 'category',
      label: 'AI & Bots',
      items: [
        'bots/overview',
        'bots/configuration',
        'bots/escalation',
        'bots/testing',
      ],
    },
    {
      type: 'category',
      label: 'Flows',
      items: [
        'flows/overview',
        'flows/node-types',
        'flows/triggers',
      ],
    },
    {
      type: 'category',
      label: 'Knowledge Base',
      items: [
        'knowledge-base/overview',
        'knowledge-base/embeddings',
      ],
    },
    {
      type: 'category',
      label: 'SDKs',
      items: [
        'sdks/overview',
        'sdks/typescript',
        'sdks/python',
        'sdks/go',
        'sdks/java',
        'sdks/rust',
        'sdks/dotnet',
        'sdks/php',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/overview',
        'api/authentication',
        'api/conversations',
        'api/messages',
        'api/channels',
        'api/bots',
        'api/webhooks',
      ],
    },
    {
      type: 'category',
      label: 'Self-Hosting',
      items: [
        'self-hosting/docker',
        'self-hosting/kubernetes',
      ],
    },
  ],
};

export default sidebars;
