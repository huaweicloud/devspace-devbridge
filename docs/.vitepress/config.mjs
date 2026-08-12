import { defineConfig } from "vitepress";

const base = "/devspace-devbridge/";

const sidebar = [
  {
    text: "开始使用",
    items: [
      { text: "创建并托管隧道", link: "/" },
      { text: "什么是开发隧道", link: "/guide/overview" },
      { text: "安装 DevBridge CLI", link: "/guide/install" },
      { text: "登录与凭证", link: "/guide/authentication" },
    ],
  },
  {
    text: "使用隧道",
    items: [
      { text: "管理隧道", link: "/guide/tunnels" },
      { text: "管理端口", link: "/guide/ports" },
      { text: "Host：托管本地服务", link: "/guide/host" },
      { text: "Connect：连接远程服务", link: "/guide/connect" },
      { text: "配额查询与调试工具", link: "/guide/quota-and-debug" },
    ],
  },
  {
    text: "最佳实践",
    items: [
      { text: "托管与公网访问", link: "/guide/best-practices/host-public-access" },
      { text: "托管与远程连接", link: "/guide/best-practices/host-remote-connect" },
    ],
  },
  {
    text: "参考",
    items: [
      { text: "CLI 命令参考", link: "/reference/cli" },
      { text: "AI Agent Skill", link: "/reference/skill" },
      { text: "REST API", link: "/reference/api" },
      { text: "本地配置与目录", link: "/reference/configuration" },
      { text: "问题排查", link: "/reference/troubleshooting" },
      { text: "更新日志", link: "/changelog" },
    ],
  },
];

export default defineConfig({
  lang: "zh-CN",
  title: "DevBridge",
  titleTemplate: ":title | DevBridge",
  description: "使用 DevBridge 安全地托管和访问本地开发服务。",
  base,
  cleanUrls: true,
  lastUpdated: true,
  sitemap: {
    hostname: "https://huaweicloud.github.io/devspace-devbridge/",
    transformItems: (items) =>
      items.filter((item) => item.url !== "404" && item.url !== "/404"),
  },
  head: [
    ["meta", { name: "theme-color", content: "#ffffff" }],
    ["meta", { name: "color-scheme", content: "light dark" }],
  ],
  themeConfig: {
    siteTitle: "DevBridge",
    nav: [
      { text: "开发隧道", link: "/" },
      { text: "命令参考", link: "/reference/cli" },
      { text: "REST API", link: "/reference/api" },
      { text: "更新日志", link: "/changelog" },
    ],
    sidebar,
    outline: {
      label: "本页内容",
      level: [2, 3],
    },
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
    docFooter: {
      prev: "上一篇",
      next: "下一篇",
    },
    lastUpdated: {
      text: "最后更新于",
      formatOptions: {
        dateStyle: "medium",
        timeStyle: "short",
      },
    },
    darkModeSwitchLabel: "外观",
    lightModeSwitchTitle: "切换到浅色模式",
    darkModeSwitchTitle: "切换到深色模式",
    sidebarMenuLabel: "目录",
    returnToTopLabel: "返回顶部",
    langMenuLabel: "语言",
    externalLinkIcon: true,
  },
});
