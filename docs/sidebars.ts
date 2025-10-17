import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    "intro",
    {
      type: "category",
      label: "Getting Started",
      items: ["getting-started/quick-start"],
    },
    {
      type: "category",
      label: "API Reference",
      items: ["api/image-endpoint"],
    },
  ],
};

export default sidebars;
