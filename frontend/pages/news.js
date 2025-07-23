import React from 'react';
import Layout from '../components/layout/Layout';
import Link from 'next/link';
import { getCategories } from '../lib/api';

/**
 * News page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function News({ categories = [] }) {
  // Sample news items - in a real implementation, these would come from an API
  const newsItems = [
    {
      id: 1,
      title: 'Welcome to the New CreateMod.com',
      date: '2025-07-20',
      summary: 'We\'re excited to announce the launch of our new website built with Next.js!',
      content: 'Today marks the official launch of our completely redesigned CreateMod.com website. The new site is built using Next.js, providing a faster, more responsive experience for all users. We\'ve improved navigation, search functionality, and added many new features to help you discover and share Create mod schematics more easily.'
    },
    {
      id: 2,
      title: 'New Schematic Upload Features',
      date: '2025-07-15',
      summary: 'We\'ve added new features to make uploading and sharing your schematics easier than ever.',
      content: 'Our latest update includes several improvements to the schematic upload process. You can now add multiple images to showcase your creations, include YouTube videos for tutorials, and specify dependencies more clearly. We\'ve also improved the categorization system to help others find your schematics more easily.'
    },
    {
      id: 3,
      title: 'Community Spotlight: July 2025',
      date: '2025-07-10',
      summary: 'Check out this month\'s featured community creations and contributors.',
      content: 'Each month, we highlight exceptional contributions from our community members. This month\'s spotlight features amazing railway systems, factory designs, and automated farms. We\'re also recognizing users who have been particularly helpful in the comments section, providing support and suggestions to fellow creators.'
    }
  ];

  // Format date for display
  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    }).format(date);
  };

  return (
    <Layout 
      title="News - CreateMod.com"
      description="Latest news, updates, and announcements for the CreateMod.com community"
      categories={categories}
    >
      <div className="container-xl py-4">
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">News & Announcements</h2>
          </div>
          <div className="card-body">
            <p className="text-muted mb-4">
              Stay up to date with the latest developments, features, and community highlights from CreateMod.com.
            </p>
            
            {newsItems.length > 0 ? (
              <div className="divide-y">
                {newsItems.map((item) => (
                  <div key={item.id} className="py-4">
                    <h3 className="mb-1">{item.title}</h3>
                    <div className="text-muted mb-2">{formatDate(item.date)}</div>
                    <p className="mb-2">{item.content}</p>
                    {/* In a real implementation, this would link to a full news article */}
                    <Link href={`/news/${item.id}`} className="btn btn-sm btn-outline-primary">
                      Read More
                    </Link>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty">
                <div className="empty-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-news" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11"></path>
                    <path d="M8 8l4 0"></path>
                    <path d="M8 12l4 0"></path>
                    <path d="M8 16l4 0"></path>
                  </svg>
                </div>
                <p className="empty-title">No news yet</p>
                <p className="empty-subtitle text-muted">
                  Check back soon for news and announcements about CreateMod.com!
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </Layout>
  );
}

/**
 * Server-side data fetching
 * 
 * @param {Object} context - Next.js context
 * @returns {Object} - Props for the page component
 */
export async function getServerSideProps(context) {
  try {
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    return {
      props: {
        categories
      }
    };
  } catch (error) {
    console.error('Error fetching data for news page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: []
      }
    };
  }
}