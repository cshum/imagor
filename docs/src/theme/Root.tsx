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
    '/configuration': {
      title: 'imagor configuration | flags and environment variables | imagor docs',
      description:
        'Complete imagor configuration reference covering command-line flags, environment variables, loaders, storage, security, processing, and deployment settings.',
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
