import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "imagor",
  tagline: "Fast, secure image processing server and Go library",
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

  themes: ["@docusaurus/theme-mermaid"],

  themeConfig: {
    navbar: {
      title: "imagor",
      items: [
        {
          type: "docSidebar",
          sidebarId: "tutorialSidebar",
          position: "left",
          label: "Documentation",
        },
        {
          href: "https://docs.studio.imagor.net",
          label: "Imagor Studio",
          position: "right",
        },
        {
          href: "https://github.com/cshum/imagor",
          label: "GitHub",
          position: "right",
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
              label: "Configuration",
              to: "/configuration/overview",
            },
            {
              label: "API Reference",
              to: "/api/image-endpoint",
            },
          ],
        },
        {
          title: "Ecosystem",
          items: [
            {
              label: "imagor-studio",
              href: "https://github.com/cshum/imagor-studio",
            },
            {
              label: "vipsgen",
              href: "https://github.com/cshum/vipsgen",
            },
            {
              label: "imagorvideo",
              href: "https://github.com/cshum/imagorvideo",
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
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Adrian Shum.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["bash", "yaml", "docker", "go", "javascript"],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
