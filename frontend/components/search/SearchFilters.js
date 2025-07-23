import React from 'react';
import { useRouter } from 'next/router';

/**
 * SearchFilters component for filtering schematic search results
 * 
 * @param {Object} props - Component props
 * @param {Array} props.categories - Available categories
 * @param {Array} props.tags - Available tags
 * @param {Object} props.currentFilters - Current active filters
 * @param {Function} props.onFilterChange - Callback when filters change
 */
export default function SearchFilters({ categories = [], tags = [], currentFilters = {}, onFilterChange }) {
  const router = useRouter();
  
  /**
   * Handle filter change
   * @param {string} filterType - Type of filter (category, tag, etc.)
   * @param {string} value - New filter value
   */
  const handleFilterChange = (filterType, value) => {
    if (onFilterChange) {
      onFilterChange(filterType, value);
    } else {
      // Default behavior: update URL query params
      const newQuery = { ...router.query, [filterType]: value };
      
      // Reset to page 1 when filters change
      if (newQuery.page) {
        newQuery.page = 1;
      }
      
      router.push({
        pathname: router.pathname,
        query: newQuery
      }, undefined, { scroll: false });
    }
  };
  
  /**
   * Get the current value of a filter
   * @param {string} filterType - Type of filter
   * @param {string} defaultValue - Default value if not set
   * @returns {string} - Current filter value
   */
  const getCurrentValue = (filterType, defaultValue = '') => {
    return currentFilters[filterType] || router.query[filterType] || defaultValue;
  };
  
  return (
    <div className="card">
      <div className="card-header">
        <h3 className="card-title">Filters</h3>
      </div>
      <div className="card-body">
        {/* Sort */}
        <div className="mb-3">
          <label className="form-label">Sort By</label>
          <select 
            className="form-select" 
            value={getCurrentValue('sort', '1')}
            onChange={(e) => handleFilterChange('sort', e.target.value)}
          >
            <option value="1">Newest First</option>
            <option value="2">Oldest First</option>
            <option value="3">Most Views</option>
            <option value="4">Highest Rated</option>
            <option value="5">Alphabetical (A-Z)</option>
            <option value="6">Random</option>
          </select>
        </div>
        
        {/* Category */}
        <div className="mb-3">
          <label className="form-label">Category</label>
          <select 
            className="form-select" 
            value={getCurrentValue('category', 'all')}
            onChange={(e) => handleFilterChange('category', e.target.value)}
          >
            <option value="all">All Categories</option>
            {categories.map((category) => (
              <option key={category.id} value={category.key}>
                {category.name}
              </option>
            ))}
          </select>
        </div>
        
        {/* Tags */}
        {tags.length > 0 && (
          <div className="mb-3">
            <label className="form-label">Tag</label>
            <select 
              className="form-select" 
              value={getCurrentValue('tag', 'all')}
              onChange={(e) => handleFilterChange('tag', e.target.value)}
            >
              <option value="all">All Tags</option>
              {tags.map((tag) => (
                <option key={tag.id} value={tag.key}>
                  {tag.name}
                </option>
              ))}
            </select>
          </div>
        )}
        
        {/* Rating */}
        <div className="mb-3">
          <label className="form-label">Minimum Rating</label>
          <select 
            className="form-select" 
            value={getCurrentValue('rating', '-1')}
            onChange={(e) => handleFilterChange('rating', e.target.value)}
          >
            <option value="-1">Any Rating</option>
            <option value="1">★ and above</option>
            <option value="2">★★ and above</option>
            <option value="3">★★★ and above</option>
            <option value="4">★★★★ and above</option>
            <option value="5">★★★★★ only</option>
          </select>
        </div>
        
        {/* Minecraft Version */}
        <div className="mb-3">
          <label className="form-label">Minecraft Version</label>
          <select 
            className="form-select" 
            value={getCurrentValue('minecraft', 'all')}
            onChange={(e) => handleFilterChange('minecraft', e.target.value)}
          >
            <option value="all">All Versions</option>
            <option value="1.20">1.20</option>
            <option value="1.19">1.19</option>
            <option value="1.18">1.18</option>
            <option value="1.17">1.17</option>
            <option value="1.16">1.16</option>
          </select>
        </div>
        
        {/* Create Mod Version */}
        <div className="mb-3">
          <label className="form-label">Create Mod Version</label>
          <select 
            className="form-select" 
            value={getCurrentValue('createmod', 'all')}
            onChange={(e) => handleFilterChange('createmod', e.target.value)}
          >
            <option value="all">All Versions</option>
            <option value="0.5.1">0.5.1</option>
            <option value="0.5.0">0.5.0</option>
            <option value="0.4.1">0.4.1</option>
            <option value="0.4.0">0.4.0</option>
          </select>
        </div>
      </div>
    </div>
  );
}