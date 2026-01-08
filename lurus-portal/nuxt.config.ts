// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-01-07',
  devtools: { enabled: true },

  // Modules
  modules: [
    '@nuxtjs/tailwindcss',
    '@vueuse/motion/nuxt',
  ],

  // App configuration
  app: {
    head: {
      title: 'Lurus AI - Intelligent AI Infrastructure',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Lurus AI - Your intelligent AI infrastructure platform. Unified gateway for 40+ AI providers.' },
        { name: 'theme-color', content: '#0f172a' },
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap' },
      ],
    },
    pageTransition: { name: 'page', mode: 'out-in' },
  },

  // Tailwind CSS
  tailwindcss: {
    cssPath: '~/assets/css/tailwind.css',
    configPath: 'tailwind.config.ts',
  },

  // SSG for static deployment
  ssr: true,
  nitro: {
    prerender: {
      routes: [
        '/',
        '/products',
        '/products/codeswitch',
        '/products/gateway',
        '/pricing',
        '/login',
        '/docs',
        '/docs/quickstart',
      ],
      crawlLinks: false,
      failOnError: false,
    },
  },

  // TypeScript
  typescript: {
    strict: true,
  },
})
