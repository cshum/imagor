import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "imagor docs",
  tagline: "Fast, secure libvips-based image processing server and Go library",
  favicon: "img/icon.png",

  // Set the production url of your site here
  url: "https://docs.imagor.net",
  // Set the /<baseUrl>/ pathname under which your site is served
  baseUrl: "/",

  // GitHub pages deployment config.
  organizationName: "cshum",
  projectName: "imagor",

  onBrokenLinks: "throw",

  // Enable SWC for faster builds
  future: {
    experimental_faster: {
      swcJsLoader: true,
      swcJsMinimizer: true,
      swcHtmlMinimizer: true,
    },
  },

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: "./sidebars.ts",
          editUrl: "https://github.com/cshum/imagor/tree/master/docs/",
          routeBasePath: "/",
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themes: [
    "@docusaurus/theme-mermaid",
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      {
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: "/",
      },
    ],
  ],

  themeConfig: {
    image: "img/icon.png",
    metadata: [
      {
        property: "og:site_name",
        content: "imagor docs",
      },
      {
        name: "twitter:card",
        content: "summary",
      },
    ],
    navbar: {
      title: "imagor",
      items: [
        {
          type: "dropdown",
          label: "Ecosystem",
          position: "left",
          items: [
            {
              label: "Imagor Cloud",
              href: "https://imagor.net",
            },
            {
              label: "imagor",
              to: "/",
            },
            {
              label: "imagorvideo",
              to: "/imagorvideo",
            },
            {
              label: "imagorface",
              to: "/imagorface",
            },
          ],
        },
        {
          type: "search",
          position: "right",
        },
        {
          href: "https://github.com/cshum/imagor",
          label: "GitHub",
          position: "right",
        },
        {
          href: "https://imagor.net",
          label: "Imagor Cloud",
          position: "right",
          className: "navbar-buy-button",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Docs",
          items: [
            {
              label: "Getting Started",
              to: "/",
            },
            {
              label: "Benchmarks",
              to: "/benchmarks",
            },
            {
              label: "Image Endpoint",
              to: "/image-endpoint",
            },
            {
              label: "Configuration",
              to: "/configuration",
            },
          ],
        },
        {
          title: "Ecosystem",
          items: [
            {
              label: "Imagor Cloud",
              href: "https://imagor.net",
            },
            {
              label: "imagor",
              to: "/",
            },
            {
              label: "imagorvideo",
              to: "/imagorvideo",
            },
            {
              label: "imagorface",
              to: "/imagorface",
            },
          ],
        },
        {
          title: "More",
          items: [
            {
              label: "GitHub",
              href: "https://github.com/cshum/imagor",
            },
            {
              label: "Docker Hub",
              href: "https://hub.docker.com/r/shumc/imagor",
            },
            {
              label: "Benchmarks",
              to: "/benchmarks",
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} imagor.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["bash", "yaml", "docker", "go", "javascript", "php"],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
