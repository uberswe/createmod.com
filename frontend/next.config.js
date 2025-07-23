/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Server-Side Rendering is enabled by default in Next.js
  
  // Configure image remote patterns for Next.js Image component
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'createmod.com',
        pathname: '/**',
      },
      {
        protocol: 'http',
        hostname: 'localhost',
        port: '8090',
        pathname: '/api/**',
      },
    ],
  },
  // Configure async rewrites to maintain the same URL structure
  async rewrites() {
    return [
      // Rewrite API requests to the backend
      {
        source: '/api/:path*',
        destination: 'http://localhost:8090/api/:path*',
      },
    ];
  },
};

module.exports = nextConfig;