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
          href: "https://imagor.net",
          label: "imagor.net",
          position: "left",
        },
        {
          type: "dropdown",
          label: "Ecosystem",
          position: "left",
          items: [
            {
              label: "imagor Studio",
              href: "https://imagor.net",
            },
            {
              label: "imagorvideo",
              href: "https://github.com/cshum/imagorvideo",
            },
            {
              label: "imagorface",
              href: "https://github.com/cshum/imagorface",
            },
          ],
        },
        {
          href: "https://github.com/cshum/imagor",
          label: "GitHub",
          position: "right",
        },
        {
          href: "https://github.com/sponsors/cshum",
          label: "Sponsor",
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
              to: "/configuration",
            },
            {
              label: "Community",
              to: "/community",
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
              label: "imagorvideo",
              href: "https://github.com/cshum/imagorvideo",
            },
            {
              label: "imagorface",
              href: "https://github.com/cshum/imagorface",
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
