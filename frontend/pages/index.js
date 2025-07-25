import React from 'react';
import Layout from '../components/layout/Layout';
import Link from 'next/link';
import SchematicCard from '../components/schematics/SchematicCard';
import { getCategories, getRecords } from '../lib/api';

/**
 * Home page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {Array} props.schematics - Schematics data
 * @param {number} props.totalItems - Total number of schematics
 */
export default function Home({ categories = [], schematics = [], totalItems = 0 }) {
  return (
    <Layout 
      title="CreateMod.com"
      description="Share and discover schematics for the Create mod"
      categories={categories}
    >
      {/* Welcome section */}
      <div className="row mb-4">
        <div className="col-12">
          <div className="card">
            <div className="card-body">
              <h1 className="card-title">Welcome to CreateMod.com</h1>
              <p className="text-muted">
                CreateMod.com is a community-driven platform for sharing and discovering schematics for the Create mod in Minecraft.
                Browse through thousands of schematics, upload your own creations, and connect with other Create mod enthusiasts.
              </p>
              <div className="mt-4">
                <Link href="/upload" className="btn btn-primary me-2">
                  Upload Schematic
                </Link>
                <Link href="/search" className="btn btn-secondary">
                  Advanced Search
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Featured Schematics */}
      <div className="row mb-4">
        <div className="col-12">
          <div className="card">
            <div className="card-header">
              <h2 className="card-title">Featured Schematics</h2>
            </div>
            <div className="card-body">
              {schematics.length > 0 ? (
                <div className="row row-cards">
                  {schematics.map((schematic) => (
                    <div className="col-sm-6 col-lg-4" key={schematic.id}>
                      <SchematicCard schematic={schematic} />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="empty">
                  <div className="empty-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-mood-sad" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                      <path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0"></path>
                      <path d="M9 10l.01 0"></path>
                      <path d="M15 10l.01 0"></path>
                      <path d="M9.5 15.25a3.5 3.5 0 0 1 5 0"></path>
                    </svg>
                  </div>
                  <p className="empty-title">No schematics found</p>
                  <p className="empty-subtitle text-muted">
                    There are no schematics available at the moment. Be the first to upload one!
                  </p>
                  <div className="empty-action">
                    <Link href="/upload" className="btn btn-primary">
                      Upload Schematic
                    </Link>
                  </div>
                </div>
              )}
              
              {schematics.length > 0 && (
                <div className="mt-4 text-center">
                  <Link href="/search" className="btn btn-primary">
                    Browse All Schematics ({totalItems})
                  </Link>
                </div>
              )}
            </div>
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
    
    // Fetch featured or recent schematics
    // We'll sort by most recent and limit to 6 items
    const schematicsData = await getRecords('schematics', {
      sort: '-created', // Sort by newest first
      filter: 'moderated=true', // Only show moderated schematics
      expand: 'author,categories,tags', // Include related data
      page: 1,
      perPage: 6 // Show 6 schematics on the main page
    });
    
    const schematics = schematicsData.items || [];
    const totalItems = schematicsData.totalItems || 0;
    
    return {
      props: {
        categories,
        schematics,
        totalItems
      }
    };
  } catch (error) {
    console.error('Error fetching data for home page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        schematics: [],
        totalItems: 0
      }
    };
  }
}