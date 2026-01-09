// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-01-07',
  devtools: { enabled: true },

  // Modules
  modules: [
    '@nuxtjs/tailwindcss',
    '@vueuse/motion/nuxt',
    '@nuxtjs/supabase',
  ],

  // Supabase configuration
  supabase: {
    redirect: false, // We handle redirects manually
    redirectOptions: {
      login: '/auth/login',
      callback: '/auth/callback',
      exclude: ['/', '/pricing', '/products/*', '/docs/*', '/auth/*'],
    },
  },

  // Runtime configuration
  runtimeConfig: {
    // Server-side only
    billingServiceUrl: process.env.BILLING_SERVICE_URL || 'http://localhost:18103',
    subscriptionServiceUrl: process.env.SUBSCRIPTION_SERVICE_URL || 'http://localhost:18104',
    newApiUrl: process.env.NEW_API_URL || 'http://localhost:3000',
    // Public (client-side)
    public: {
      siteUrl: process.env.SITE_URL || 'http://localhost:3001',
    },
  },

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

  // SSR mode for dynamic content
  ssr: true,
  nitro: {
    prerender: {
      routes: [
        '/',
        '/products',
        '/products/codeswitch',
        '/products/gateway',
        '/pricing',
        '/docs',
        '/docs/quickstart',
        '/auth/login',
        '/auth/register',
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
