import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'obtain',
  description: 'Fetch the remote files you keep needing into the current directory.',
  // '/' locally; CI sets DOCS_BASE=/obtain/ for GitHub Pages project sites.
  base: process.env.DOCS_BASE ?? '/',
  head: [['link', { rel: 'icon', type: 'image/svg+xml', href: `${process.env.DOCS_BASE ?? '/'}logo.svg` }]],
  cleanUrls: true,
  // Serve guide/ at the site root; introduction is the landing page.
  rewrites: {
    'guide/introduction.md': 'index.md',
    'guide/:page.md': ':page.md',
  },
  lastUpdated: true,
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/' },
      { text: 'Cookbook', link: '/cookbook' },
    ],
    sidebar: [
      {
        text: 'Basics',
        items: [
          { text: 'Introduction', link: '/' },
          { text: 'Installation', link: '/installation' },
          { text: 'Getting started', link: '/getting-started' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Configuration', link: '/config' },
          { text: 'Variables', link: '/variables' },
          { text: 'Commands', link: '/commands' },
        ],
      },
      {
        text: 'Examples',
        items: [{ text: 'Cookbook', link: '/cookbook' }],
      },
    ],
    socialLinks: [{ icon: 'github', link: 'https://github.com/jorqensen/obtain' }],
    search: { provider: 'local' },
    editLink: {
      pattern: 'https://github.com/jorqensen/obtain/edit/main/docs/:path',
    },
  },
})
