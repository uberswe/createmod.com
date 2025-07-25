import React from 'react';
import Layout from '../components/layout/Layout';
import Link from 'next/link';
import SchematicCard from '../components/schematics/SchematicCard';
import { getCategories, getRecords } from '../lib/api';

/**
 * Explore page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {Array} props.featuredSchematics - Featured schematics data
 * @param {Array} props.featuredCreators - Featured creators data
 */
export default function Explore({ categories = [], featuredSchematics = [], featuredCreators = [] }) {
  return (
    <Layout 
      title="Explore - CreateMod.com"
      description="Explore featured schematics, creators, and collections on CreateMod.com"
      categories={categories}
    >
      <div className="container-xl py-4">
        {/* Hero section */}
        <div className="card mb-4">
          <div className="card-body">
            <div className="row align-items-center">
              <div className="col-md-8">
                <h2 className="card-title">Explore Create Mod Schematics</h2>
                <p className="text-muted">
                  Discover amazing creations from our community. From intricate machinery to beautiful decorations, 
                  find inspiration for your next Minecraft project.
                </p>
                <div className="mt-3">
                  <Link href="/search" className="btn btn-primary me-2">
                    Browse All Schematics
                  </Link>
                  <Link href="/upload" className="btn btn-outline-secondary">
                    Share Your Creation
                  </Link>
                </div>
              </div>
              <div className="col-md-4 d-none d-md-block text-end">
                <svg xmlns="http://www.w3.org/2000/svg" width="120" height="120" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-library-photo text-primary">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <path d="M7 3m0 2.667a2.667 2.667 0 0 1 2.667 -2.667h8.666a2.667 2.667 0 0 1 2.667 2.667v8.666a2.667 2.667 0 0 1 -2.667 2.667h-8.666a2.667 2.667 0 0 1 -2.667 -2.667z" />
                  <path d="M4.012 7.26a2.005 2.005 0 0 0 -1.012 1.737v10c0 1.1 .9 2 2 2h10c.75 0 1.158 -.385 1.5 -1" />
                  <path d="M17 7h.01" />
                  <path d="M7 13l3.644 -3.644a1.21 1.21 0 0 1 1.712 0l3.644 3.644" />
                  <path d="M15 12l1.644 -1.644a1.21 1.21 0 0 1 1.712 0l2.644 2.644" />
                </svg>
              </div>
            </div>
          </div>
        </div>
        
        {/* Featured Schematics */}
        <div className="card mb-4">
          <div className="card-header">
            <div className="d-flex justify-content-between align-items-center">
              <h3 className="card-title">Featured Schematics</h3>
              <Link href="/search?sort=4" className="btn btn-sm btn-outline-primary">
                View More
              </Link>
            </div>
          </div>
          <div className="card-body">
            {featuredSchematics.length > 0 ? (
              <div className="row row-cards">
                {featuredSchematics.map((schematic) => (
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
                <p className="empty-title">No featured schematics yet</p>
                <p className="empty-subtitle text-muted">
                  Check back soon for featured schematics or browse all schematics.
                </p>
                <div className="empty-action">
                  <Link href="/search" className="btn btn-primary">
                    Browse All Schematics
                  </Link>
                </div>
              </div>
            )}
          </div>
        </div>
        
        {/* Featured Creators */}
        <div className="card mb-4">
          <div className="card-header">
            <h3 className="card-title">Featured Creators</h3>
          </div>
          <div className="card-body">
            {featuredCreators.length > 0 ? (
              <div className="row row-cards">
                {featuredCreators.map((creator) => (
                  <div className="col-md-6 col-lg-3" key={creator.id}>
                    <div className="card">
                      <div className="card-body p-4 text-center">
                        {creator.avatar ? (
                          <span 
                            className="avatar avatar-xl mb-3 avatar-rounded" 
                            style={{ backgroundImage: `url(${creator.avatar})` }}
                          ></span>
                        ) : (
                          <span className="avatar avatar-xl mb-3 avatar-rounded">
                            {creator.username.charAt(0).toUpperCase()}
                          </span>
                        )}
                        <h3 className="m-0 mb-1">{creator.name || creator.username}</h3>
                        <div className="text-muted">{creator.schematicsCount} schematics</div>
                        <div className="mt-3">
                          <Link href={`/author/${creator.username.toLowerCase()}`} className="btn btn-primary w-100">
                            View Profile
                          </Link>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty">
                <div className="empty-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-users" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"></path>
                    <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                    <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                    <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
                  </svg>
                </div>
                <p className="empty-title">No featured creators yet</p>
                <p className="empty-subtitle text-muted">
                  Check back soon for featured creators or browse all schematics.
                </p>
              </div>
            )}
          </div>
        </div>
        
        {/* Categories Showcase */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Browse by Category</h3>
          </div>
          <div className="card-body">
            <div className="row g-3">
              {categories.length > 0 ? (
                categories.map((category) => (
                  <div className="col-sm-6 col-md-4 col-lg-3" key={category.id}>
                    <Link 
                      href={`/search?category=${category.key}&sort=6`}
                      className="card card-link card-sm"
                    >
                      <div className="card-body">
                        <div className="row align-items-center">
                          <div className="col-auto">
                            <span className="bg-primary text-white avatar">
                              {/* Display first letter of category name */}
                              {category.name.charAt(0)}
                            </span>
                          </div>
                          <div className="col">
                            <div className="font-weight-medium">{category.name}</div>
                            <div className="text-muted">{category.count || 0} schematics</div>
                          </div>
                        </div>
                      </div>
                    </Link>
                  </div>
                ))
              ) : (
                <div className="col-12">
                  <div className="empty">
                    <p className="empty-title">No categories available</p>
                    <p className="empty-subtitle text-muted">
                      Check back soon for categories or browse all schematics.
                    </p>
                    <div className="empty-action">
                      <Link href="/search" className="btn btn-primary">
                        Browse All Schematics
                      </Link>
                    </div>
                  </div>
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
    // Get categories for sidebar and showcase
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Add count property to categories (in a real implementation, this would come from the API)
    const categoriesWithCount = categories.map(category => ({
      ...category,
      count: Math.floor(Math.random() * 100) // Placeholder random count
    }));
    
    // Get featured schematics (in a real implementation, this would use a filter for featured items)
    let featuredSchematics = [];
    try {
      const schematicsData = await getRecords('schematics', {
        sort: '-rating', // Sort by highest rating
        filter: 'moderated=true',
        expand: 'author,categories,tags',
        page: 1,
        perPage: 6
      });
      
      featuredSchematics = schematicsData.items || [];
    } catch (error) {
      console.error('Error fetching featured schematics:', error);
    }
    
    // Sample featured creators (in a real implementation, this would come from the API)
    const featuredCreators = [
      {
        id: '1',
        username: 'RedstoneWizard',
        name: 'Redstone Wizard',
        avatar: null,
        schematicsCount: 42
      },
      {
        id: '2',
        username: 'MechanicalMaster',
        name: 'Mechanical Master',
        avatar: null,
        schematicsCount: 37
      },
      {
        id: '3',
        username: 'CreateEngineer',
        name: 'Create Engineer',
        avatar: null,
        schematicsCount: 28
      },
      {
        id: '4',
        username: 'FactoryBuilder',
        name: 'Factory Builder',
        avatar: null,
        schematicsCount: 23
      }
    ];
    
    return {
      props: {
        categories: categoriesWithCount,
        featuredSchematics,
        featuredCreators
      }
    };
  } catch (error) {
    console.error('Error fetching data for explore page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        featuredSchematics: [],
        featuredCreators: []
      }
    };
  }
}