export const siteConfig = {
  title: 'swoop',
  description:
    'fzf finds. libghostty draws. Every extension is a program that prints lines. An open-source keyboard launcher for a Mac or Linux, one key away.',
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
