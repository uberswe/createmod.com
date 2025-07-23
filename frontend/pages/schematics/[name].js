import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import SchematicCard from '../../components/schematics/SchematicCard';
import Link from 'next/link';
import Image from 'next/image';
import { getSchematicByName, getCategories, getSchematicComments, postComment, rateSchematic, getRecords } from '../../lib/api';
import { useRouter } from 'next/router';

/**
 * Schematic detail page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Object} props.schematic - Schematic data
 * @param {Array} props.fromAuthor - Other schematics from the same author
 * @param {Array} props.similar - Similar schematics
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.notFound - Whether the schematic was not found
 */
export default function SchematicDetail({ schematic, fromAuthor, similar, categories, notFound }) {
  const router = useRouter();
  const [activeImage, setActiveImage] = useState(0);
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState('');
  const [showRatingSuccess, setShowRatingSuccess] = useState(false);
  const [comments, setComments] = useState([]);
  const [loadingComments, setLoadingComments] = useState(false);
  
  // Fetch comments when component mounts or schematic changes
  useEffect(() => {
    if (schematic && schematic.id) {
      fetchComments();
    }
  }, [schematic]);
  
  // Function to fetch comments
  const fetchComments = async () => {
    setLoadingComments(true);
    try {
      const commentsData = await getSchematicComments(schematic.id);
      setComments(commentsData.items || []);
    } catch (error) {
      console.error('Error fetching comments:', error);
    } finally {
      setLoadingComments(false);
    }
  };
  
  // Handle 404 case
  if (notFound) {
    return (
      <Layout 
        title="Schematic Not Found" 
        description="The requested schematic could not be found"
        categories={categories}
      >
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
          <p className="empty-title">Schematic Not Found</p>
          <p className="empty-subtitle text-muted">
            The schematic you are looking for does not exist or has been removed.
          </p>
          <div className="empty-action">
            <Link href="/search" className="btn btn-primary">
              Browse Schematics
            </Link>
          </div>
        </div>
      </Layout>
    );
  }
  
  // If the page is still loading (router.isFallback is true)
  if (router.isFallback) {
    return (
      <Layout title="Loading..." description="Loading schematic details" categories={categories}>
        <div className="d-flex justify-content-center align-items-center" style={{ height: '50vh' }}>
          <div className="spinner-border text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Format date
  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    }).format(date);
  };
  
  // Handle rating change
  const handleRatingChange = async (newRating) => {
    setRating(newRating);
    try {
      // Send rating to the API
      await rateSchematic(schematic.id, newRating);
      setShowRatingSuccess(true);
      setTimeout(() => setShowRatingSuccess(false), 3000);
    } catch (error) {
      console.error('Error rating schematic:', error);
      alert('Failed to save rating. Please try again.');
    }
  };
  
  // Handle comment submission
  const handleCommentSubmit = async (e) => {
    e.preventDefault();
    
    if (!comment.trim()) {
      alert('Please enter a comment before submitting.');
      return;
    }
    
    try {
      // Send comment to the API
      await postComment({
        schematic: schematic.id,
        content: comment,
        approved: false // Comments require approval before being displayed
      });
      
      setComment('');
      
      // Refresh comments list
      fetchComments();
      
      alert('Your comment has been submitted and is pending approval.');
    } catch (error) {
      console.error('Error posting comment:', error);
      alert('Failed to submit comment. Please try again.');
    }
  };
  
  // Handle copy link
  const handleCopyLink = () => {
    const url = `${window.location.origin}/schematics/${schematic.name}`;
    navigator.clipboard.writeText(url);
    alert('Link copied to clipboard!');
  };
  
  // Get all images (featured + gallery)
  const allImages = [
    schematic.featured_image,
    ...(schematic.gallery || [])
  ];
  
  return (
    <Layout 
      title={schematic.title}
      description={schematic.content?.replace(/<[^>]*>?/gm, '').substring(0, 160) || ''}
      thumbnail={`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090'}/api/files/schematics/${schematic.id}/${schematic.featured_image}`}
      slug={`schematics/${schematic.name}`}
      categories={categories}
    >
      <div className="row g-2 g-md-3">
        <div className="col-lg-7">
          <div className="row row-cards">
            <div className="col-sm-12 col-lg-12">
              <div className="card card-sm mb-4">
                {/* Main image */}
                <div 
                  className="img-responsive img-responsive-21x9 card-img-top" 
                  style={{
                    backgroundImage: `url(${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090'}/api/files/schematics/${schematic.id}/${allImages[activeImage]})`
                  }}
                ></div>
                
                <div className="card-body">
                  {/* Gallery thumbnails */}
                  {schematic.gallery && schematic.gallery.length > 0 && (
                    <div className="row row-cols-6 g-3 mb-2">
                      {allImages.map((image, index) => (
                        <div className="col" key={index}>
                          <div 
                            className={`img-responsive img-responsive-1x1 rounded border ${index === activeImage ? 'border-primary' : ''}`}
                            style={{
                              backgroundImage: `url(${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090'}/api/files/schematics/${schematic.id}/${image}?thumb=150x150)`,
                              cursor: 'pointer'
                            }}
                            onClick={() => setActiveImage(index)}
                          ></div>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  {/* Description and actions */}
                  <div className="row mt-4">
                    <div className="col">
                      <h3 className="card-title">Description</h3>
                    </div>
                    <div className="col-auto pt-2">
                      <div className="star-rating">
                        {[1, 2, 3, 4, 5].map((star) => (
                          <span 
                            key={star}
                            className={`star ${rating >= star ? 'filled' : ''}`}
                            onClick={() => handleRatingChange(star)}
                            style={{ cursor: 'pointer', fontSize: '1.5rem', color: rating >= star ? 'gold' : 'gray' }}
                          >
                            ★
                          </span>
                        ))}
                      </div>
                      {showRatingSuccess && (
                        <div className="text-success">Rating saved!</div>
                      )}
                    </div>
                    <div className="col-auto">
                      <button className="btn btn-secondary" onClick={handleCopyLink}>
                        Copy Link
                      </button>
                    </div>
                    <div className="col-auto">
                      <a 
                        href={`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090'}/api/files/schematics/${schematic.id}/${schematic.schematic_file}`} 
                        className="btn btn-primary"
                        download
                      >
                        Download
                      </a>
                    </div>
                  </div>
                  
                  {/* Content */}
                  <div className="d-flex align-items-center">
                    <div dangerouslySetInnerHTML={{ __html: schematic.content }}></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          {/* Schematic details */}
          <div className="row row-cards mb-4">
            <div className="col-sm-12 col-lg-12">
              <div className="card">
                <div className="card-body">
                  <div className="datagrid">
                    <div className="datagrid-item">
                      <div className="datagrid-title">Mod Version</div>
                      <div className="datagrid-content">{schematic.expand?.createmod_version?.version || 'Unknown'}</div>
                    </div>
                    <div className="datagrid-item">
                      <div className="datagrid-title">Game Version</div>
                      <div className="datagrid-content">{schematic.expand?.minecraft_version?.version || 'Unknown'}</div>
                    </div>
                    <div className="datagrid-item">
                      <div className="datagrid-title">Category</div>
                      <div className="datagrid-content">
                        <div className="avatar-list avatar-list-stacked">
                          {schematic.expand?.categories?.map((category) => (
                            <Link 
                              key={category.id} 
                              href={`/search?category=${category.key}`}
                              className="badge badge-outline text-blue"
                            >
                              {category.name}
                            </Link>
                          ))}
                        </div>
                      </div>
                    </div>
                    <div className="datagrid-item">
                      <div className="datagrid-title">Uploaded</div>
                      <div className="datagrid-content" title={formatDate(schematic.created)}>
                        {formatDate(schematic.created)}
                      </div>
                    </div>
                    <div className="datagrid-item">
                      <div className="datagrid-title">Author</div>
                      <div className="datagrid-content">
                        <div className="d-flex align-items-center">
                          {schematic.expand?.author?.avatar && (
                            <span 
                              className="avatar avatar-xs me-2 rounded" 
                              style={{ backgroundImage: `url(${schematic.expand.author.avatar})` }}
                            ></span>
                          )}
                          <Link href={`/author/${schematic.expand?.author?.username?.toLowerCase() || ''}`}>
                            {schematic.expand?.author?.username || 'Unknown'}
                          </Link>
                        </div>
                      </div>
                    </div>
                    <div className="datagrid-item">
                      <div className="datagrid-title">Views</div>
                      <div className="datagrid-content">{schematic.views || 0}</div>
                    </div>
                    {schematic.rating && (
                      <div className="datagrid-item">
                        <div className="datagrid-title">Rating</div>
                        <div className="datagrid-content">
                          {schematic.rating} based on {schematic.rating_count || 0} ratings
                        </div>
                      </div>
                    )}
                    {schematic.expand?.tags && schematic.expand.tags.length > 0 && (
                      <div className="datagrid-item">
                        <div className="datagrid-title">Tags</div>
                        <div className="datagrid-content">
                          <div className="avatar-list avatar-list-stacked">
                            {schematic.expand.tags.map((tag) => (
                              <Link 
                                key={tag.id} 
                                href={`/search?tag=${tag.key}`}
                                className="badge badge-outline text-blue"
                              >
                                {tag.name}
                              </Link>
                            ))}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                  
                  {/* Dependencies */}
                  {schematic.has_dependencies && (
                    <div className="col-sm-12 col-lg-12 mt-4">
                      <h3 className="card-title">Dependencies</h3>
                      <div dangerouslySetInnerHTML={{ __html: schematic.dependencies }}></div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
          
          {/* Comments section */}
          <div className="row row-cards mb-4">
            <div className="col-sm-12 col-lg-12">
              <div className="card">
                <div className="card-body">
                  <h3 className="card-title">Comments</h3>
                  
                  {/* Comment form */}
                  <form onSubmit={handleCommentSubmit} className="mb-4">
                    <div className="mb-3">
                      <label className="form-label">Leave a comment</label>
                      <textarea 
                        className="form-control" 
                        rows="4"
                        value={comment}
                        onChange={(e) => setComment(e.target.value)}
                        placeholder="Write your comment here..."
                      ></textarea>
                    </div>
                    <button type="submit" className="btn btn-primary">Post Comment</button>
                  </form>
                  
                  {/* Comments display */}
                  {loadingComments ? (
                    <div className="d-flex justify-content-center my-4">
                      <div className="spinner-border text-primary" role="status">
                        <span className="visually-hidden">Loading comments...</span>
                      </div>
                    </div>
                  ) : comments.length > 0 ? (
                    <div className="comments">
                      {comments.map((comment) => (
                        <div key={comment.id} className="card mb-3">
                          <div className="card-body">
                            <div className="d-flex align-items-center mb-2">
                              {comment.expand?.author?.avatar ? (
                                <span 
                                  className="avatar avatar-sm me-2" 
                                  style={{ backgroundImage: `url(${comment.expand.author.avatar})` }}
                                ></span>
                              ) : (
                                <span className="avatar avatar-sm me-2">
                                  {comment.expand?.author?.username?.charAt(0).toUpperCase() || '?'}
                                </span>
                              )}
                              <div>
                                <strong>
                                  {comment.expand?.author?.username || 'Anonymous'}
                                </strong>
                                <small className="text-muted ms-2">
                                  {formatDate(comment.created)}
                                </small>
                              </div>
                            </div>
                            <p className="mb-0">{comment.content}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-muted text-center my-4">
                      No comments yet. Be the first to comment!
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <div className="col-lg-5">
          {/* Video */}
          {schematic.video && (
            <div className="card mb-4">
              <div className="ratio ratio-16x9">
                <iframe 
                  src={`https://www.youtube.com/embed/${schematic.video}`} 
                  title="YouTube video" 
                  allowFullScreen
                ></iframe>
              </div>
            </div>
          )}
          
          {/* More from author */}
          {fromAuthor && fromAuthor.length > 0 && (
            <div className="col-12 mb-4">
              <h3>More From <Link href={`/author/${schematic.expand?.author?.username?.toLowerCase() || ''}`}>
                {schematic.expand?.author?.username || 'Unknown'}
              </Link></h3>
              
              <div className="row row-cards">
                {fromAuthor.map((item) => (
                  <div className="col-lg-12" key={item.id}>
                    <SchematicCard schematic={item} />
                  </div>
                ))}
              </div>
            </div>
          )}
          
          {/* Similar schematics */}
          {similar && similar.length > 0 && (
            <div className="col-12 mb-4">
              <h3>Similar Schematics</h3>
              
              <div className="row row-cards">
                {similar.map((item) => (
                  <div className="col-lg-12" key={item.id}>
                    <SchematicCard schematic={item} />
                  </div>
                ))}
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
    const { name } = context.params;
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Get schematic by name
    const schematicData = await getSchematicByName(name);
    
    // If no schematic found, return notFound flag
    if (!schematicData.items || schematicData.items.length === 0) {
      return {
        props: {
          notFound: true,
          categories
        }
      };
    }
    
    const schematic = schematicData.items[0];
    
    let fromAuthor = [];
    let similar = [];
    
    // If the schematic has an author, fetch more from the same author
    if (schematic.expand?.author) {
      try {
        // Fetch more schematics from the same author (excluding current schematic)
        const authorSchematicsData = await getRecords('schematics', {
          filter: `author="${schematic.expand.author.id}" && id!="${schematic.id}" && moderated=true && deleted=null`,
          sort: '-created',
          expand: 'author,categories,tags',
          limit: 3
        });
        
        fromAuthor = authorSchematicsData.items || [];
      } catch (error) {
        console.error('Error fetching author schematics:', error);
      }
    }
    
    // Fetch similar schematics based on categories
    try {
      // Get category IDs from the current schematic
      const categoryIds = schematic.expand?.categories?.map(cat => cat.id) || [];
      
      if (categoryIds.length > 0) {
        // Create a filter to find schematics with at least one matching category
        const categoryFilter = categoryIds.map(id => `categories.id ?= "${id}"`).join(' || ');
        
        const similarSchematicsData = await getRecords('schematics', {
          filter: `(${categoryFilter}) && id!="${schematic.id}" && moderated=true && deleted=null`,
          sort: '-created',
          expand: 'author,categories,tags',
          limit: 3
        });
        
        similar = similarSchematicsData.items || [];
      }
    } catch (error) {
      console.error('Error fetching similar schematics:', error);
    }
    
    return {
      props: {
        schematic,
        fromAuthor,
        similar,
        categories,
        notFound: false
      }
    };
  } catch (error) {
    console.error('Error fetching schematic details:', error);
    
    return {
      props: {
        notFound: true,
        categories: []
      }
    };
  }
}