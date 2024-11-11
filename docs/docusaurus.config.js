// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");
const math = require("remark-math");
const katex = require("rehype-katex");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Injective Documentation",
  tagline: "Welcome to Injective - the blockchain built for finance!",
  url: "https://docs.helios.network/",
  baseUrl: "/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "throw",
  favicon: "img/favicon.png",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "Injective", // Usually your GitHub org/user name.
  projectName: "injective-core/docs", // Usually your repo name.

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  plugins: [],

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve("./sidebars.js"),
          routeBasePath: "/", // Serve the docs at the site's root
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          //editUrl:
          //  "https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/",
          remarkPlugins: [math],
          rehypePlugins: [katex],
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      }),
    ],
  ],

  stylesheets: [
    {
      href: "https://cdn.jsdelivr.net/npm/katex@0.13.24/dist/katex.min.css",
      type: "text/css",
      integrity:
        "sha384-odtC+0UGzzFL/6PNoE8rX/SPcQDXBJ+uRepguP4QkPCm2LBxH3FA3y+fKSiJ+AmM",
      crossorigin: "anonymous",
    },
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      prism: {
        additionalLanguages: ["rust"],
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
      image: "/img/inj_meta.png",
      docs: {
        sidebar: {
          autoCollapseCategories: true,
        },
      },
      navbar: {
        title: "Injective",
        logo: {
          alt: "Injective Docs",
          src: "img/helios.svg",
        },
        items: [
          {
            to: '/learn',
            label: 'Learn',
            position: 'left',
          },
          {
            to: '/develop',
            label: 'Develop',
            position: 'left',
          },
          {
            to: '/trade',
            label: 'Trade',
            position: 'left',
          },
          {
            to: '/nodes',
            label: 'Nodes & Validators',
            position: 'left',
          },
          {
            to: '/resources',
            label: 'Resources',
            position: 'left',
          },
          {
            href: "https://helios.com/",
            label: "Injective",
            position: "right",
          },
          {
            href: "https://github.com/InjectiveLabs",
            label: "GitHub",
            position: "right",
          },
          {
            type: "search",
            position: "right",
          },
        ],
      },
      footer: {
        style: "light",
        logo: {
          href: "https://helios.network",
          target: "_blank",
          srcDark: "img/injective_logo.svg",
          src: "img/injective_logo_dark.svg",
          height: "36px",
          style: { textAlign: "left" },
          alt: "Injective Logo",
        },
        links: [
          {
            title: "Injective",
            items: [
              {
                label: "Hub",
                href: "https://hub.helios.network/",
              },
              {
                label: "Explorer",
                href: "https://explorer.helios.network/",
              },
              {
                label: "Blog",
                href: "https://blog.helios.com/",
              },
            ],
          },
          {
            title: "Community",
            items: [
              {
                label: "Blog",
                href: "https://blog.helios.com/",
              },
              {
                label: "Injective Forum",
                href: "https://gov.helios.network/",
              },
              {
                label: "Discord",
                href: "https://discord.gg/injective",
              },
              {
                label: "Reddit",
                href: "https://www.reddit.com/r/injective/",
              },
            ],
          },
          {
            title: "Social",
            items: [
              {
                label: "Twitter",
                href: "https://twitter.com/Injective_",
              },
              {
                label: "Youtube",
                href: "https://www.youtube.com/channel/UCN99m0dicoMjNmJV9mxioqQ",
              },
              {
                label: "LinkedIn",
                href: "https://www.linkedin.com/company/injective-protocol",
              },
              {
                label: "Medium",
                href: "https://injectiveprotocol.medium.com/",
              },
            ],
          },
        ],
        copyright: `Copyright © Injective since 2021. All rights reserved <a href="https://helios.com/">Injective</a>`,
      },
      algolia: {
        appId: "OSH0IFX0OC",
        apiKey: "b8c3930ea2d1ed063992787837d3567f",
        indexName: "injective",
        contextualSearch: true,
        searchParameters: {},
      },
    }),
};

module.exports = config;
