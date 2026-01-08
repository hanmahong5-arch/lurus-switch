<script setup lang="ts">
definePageMeta({
  layout: 'docs',
})

useSeoMeta({
  title: 'Documentation - Lurus AI',
  description: 'Learn how to use Lurus AI products with our comprehensive documentation.',
})
</script>

<template>
  <div>
    <h1>Welcome to Lurus AI Documentation</h1>

    <p class="lead">
      Learn how to integrate and use Lurus AI products to build powerful AI applications.
    </p>

    <h2>Quick Links</h2>

    <div class="grid md:grid-cols-2 gap-4 not-prose">
      <NuxtLink
        to="/docs/quickstart"
        class="glass-card group"
      >
        <h3 class="text-lg font-semibold text-white mb-2 group-hover:text-primary-400 transition-colors">
          Quick Start
        </h3>
        <p class="text-surface-400 text-sm">
          Get up and running in 5 minutes with our step-by-step guide.
        </p>
      </NuxtLink>

      <NuxtLink
        to="/docs/api/chat"
        class="glass-card group"
      >
        <h3 class="text-lg font-semibold text-white mb-2 group-hover:text-primary-400 transition-colors">
          API Reference
        </h3>
        <p class="text-surface-400 text-sm">
          Complete API documentation with examples and code snippets.
        </p>
      </NuxtLink>

      <NuxtLink
        to="/docs/codeswitch"
        class="glass-card group"
      >
        <h3 class="text-lg font-semibold text-white mb-2 group-hover:text-primary-400 transition-colors">
          CodeSwitch Guide
        </h3>
        <p class="text-surface-400 text-sm">
          Learn how to set up and use the CodeSwitch desktop application.
        </p>
      </NuxtLink>

      <NuxtLink
        to="/docs/gateway"
        class="glass-card group"
      >
        <h3 class="text-lg font-semibold text-white mb-2 group-hover:text-primary-400 transition-colors">
          Gateway Setup
        </h3>
        <p class="text-surface-400 text-sm">
          Configure the Gateway API for your applications.
        </p>
      </NuxtLink>
    </div>

    <h2>Overview</h2>

    <p>
      Lurus AI provides a unified interface for working with multiple AI providers.
      Whether you're building a chatbot, content generation tool, or any AI-powered
      application, our platform makes it easy to:
    </p>

    <ul>
      <li><strong>Switch providers</strong> - Change between OpenAI, Anthropic, Google, and 40+ other providers with a single config change</li>
      <li><strong>Manage costs</strong> - Track token usage, set budgets, and optimize spending</li>
      <li><strong>Ensure reliability</strong> - Automatic failover and load balancing across providers</li>
      <li><strong>Stay secure</strong> - Enterprise-grade security with SOC 2 compliance</li>
    </ul>

    <h2>Products</h2>

    <h3>CodeSwitch</h3>
    <p>
      A desktop application for developers who want full control over their AI provider
      configuration. Features real-time analytics, request logging, and seamless integration
      with CLI tools like Claude Code and Codex.
    </p>

    <h3>Gateway</h3>
    <p>
      A cloud-based API gateway that provides an OpenAI-compatible interface to 40+ AI providers.
      Perfect for teams and organizations who need a centralized, managed solution.
    </p>

    <h2>Getting Help</h2>

    <p>
      If you have questions or need assistance:
    </p>

    <ul>
      <li>Check our <a href="/docs/quickstart">Quick Start Guide</a></li>
      <li>Browse the <a href="/docs/api/chat">API Reference</a></li>
      <li>Join our <a href="https://discord.gg/lurus">Discord community</a></li>
      <li>Contact <a href="mailto:support@lurus.cn">support@lurus.cn</a></li>
    </ul>
  </div>
</template>
