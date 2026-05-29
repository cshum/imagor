import {JSX, PropsWithChildren} from 'react';
import Head from '@docusaurus/Head';
import {useLocation} from '@docusaurus/router';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

export default function Root({ children }: PropsWithChildren): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  const { pathname } = useLocation();
  const normalizedPath = pathname !== '/' && pathname.endsWith('/')
    ? pathname.slice(0, -1)
    : pathname;

  const routeMetadata: Record<
    string,
    {title: string; description: string}
  > = {
    '/image-endpoint': {
      title: 'imagor image endpoint | URL syntax, resize, crop, filters | imagor docs',
      description:
        'imagor URL syntax reference for resize, crop, fit-in, smart crop, padding, trim, and image source paths.',
    },
    '/configuration': {
      title: 'imagor configuration | flags and environment variables | imagor docs',
      description:
        'Complete imagor configuration reference covering command-line flags, environment variables, loaders, storage, security, processing, and deployment settings.',
    },
    '/filters': {
      title: 'imagor filters | watermark, format conversion, text overlay | imagor docs',
      description:
        'Full imagor filter reference for format conversion, quality, watermarking, text overlays, color operations, metadata, and output control.',
    },
    '/imagorface': {
      title: 'imagorface | face detection, smart crop, redaction | imagor docs',
      description:
        'Extends imagor with face detection for face-centred smart crop, privacy redaction, and detected region metadata.',
    },
    '/imagorvideo': {
      title: 'imagorvideo | video thumbnails through the imagor pipeline | imagor docs',
      description:
        'Extends imagor with video thumbnail extraction so video frames can be processed through the full imagor pipeline.',
    },
    '/in-memory-cache': {
      title: 'imagor in-memory cache | decoded image pixel cache | imagor docs',
      description:
        'Cache decoded image pixels in memory with LRU eviction to avoid repeated I/O and decoding of the same source image across requests.',
    },
    '/loader-http': {
      title: 'imagor HTTP Loader | allowed sources, proxy, SSRF | imagor docs',
      description:
        'Configure imagor HTTP Loader with allowed sources, base URL, proxying, header forwarding, SSRF protection, and size limits.',
    },
    '/metadata-and-exif': {
      title: 'imagor metadata and Exif | /meta, BlurHash, ThumbHash | imagor docs',
      description:
        'Extract image format, resolution and Exif metadata via the /meta endpoint, and compute BlurHash, ThumbHash and average color.',
    },
    '/storage': {
      title: 'imagor storage | loaders, source storage, result storage | imagor docs',
      description:
        'Configure imagor loaders, source storage, and result storage for HTTP, file system, AWS S3, and Google Cloud Storage.',
    },
    '/post-upload': {
      title: 'imagor POST upload | direct image processing via HTTP POST | imagor docs',
      description:
        'Process and transform images via HTTP POST upload, an opt-in imagor feature intended for trusted internal environments.',
    },
    '/storage-gcloud': {
      title:
        'imagor Google Cloud Storage | loader, storage, result storage | imagor docs',
      description:
        'Configure imagor with Google Cloud Storage for loader, source storage, result storage, key normalization, and bucket path layout.',
    },
    '/storage-s3': {
      title: 'imagor S3 | AWS S3 loader, storage, result storage | imagor docs',
      description:
        'Configure imagor with AWS S3 or S3-compatible storage for loader, source storage, result storage, routing, and key normalization.',
    },
    '/vips-performance': {
      title: 'imagor VIPS performance tuning | concurrency and threading | imagor docs',
      description:
        'Tune libvips concurrency and threading to optimize imagor image processing performance for your deployment.',
    },
  };
  const pageMetadata = routeMetadata[normalizedPath];

  const organizationSchema = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: 'imagor',
    url: siteConfig.url,
    logo: `${siteConfig.url}/img/icon.png`,
    sameAs: [
      'https://github.com/cshum/imagor',
      'https://imagor.net',
      'https://hub.docker.com/r/shumc/imagor',
    ],
  };

  const websiteSchema = {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: siteConfig.title,
    url: siteConfig.url,
    potentialAction: {
      '@type': 'SearchAction',
      target: `${siteConfig.url}/search?q={search_term_string}`,
      'query-input': 'required name=search_term_string',
    },
    publisher: {
      '@type': 'Organization',
      name: 'imagor',
    },
  };

  return (
    <>
      <Head>
        <meta property="og:type" content="website" />
        {pageMetadata && <title>{pageMetadata.title}</title>}
        {pageMetadata && (
          <meta property="og:title" content={pageMetadata.title} />
        )}
        {pageMetadata && (
          <meta name="twitter:title" content={pageMetadata.title} />
        )}
        {pageMetadata && (
          <meta property="og:description" content={pageMetadata.description} />
        )}
        {pageMetadata && (
          <meta name="twitter:description" content={pageMetadata.description} />
        )}
        <script type="application/ld+json">
          {JSON.stringify(organizationSchema)}
        </script>
        <script type="application/ld+json">
          {JSON.stringify(websiteSchema)}
        </script>
      </Head>
      {children}
    </>
  );
}
