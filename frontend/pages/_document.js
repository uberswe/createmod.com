import { Html, Head, Main, NextScript } from 'next/document';

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        {/* Main stylesheet */}
        <link rel="stylesheet" href="/assets/style-XHxDiORf.css" />
        
        {/* Additional stylesheets */}
        <link rel="stylesheet" href="/libs/star-rating/dist/star-rating.min.css" />
        <link rel="stylesheet" href="/libs/plyr/dist/plyr.css" />
        
        {/* Favicon */}
        <link rel="shortcut icon" href="/favicon-192x192.png" type="image/x-icon" />
      </Head>
      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  );
}