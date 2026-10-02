export const siteConfig = {
  title: 'swoop',
  description:
    'An open-source keyboard launcher built the Unix way: fzf does the finding, libghostty does the drawing, and every extension is a program that prints lines.',
  logo: null,
  editUrlBase: 'https://github.com/beatzball/swoop/edit/main/site/content/docs',
  nav: [
    { label: 'Docs', href: '/docs/getting-started' },
    { label: 'Extensions', href: '/docs/writing-an-extension' },
    { label: 'GitHub', href: 'https://github.com/beatzball/swoop' },
  ],
  sidebar: [
    {
      label: 'Start Here',
      items: [
        { label: 'Getting Started', slug: 'getting-started' },
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
      ],
    },
  ],
};

export default siteConfig;
