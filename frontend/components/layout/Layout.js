import React from 'react';
import Head from 'next/head';
import Sidebar from './Sidebar';
import Header from './Header';
import Footer from './Footer';

/**
 * Main layout component for the application
 * 
 * @param {Object} props - Component props
 * @param {React.ReactNode} props.children - Page content
 * @param {string} props.title - Page title
 * @param {string} props.description - Page description
 * @param {string} props.thumbnail - Thumbnail image URL for social sharing
 * @param {string} props.slug - Page slug for canonical URL
 * @param {string} props.subCategory - Page subcategory for header
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether user is authenticated
 * @param {Object} props.user - User data if authenticated
 */
export default function Layout({ 
  children, 
  title = 'CreateMod.com', 
  description = 'Share and discover schematics for the Create mod',
  thumbnail = '',
  slug = '',
  subCategory = '',
  categories = [],
  isAuthenticated = false,
  user = null
}) {
  // Construct canonical URL
  const canonicalUrl = slug ? `https://createmod.com/${slug}` : 'https://createmod.com';
  
  return (
    <div className="page">
      <Head>
        <title>{title}</title>
        <meta name="description" content={description} />
        <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
        <link rel="shortcut icon" href="/favicon-192x192.png" type="image/x-icon" />
        
        {/* Canonical URL */}
        <link rel="canonical" href={canonicalUrl} />
        
        {/* Open Graph / Social Media Meta Tags */}
        <meta property="og:title" content={title} />
        <meta property="og:description" content={description} />
        <meta property="og:type" content="website" />
        <meta property="og:url" content={canonicalUrl} />
        {thumbnail && <meta property="og:image" content={thumbnail} />}
        
        {/* Twitter Card Meta Tags */}
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content={title} />
        <meta name="twitter:description" content={description} />
        {thumbnail && <meta name="twitter:image" content={thumbnail} />}
      </Head>
      
      {/* Sidebar */}
      <Sidebar 
        categories={categories}
        isAuthenticated={isAuthenticated}
        user={user}
      />
      
      <div className="page-wrapper">
        {/* Header */}
        <Header 
          title={title}
          subCategory={subCategory}
          isAuthenticated={isAuthenticated}
          user={user}
        />
        
        {/* Main content */}
        <div className="page-body">
          <div className="container-xl">
            {children}
          </div>
        </div>
        
        {/* Footer */}
        <Footer />
      </div>
    </div>
  );
}