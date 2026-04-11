import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    {
      type: "doc",
      id: "intro",
      label: "Getting Started",
    },
    {
      type: "doc",
      id: "image-endpoint",
      label: "Image Endpoint",
    },
    {
      type: "doc",
      id: "filters",
      label: "Filters",
    },
    {
      type: "category",
      label: "Storage",
      collapsed: false,
      items: [
        "storage",
        "storage-filesystem",
        "storage-s3",
        "storage-gcloud",
        "loader-http",
      ],
    },
    {
      type: "doc",
      id: "security",
      label: "Security",
    },
    {
      type: "category",
      label: "Advanced",
      items: [
        "metadata-and-exif",
        "color-image",
        "in-memory-cache",
        "vips-performance",
        "post-upload",
      ],
    },
    {
      type: "category",
      label: "imagor Variants",
      items: ["mozjpeg-support", "imagemagick-support"],
    },
    {
      type: "category",
      label: "Plugins",
      collapsed: false,
      items: ["imagorvideo", "imagorface"],
    },
    {
      type: "doc",
      id: "configuration",
      label: "Configuration",
    },
    {
      type: "doc",
      id: "community",
      label: "Community",
    },
  ],
};

export default sidebars;
