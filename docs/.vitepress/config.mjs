import { defineConfig } from "vitepress";

const base = "/devspace-devbridge/";

const enSidebar = [
  {
    text: "Getting started",
    items: [
      { text: "What is DevBridge", link: "/guide/overview" },
      { text: "Install the DevBridge CLI", link: "/guide/install" },
      { text: "Login and credentials", link: "/guide/authentication" },
      { text: "Create and host a tunnel", link: "/guide/quickstart" },
    ],
  },
  {
    text: "Working with tunnels",
    items: [
      { text: "Manage tunnels", link: "/guide/tunnels" },
      { text: "Manage ports", link: "/guide/ports" },
      { text: "Host: expose local services", link: "/guide/host" },
      { text: "Connect: connect to remote services", link: "/guide/connect" },
      { text: "Quota query and debug tools", link: "/guide/quota-and-debug" },
    ],
  },
  {
    text: "Best practices",
    items: [
      {
        text: "Hosting and public access",
        link: "/guide/best-practices/host-public-access",
      },
      {
        text: "Hosting and remote connection",
        link: "/guide/best-practices/host-remote-connect",
      },
    ],
  },
  {
    text: "Integrations",
    items: [{ text: "AI Agent Skill", link: "/integrations/skill" }],
  },
  {
    text: "Reference",
    items: [
      { text: "CLI command reference", link: "/reference/cli" },
      { text: "REST API", link: "/reference/api" },
      { text: "Troubleshooting", link: "/reference/troubleshooting" },
      { text: "Changelog", link: "/changelog" },
      { text: "Disclaimer", link: "/reference/disclaimer" },
      { text: "Contributing", link: "/reference/contributing" },
    ],
  },
];

const zhSidebar = [
  {
    text: "开始使用",
    items: [
      { text: "什么是开发隧道", link: "/zh/guide/overview" },
      { text: "安装 DevBridge CLI", link: "/zh/guide/install" },
      { text: "登录与凭证", link: "/zh/guide/authentication" },
      { text: "创建并托管隧道", link: "/zh/guide/quickstart" },
    ],
  },
  {
    text: "使用隧道",
    items: [
      { text: "管理隧道", link: "/zh/guide/tunnels" },
      { text: "管理端口", link: "/zh/guide/ports" },
      { text: "Host：托管本地服务", link: "/zh/guide/host" },
      { text: "Connect：连接远程服务", link: "/zh/guide/connect" },
      { text: "配额查询与调试工具", link: "/zh/guide/quota-and-debug" },
    ],
  },
  {
    text: "最佳实践",
    items: [
      {
        text: "托管与公网访问",
        link: "/zh/guide/best-practices/host-public-access",
      },
      {
        text: "托管与远程连接",
        link: "/zh/guide/best-practices/host-remote-connect",
      },
    ],
  },
  {
    text: "集成",
    items: [{ text: "AI Agent Skill", link: "/zh/integrations/skill" }],
  },
  {
    text: "参考",
    items: [
      { text: "CLI 命令参考", link: "/zh/reference/cli" },
      { text: "REST API", link: "/zh/reference/api" },
      { text: "问题排查", link: "/zh/reference/troubleshooting" },
      { text: "更新日志", link: "/zh/changelog" },
      { text: "免责声明", link: "/zh/reference/disclaimer" },
      { text: "贡献指南", link: "/zh/reference/contributing" },
    ],
  },
];

export default defineConfig({
  lang: "en",
  title: "DevBridge",
  titleTemplate: ":title | DevBridge",
  base,
  cleanUrls: true,
  srcExclude: ["README.md", "README.zh-CN.md"],
  lastUpdated: true,
  locales: {
    root: {
      label: "English",
      lang: "en",
      title: "DevBridge",
      titleTemplate: ":title | DevBridge",
      description:
        "Securely host and access local development services with DevBridge.",
      themeConfig: {
        siteTitle: "DevBridge",
        nav: [
          { text: "DevBridge", link: "/" },
          { text: "CLI reference", link: "/reference/cli" },
          { text: "REST API", link: "/reference/api" },
          { text: "Changelog", link: "/changelog" },
        ],
        sidebar: enSidebar,
        outline: { label: "On this page", level: [2, 3] },
        search: { provider: "local" },
        docFooter: { prev: "Previous", next: "Next" },
        lastUpdated: {
          text: "Last updated",
          formatOptions: { dateStyle: "medium", timeStyle: "short" },
        },
        darkModeSwitchLabel: "Appearance",
        lightModeSwitchTitle: "Switch to light mode",
        darkModeSwitchTitle: "Switch to dark mode",
        sidebarMenuLabel: "Menu",
        returnToTopLabel: "Back to top",
        langMenuLabel: "Language",
        externalLinkIcon: true,
      },
    },
    zh: {
      label: "中文",
      lang: "zh-CN",
      link: "/zh/",
      title: "DevBridge",
      titleTemplate: ":title | DevBridge",
      description: "使用 DevBridge 安全地托管和访问本地开发服务。",
      themeConfig: {
        siteTitle: "DevBridge",
        nav: [
          { text: "开发隧道", link: "/zh/" },
          { text: "命令参考", link: "/zh/reference/cli" },
          { text: "REST API", link: "/zh/reference/api" },
          { text: "更新日志", link: "/zh/changelog" },
        ],
        sidebar: zhSidebar,
        outline: { label: "本页内容", level: [2, 3] },
        search: {
          provider: "local",
          options: {
            translations: {
              button: {
                buttonText: "搜索",
                buttonAriaLabel: "搜索文档",
              },
              modal: {
                noResultsText: "没有找到相关内容",
                resetButtonTitle: "清除查询",
                footer: {
                  selectText: "选择",
                  navigateText: "切换",
                  closeText: "关闭",
                },
              },
            },
          },
        },
        docFooter: { prev: "上一篇", next: "下一篇" },
        lastUpdated: {
          text: "最后更新于",
          formatOptions: { dateStyle: "medium", timeStyle: "short" },
        },
        darkModeSwitchLabel: "外观",
        lightModeSwitchTitle: "切换到浅色模式",
        darkModeSwitchTitle: "切换到深色模式",
        sidebarMenuLabel: "目录",
        returnToTopLabel: "返回顶部",
        langMenuLabel: "语言",
        externalLinkIcon: true,
      },
    },
  },
  sitemap: {
    hostname: "https://hhuchang.github.io/devspace-devbridge/",
    transformItems: (items) =>
      items.filter((item) => item.url !== "404" && item.url !== "/404"),
  },
  head: [
    ["meta", { name: "theme-color", content: "#ffffff" }],
    ["meta", { name: "color-scheme", content: "light dark" }],
  ],
  themeConfig: {
    search: { provider: "local" },
  },
});
