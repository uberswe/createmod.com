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
        
        {/* Bootstrap JavaScript */}
        <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js" integrity="sha384-C6RzsynM9kWDrMNeT87bh95OGNyZPhcTNXj1NW7RuBCsyN/o0jlpcV8Qyq46cDfL" crossOrigin="anonymous"></script>
      </body>
    </Html>
  );
}