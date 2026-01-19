<script setup lang="ts">
import { Icon } from "@iconify/vue";

interface FooterLink {
    label: string;
    href: string;
    external?: boolean;
}

interface FooterSection {
    title: string;
    titleEn: string;
    links: FooterLink[];
}

const footerLinks: Record<string, FooterSection> = {
    products: {
        title: "产品",
        titleEn: "Products",
        links: [
            {
                label: "API Gateway",
                href: "https://api.lurus.cn",
                external: true,
            },
            { label: "CodeSwitch", href: "/download", external: false },
            { label: "企业方案", href: "/enterprise", external: false },
            {
                label: "控制台",
                href: "https://api.lurus.cn/console",
                external: true,
            },
        ],
    },
    resources: {
        title: "资源",
        titleEn: "Resources",
        links: [
            {
                label: "API 文档",
                href: "https://docs.lurus.cn",
                external: true,
            },
            {
                label: "系统状态",
                href: "https://api.lurus.cn/status",
                external: true,
            },
            { label: "更新日志", href: "/changelog", external: false },
            {
                label: "帮助中心",
                href: "https://docs.lurus.cn/help",
                external: true,
            },
        ],
    },
    company: {
        title: "公司",
        titleEn: "Company",
        links: [
            { label: "关于我们", href: "/about", external: false },
            { label: "博客", href: "/blog", external: false },
            { label: "招聘", href: "/careers", external: false },
            {
                label: "联系我们",
                href: "mailto:xiaohan@lurus.cn",
                external: false,
            },
        ],
    },
    legal: {
        title: "法律",
        titleEn: "Legal",
        links: [
            { label: "服务条款", href: "/terms", external: false },
            { label: "隐私政策", href: "/privacy", external: false },
            { label: "使用政策", href: "/acceptable-use", external: false },
        ],
    },
};

const socialLinks = [
    {
        icon: "mdi:github",
        href: "https://github.com/lurus-ai",
        label: "GitHub",
    },
    {
        icon: "mdi:twitter",
        href: "https://twitter.com/lurus_ai",
        label: "Twitter",
    },
    { icon: "mdi:email", href: "mailto:xiaohan@lurus.cn", label: "Email" },
];
</script>

<template>
    <footer class="relative border-t border-surface-800/50">
        <!-- Background -->
        <div class="absolute inset-0 bg-surface-950"></div>
        <div class="absolute inset-0 bg-grid opacity-30"></div>

        <div class="section-container relative">
            <!-- Main footer content -->
            <div class="py-16">
                <div
                    class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-8 lg:gap-12"
                >
                    <!-- Brand column -->
                    <div
                        class="col-span-2 md:col-span-4 lg:col-span-1 mb-8 lg:mb-0"
                    >
                        <router-link
                            to="/"
                            class="flex items-center gap-2.5 mb-4"
                        >
                            <div
                                class="w-9 h-9 flex items-center justify-center rounded-xl bg-gradient-to-br from-primary-500 to-accent-500"
                            >
                                <Icon
                                    icon="mdi:robot-outline"
                                    class="w-5 h-5 text-white"
                                />
                            </div>
                            <span class="text-xl font-bold text-white"
                                >LURUS</span
                            >
                        </router-link>
                        <p class="text-surface-400 text-sm mb-6 max-w-xs">
                            统一智能基础设施平台，让 AI 集成变得简单。
                        </p>

                        <!-- Social links -->
                        <div class="flex items-center gap-3">
                            <a
                                v-for="social in socialLinks"
                                :key="social.label"
                                :href="social.href"
                                target="_blank"
                                :aria-label="social.label"
                                class="w-9 h-9 flex items-center justify-center rounded-lg bg-surface-800/50 text-surface-400 hover:bg-surface-700/50 hover:text-white transition-colors"
                            >
                                <Icon :icon="social.icon" class="w-5 h-5" />
                            </a>
                        </div>
                    </div>

                    <!-- Link columns -->
                    <div v-for="(section, key) in footerLinks" :key="key">
                        <h4 class="text-white font-semibold mb-4">
                            {{ section.title }}
                            <span
                                class="text-surface-600 text-xs font-normal ml-1"
                                >{{ section.titleEn }}</span
                            >
                        </h4>
                        <ul class="space-y-3">
                            <li v-for="link in section.links" :key="link.label">
                                <a
                                    :href="link.href"
                                    :target="
                                        link.external ? '_blank' : undefined
                                    "
                                    class="text-surface-400 text-sm hover:text-white transition-colors inline-flex items-center gap-1"
                                >
                                    {{ link.label }}
                                    <Icon
                                        v-if="link.external"
                                        icon="mdi:open-in-new"
                                        class="w-3 h-3 opacity-50"
                                    />
                                </a>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>

            <!-- Divider -->
            <div class="divider-sketch"></div>

            <!-- Bottom bar -->
            <div
                class="py-6 flex flex-col md:flex-row items-center justify-between gap-4"
            >
                <!-- Copyright & ICP Beian -->
                <div class="flex flex-col md:flex-row items-center gap-3 text-surface-500 text-sm">
                    <div>
                        &copy; 2024-{{ new Date().getFullYear() }} Lurus AI. All
                        rights reserved.
                    </div>
                    <div class="hidden md:block text-surface-600">|</div>
                    <a
                        href="https://beian.miit.gov.cn/"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="flex items-center gap-1.5 hover:text-white transition-colors"
                    >
                        <Icon icon="mdi:shield-check-outline" class="w-4 h-4" />
                        <span>鲁ICP备2026000242号-1</span>
                    </a>
                </div>

                <div class="flex items-center gap-6 text-surface-500 text-sm">
                    <!-- Status indicator -->
                    <a
                        href="https://api.lurus.cn/status"
                        target="_blank"
                        class="flex items-center gap-2 hover:text-white transition-colors"
                    >
                        <span
                            class="w-2 h-2 bg-success-500 rounded-full animate-pulse"
                        ></span>
                        系统正常
                    </a>

                    <!-- Language selector (placeholder) -->
                    <div class="flex items-center gap-1.5">
                        <Icon icon="mdi:translate" class="w-4 h-4" />
                        <span>简体中文</span>
                    </div>
                </div>
            </div>
        </div>
    </footer>
</template>
