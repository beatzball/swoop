export const siteConfig = {
  title: 'swoop',
  description:
    'fzf finds. libghostty draws. Any program can be an extension. An open-source keyboard launcher for a Mac or Linux, one key away.',
  logo: null,
  editUrlBase: 'https://github.com/beatzball/swoop/edit/main/site/content/docs',
  nav: [
    { label: 'Docs', href: '/docs/getting-started' },
    { label: 'Extensions', href: '/docs/writing-an-extension' },
    { label: 'Blog', href: '/blog' },
    { label: 'GitHub', href: 'https://github.com/beatzball/swoop' },
  ],
  sidebar: [
    {
      label: 'Start Here',
      items: [
        { label: 'Getting Started', slug: 'getting-started' },
        { label: 'Coming from a GUI Launcher', slug: 'coming-from-a-gui-launcher' },
      ],
    },
    {
      label: 'Using swoop',
      items: [
        { label: 'Ask AI',     slug: 'ask-ai' },
        { label: 'Extensions', slug: 'extensions' },
        { label: 'Settings',   slug: 'settings' },
      ],
    },
    {
      label: 'Build',
      items: [
        { label: 'Writing an Extension', slug: 'writing-an-extension' },
        { label: 'The Extension Contract', slug: 'contract' },
        { label: 'Build Your Own Tool', slug: 'build-your-own' },
        { label: 'Agent Skill', slug: 'agent-skill' },
      ],
    },
  ],
};

export default siteConfig;
