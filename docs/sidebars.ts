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
      link: {
        type: "doc",
        id: "storage",
      },
      items: [
        "storage-filesystem",
        "storage-s3",
        "storage-gcloud",
        "loader-http",
        {
          type: "doc",
          id: "storage-path-style",
          label: "Path Style",
        },
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
      link: {
        type: "generated-index",
        description:
          "Advanced topics covering metadata, in-memory caching, color profiles, performance tuning, file upload, and build variants.",
      },
      items: [
        "metadata-and-exif",
        "in-memory-cache",
        "color-image",
        "vips-performance",
        "post-upload",
        "mozjpeg-support",
        "imagemagick-support",
      ],
    },
    {
      type: "category",
      label: "Plugins",
      collapsed: false,
      link: {
        type: "generated-index",
        description:
          "Extend imagor with additional capabilities through plugins.",
      },
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
