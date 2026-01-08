<script setup lang="ts">
definePageMeta({
  layout: 'docs',
})

useSeoMeta({
  title: 'Quick Start - Lurus AI Documentation',
  description: 'Get started with Lurus AI in 5 minutes.',
})
</script>

<template>
  <div>
    <h1>Quick Start</h1>

    <p class="lead">
      Get up and running with Lurus AI in under 5 minutes.
    </p>

    <h2>1. Get Your API Key</h2>

    <p>
      First, sign up for a free account and get your API key:
    </p>

    <ol>
      <li>Go to <a href="/login">ai.lurus.cn/login</a></li>
      <li>Create an account or sign in</li>
      <li>Navigate to <strong>Settings → API Keys</strong></li>
      <li>Click <strong>Generate New Key</strong></li>
    </ol>

    <div class="not-prose my-6 p-4 rounded-xl bg-surface-800 border border-surface-700">
      <p class="text-yellow-400 text-sm font-medium mb-2">Important</p>
      <p class="text-surface-300 text-sm">
        Keep your API key secure. Never commit it to version control or share it publicly.
      </p>
    </div>

    <h2>2. Make Your First Request</h2>

    <p>
      The Lurus Gateway API is compatible with the OpenAI API format. Here's a simple example:
    </p>

    <pre><code class="language-bash">curl https://gateway.lurus.cn/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'</code></pre>

    <h2>3. Use with OpenAI SDK</h2>

    <p>
      Since Lurus is OpenAI-compatible, you can use the official OpenAI SDK:
    </p>

    <h3>Python</h3>

    <pre><code class="language-python">from openai import OpenAI

client = OpenAI(
    api_key="YOUR_LURUS_API_KEY",
    base_url="https://gateway.lurus.cn/v1"
)

response = client.chat.completions.create(
    model="claude-3-5-sonnet",  # Use any supported model
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)</code></pre>

    <h3>JavaScript/TypeScript</h3>

    <pre><code class="language-typescript">import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'YOUR_LURUS_API_KEY',
  baseURL: 'https://gateway.lurus.cn/v1',
});

const response = await client.chat.completions.create({
  model: 'claude-3-5-sonnet',
  messages: [
    { role: 'user', content: 'Hello!' }
  ],
});

console.log(response.choices[0].message.content);</code></pre>

    <h2>4. Configure Claude Code</h2>

    <p>
      To use Lurus with Claude Code CLI:
    </p>

    <ol>
      <li>Set the environment variable:
        <pre><code class="language-bash">export ANTHROPIC_BASE_URL=https://gateway.lurus.cn
export ANTHROPIC_API_KEY=YOUR_LURUS_API_KEY</code></pre>
      </li>
      <li>Run Claude Code as normal:
        <pre><code class="language-bash">claude</code></pre>
      </li>
    </ol>

    <h2>5. Available Models</h2>

    <p>
      Lurus supports 40+ AI providers. Here are some popular models:
    </p>

    <table>
      <thead>
        <tr>
          <th>Model ID</th>
          <th>Provider</th>
          <th>Description</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td><code>gpt-4</code></td>
          <td>OpenAI</td>
          <td>GPT-4 latest</td>
        </tr>
        <tr>
          <td><code>gpt-4-turbo</code></td>
          <td>OpenAI</td>
          <td>GPT-4 Turbo with vision</td>
        </tr>
        <tr>
          <td><code>claude-3-5-sonnet</code></td>
          <td>Anthropic</td>
          <td>Claude 3.5 Sonnet</td>
        </tr>
        <tr>
          <td><code>claude-3-opus</code></td>
          <td>Anthropic</td>
          <td>Claude 3 Opus</td>
        </tr>
        <tr>
          <td><code>gemini-1.5-pro</code></td>
          <td>Google</td>
          <td>Gemini 1.5 Pro</td>
        </tr>
        <tr>
          <td><code>deepseek-chat</code></td>
          <td>DeepSeek</td>
          <td>DeepSeek Chat</td>
        </tr>
      </tbody>
    </table>

    <h2>Next Steps</h2>

    <ul>
      <li><a href="/docs/api/chat">API Reference</a> - Full API documentation</li>
      <li><a href="/docs/codeswitch">CodeSwitch Setup</a> - Desktop app configuration</li>
      <li><a href="/docs/guides/routing">Model Routing</a> - Configure failover and load balancing</li>
    </ul>
  </div>
</template>
