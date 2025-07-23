import React from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';

/**
 * Pagination component for navigating through paginated results
 * 
 * @param {Object} props - Component props
 * @param {number} props.currentPage - Current page number (1-based)
 * @param {number} props.totalPages - Total number of pages
 * @param {number} props.totalItems - Total number of items
 * @param {number} props.perPage - Number of items per page
 * @param {Function} props.onPageChange - Callback when page changes
 */
export default function Pagination({ 
  currentPage = 1, 
  totalPages = 1, 
  totalItems = 0, 
  perPage = 12,
  onPageChange 
}) {
  const router = useRouter();
  
  // Calculate start and end item numbers
  const startItem = (currentPage - 1) * perPage + 1;
  const endItem = Math.min(startItem + perPage - 1, totalItems);
  
  /**
   * Handle page change
   * @param {number} page - New page number
   * @param {Event} e - Click event
   */
  const handlePageChange = (page, e) => {
    if (e) e.preventDefault();
    
    // Don't navigate to invalid pages
    if (page < 1 || page > totalPages) return;
    
    if (onPageChange) {
      onPageChange(page);
    } else {
      // Default behavior: update URL query params
      const newQuery = { ...router.query, page };
      router.push({
        pathname: router.pathname,
        query: newQuery
      }, undefined, { scroll: true });
    }
  };
  
  // Generate array of page numbers to display
  const getPageNumbers = () => {
    const pages = [];
    const maxPagesToShow = 5; // Show at most 5 page numbers
    
    if (totalPages <= maxPagesToShow) {
      // If we have fewer pages than maxPagesToShow, show all pages
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      // Always include first and last page
      pages.push(1);
      
      // Calculate start and end of page range
      let startPage = Math.max(2, currentPage - 1);
      let endPage = Math.min(totalPages - 1, currentPage + 1);
      
      // Adjust if we're near the beginning or end
      if (currentPage <= 2) {
        endPage = 4;
      } else if (currentPage >= totalPages - 1) {
        startPage = totalPages - 3;
      }
      
      // Add ellipsis if needed
      if (startPage > 2) {
        pages.push('...');
      }
      
      // Add page numbers in range
      for (let i = startPage; i <= endPage; i++) {
        pages.push(i);
      }
      
      // Add ellipsis if needed
      if (endPage < totalPages - 1) {
        pages.push('...');
      }
      
      // Add last page if not already included
      if (totalPages > 1) {
        pages.push(totalPages);
      }
    }
    
    return pages;
  };
  
  // If there's only one page, don't show pagination
  if (totalPages <= 1) return null;
  
  return (
    <div className="d-flex align-items-center justify-content-between mt-4">
      <div className="text-muted">
        Showing <span className="fw-bold">{startItem}</span> to <span className="fw-bold">{endItem}</span> of <span className="fw-bold">{totalItems}</span> results
      </div>
      
      <ul className="pagination m-0">
        {/* Previous page button */}
        <li className={`page-item ${currentPage === 1 ? 'disabled' : ''}`}>
          <a 
            href="#" 
            className="page-link" 
            onClick={(e) => handlePageChange(currentPage - 1, e)}
            aria-label="Previous"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
              <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
              <path d="M15 6l-6 6l6 6" />
            </svg>
            prev
          </a>
        </li>
        
        {/* Page numbers */}
        {getPageNumbers().map((page, index) => (
          <li 
            key={`page-${index}`} 
            className={`page-item ${page === currentPage ? 'active' : ''} ${page === '...' ? 'disabled' : ''}`}
          >
            {page === '...' ? (
              <span className="page-link">...</span>
            ) : (
              <a 
                href="#" 
                className="page-link" 
                onClick={(e) => handlePageChange(page, e)}
              >
                {page}
              </a>
            )}
          </li>
        ))}
        
        {/* Next page button */}
        <li className={`page-item ${currentPage === totalPages ? 'disabled' : ''}`}>
          <a 
            href="#" 
            className="page-link" 
            onClick={(e) => handlePageChange(currentPage + 1, e)}
            aria-label="Next"
          >
            next
            <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
              <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
              <path d="M9 6l6 6l-6 6" />
            </svg>
          </a>
        </li>
      </ul>
    </div>
  );
}