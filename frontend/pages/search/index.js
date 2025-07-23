import React, { useState } from 'react';
import Layout from '../../components/layout/Layout';
import SchematicCard from '../../components/schematics/SchematicCard';
import SearchFilters from '../../components/search/SearchFilters';
import Pagination from '../../components/common/Pagination';
import { getSchematics, getCategories, getTags } from '../../lib/api';
import { useRouter } from 'next/router';

/**
 * Search page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.schematics - Schematics data
 * @param {Array} props.categories - Categories data
 * @param {Array} props.tags - Tags data
 * @param {number} props.totalItems - Total number of items
 * @param {number} props.totalPages - Total number of pages
 * @param {Object} props.filters - Current active filters
 */
export default function Search({ 
  schematics = [], 
  categories = [], 
  tags = [], 
  totalItems = 0, 
  totalPages = 1,
  filters = {}
}) {
  const router = useRouter();
  const [searchTerm, setSearchTerm] = useState(router.query.term || '');
  
  // Get current page from URL or default to 1
  const currentPage = parseInt(router.query.page || '1', 10);
  
  // Items per page
  const perPage = 12;
  
  /**
   * Handle search form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSearch = (e) => {
    e.preventDefault();
    if (searchTerm.trim()) {
      const slug = searchTerm.trim()
        .toLowerCase()
        .replace(/[^a-z0-9 -]/g, '')
        .replace(/\s+/g, '-')
        .replace(/-+/g, '-');
      
      router.push(`/search/${slug}`);
    }
  };
  
  // Determine page title based on filters
  const getPageTitle = () => {
    if (router.query.term) {
      return `Search results for "${router.query.term}"`;
    }
    
    if (router.query.category && router.query.category !== 'all') {
      const category = categories.find(c => c.key === router.query.category);
      if (category) return `${category.name} Schematics`;
    }
    
    if (router.query.tag && router.query.tag !== 'all') {
      const tag = tags.find(t => t.key === router.query.tag);
      if (tag) return `Schematics tagged with "${tag.name}"`;
    }
    
    return 'Browse Schematics';
  };
  
  return (
    <Layout 
      title={getPageTitle()}
      description="Browse and search for Create mod schematics"
      categories={categories}
    >
      <div className="row mb-4">
        <div className="col-12">
          <div className="card">
            <div className="card-body">
              <h1 className="card-title">{getPageTitle()}</h1>
              
              {/* Search form */}
              <form onSubmit={handleSearch} className="mt-3">
                <div className="input-icon mb-3">
                  <input 
                    type="text" 
                    className="form-control" 
                    placeholder="Search schematics..." 
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                  />
                  <span className="input-icon-addon">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <circle cx="10" cy="10" r="7"/>
                      <line x1="21" y1="21" x2="15" y2="15"/>
                    </svg>
                  </span>
                </div>
                <button type="submit" className="btn btn-primary">Search</button>
              </form>
            </div>
          </div>
        </div>
      </div>
      
      <div className="row g-4">
        {/* Filters sidebar */}
        <div className="col-lg-3">
          <SearchFilters 
            categories={categories} 
            tags={tags}
            currentFilters={filters}
          />
        </div>
        
        {/* Results */}
        <div className="col-lg-9">
          {schematics.length > 0 ? (
            <>
              <div className="row row-cards">
                {schematics.map((schematic) => (
                  <div className="col-sm-6 col-lg-4" key={schematic.id}>
                    <SchematicCard schematic={schematic} />
                  </div>
                ))}
              </div>
              
              {/* Pagination */}
              <Pagination 
                currentPage={currentPage}
                totalPages={totalPages}
                totalItems={totalItems}
                perPage={perPage}
              />
            </>
          ) : (
            <div className="empty">
              <div className="empty-icon">
                <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-search-off" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                  <path d="M5.039 5.062a7 7 0 0 0 9.91 9.89m1.584 -2.434a7 7 0 0 0 -9.038 -9.057"></path>
                  <path d="M3 3l18 18"></path>
                </svg>
              </div>
              <p className="empty-title">No results found</p>
              <p className="empty-subtitle text-muted">
                Try adjusting your search or filter criteria to find what you're looking for.
              </p>
              <div className="empty-action">
                <a href="/search" className="btn btn-primary">
                  Clear all filters
                </a>
              </div>
            </div>
          )}
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
    // Get page from query params or default to 1
    const page = parseInt(context.query.page || '1', 10);
    const perPage = 12;
    
    // Build filter object from query params
    const filters = {
      sort: context.query.sort || '1',
      category: context.query.category || 'all',
      tag: context.query.tag || 'all',
      rating: context.query.rating || '-1',
      minecraft: context.query.minecraft || 'all',
      createmod: context.query.createmod || 'all',
      term: context.query.term || ''
    };
    
    // Get categories for sidebar and filters
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Get tags for filters
    const tagsData = await getTags();
    const tags = tagsData.items || [];
    
    // Build filter string for API
    let filterString = 'moderated=true && deleted=null';
    
    // Add category filter
    if (filters.category !== 'all') {
      const category = categories.find(c => c.key === filters.category);
      if (category) {
        filterString += ` && categories.id ?= "${category.id}"`;
      }
    }
    
    // Add tag filter
    if (filters.tag !== 'all') {
      const tag = tags.find(t => t.key === filters.tag);
      if (tag) {
        filterString += ` && tags.id ?= "${tag.id}"`;
      }
    }
    
    // Add minecraft version filter
    if (filters.minecraft !== 'all') {
      filterString += ` && minecraft_version.version = "${filters.minecraft}"`;
    }
    
    // Add createmod version filter
    if (filters.createmod !== 'all') {
      filterString += ` && createmod_version.version = "${filters.createmod}"`;
    }
    
    // Add rating filter
    if (filters.rating !== '-1') {
      filterString += ` && rating >= ${filters.rating}`;
    }
    
    // Add search term filter
    if (filters.term) {
      filterString += ` && (title ~ "${filters.term}" || content ~ "${filters.term}")`;
    }
    
    // Determine sort order
    let sortString = '-created';
    switch (filters.sort) {
      case '1': sortString = '-created'; break; // Newest
      case '2': sortString = '+created'; break; // Oldest
      case '3': sortString = '-views'; break; // Most views
      case '4': sortString = '-rating'; break; // Highest rated
      case '5': sortString = '+title'; break; // Alphabetical
      case '6': sortString = '@random'; break; // Random
      default: sortString = '-created';
    }
    
    // Get schematics with filters
    const schematicsData = await getSchematics({
      sort: sortString,
      filter: filterString,
      expand: 'author,categories,tags,minecraft_version,createmod_version',
      page: page,
      perPage: perPage
    });
    
    const schematics = schematicsData.items || [];
    const totalItems = schematicsData.totalItems || 0;
    const totalPages = Math.ceil(totalItems / perPage);
    
    return {
      props: {
        schematics,
        categories,
        tags,
        totalItems,
        totalPages,
        filters
      }
    };
  } catch (error) {
    console.error('Error fetching search results:', error);
    
    // Return empty data on error
    return {
      props: {
        schematics: [],
        categories: [],
        tags: [],
        totalItems: 0,
        totalPages: 1,
        filters: {}
      }
    };
  }
}