import React from 'react';
import Link from 'next/link';
import Image from 'next/image';

/**
 * SchematicCard component for displaying a schematic preview
 * 
 * @param {Object} props - Component props
 * @param {Object} props.schematic - Schematic data
 */
export default function SchematicCard({ schematic }) {
  if (!schematic) return null;
  
  // Format the date
  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    }).format(date);
  };
  
  // Truncate text to a certain length
  const truncate = (text, length = 100) => {
    if (!text) return '';
    return text.length > length ? text.substring(0, length) + '...' : text;
  };
  
  // Strip HTML tags from content
  const stripHtml = (html) => {
    return html?.replace(/<[^>]*>?/gm, '') || '';
  };
  
  return (
    <div className="card card-sm">
      <Link href={`/schematics/${schematic.name}`} className="d-block">
        <div className="img-responsive img-responsive-21x9 card-img-top" style={{
          backgroundImage: `url(${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090'}/api/files/schematics/${schematic.id}/${schematic.featured_image}?thumb=600x400)`
        }}></div>
      </Link>
      <div className="card-body">
        <div className="d-flex align-items-center mb-2">
          {schematic.expand?.author && (
            <div className="me-2">
              {schematic.expand.author.avatar && (
                <span className="avatar avatar-xs" style={{
                  backgroundImage: `url(${schematic.expand.author.avatar})`
                }}></span>
              )}
            </div>
          )}
          <div>
            <div className="font-weight-medium">
              <Link href={`/schematics/${schematic.name}`} className="text-reset">
                {schematic.title}
              </Link>
            </div>
            <div className="text-muted">
              by{' '}
              <Link href={`/author/${schematic.expand?.author?.username?.toLowerCase() || ''}`} className="text-reset">
                {schematic.expand?.author?.username || 'Unknown'}
              </Link>
              {' '}&bull;{' '}
              {formatDate(schematic.created)}
            </div>
          </div>
        </div>
        
        <div className="text-muted mb-3">
          {truncate(stripHtml(schematic.content), 120)}
        </div>
        
        <div className="d-flex align-items-center justify-content-between">
          <div>
            {schematic.expand?.categories?.map((category) => (
              <Link 
                key={category.id} 
                href={`/search?category=${category.key}`}
                className="badge badge-outline text-blue me-1"
              >
                {category.name}
              </Link>
            ))}
            {schematic.expand?.tags?.map((tag) => (
              <Link 
                key={tag.id} 
                href={`/search?tag=${tag.key}`}
                className="badge badge-outline text-secondary me-1"
              >
                {tag.name}
              </Link>
            ))}
          </div>
          <div className="text-muted">
            <svg xmlns="http://www.w3.org/2000/svg" className="icon me-1" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
              <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
              <path d="M12 12m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0" />
              <path d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7" />
            </svg>
            {schematic.views || 0}
          </div>
        </div>
      </div>
    </div>
  );
}