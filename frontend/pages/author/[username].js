import React, { useState } from 'react';
import Layout from '../../components/layout/Layout';
import SchematicCard from '../../components/schematics/SchematicCard';
import Pagination from '../../components/common/Pagination';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getUserByUsername, getSchematicsByAuthor, getCategories } from '../../lib/api';
import { validateServerAuth } from '../../lib/auth';

/**
 * User profile page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Object} props.user - User data
 * @param {Array} props.schematics - User's schematics
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether viewer is authenticated
 * @param {Object} props.currentUser - Current authenticated user data
 * @param {boolean} props.isOwnProfile - Whether viewing own profile
 * @param {boolean} props.userNotFound - Whether user was not found
 * @param {number} props.totalItems - Total number of schematics
 * @param {number} props.totalPages - Total number of pages
 * @param {number} props.currentPage - Current page number
 */
export default function UserProfile({ 
  user = {}, 
  schematics = [], 
  categories = [], 
  isAuthenticated = false,
  currentUser = null,
  isOwnProfile = false,
  userNotFound = false,
  totalItems = 0,
  totalPages = 1,
  currentPage = 1
}) {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('schematics');
  
  // Debug: Log user data to see what's available
  console.log('[DEBUG] User Profile Component - User data:', user);
  console.log('[DEBUG] User Profile Component - User data keys:', Object.keys(user));
  console.log('[DEBUG] User Profile Component - Is own profile:', isOwnProfile);
  console.log('[DEBUG] User Profile Component - Current user:', currentUser);
  console.log('[DEBUG] User Profile Component - Is authenticated:', isAuthenticated);
  
  // Additional error handling and validation
  if (!user || Object.keys(user).length === 0) {
    console.error('[ERROR] User Profile Component - User data is empty or undefined');
  }
  
  if (isOwnProfile && (!currentUser || Object.keys(currentUser).length === 0)) {
    console.error('[ERROR] User Profile Component - Own profile but currentUser is empty or undefined');
  }
  
  // Ensure user object has all required fields with fallbacks
  const safeUser = {
    id: user?.id || '',
    username: user?.username || 'Unknown User',
    name: user?.name || '',
    avatar: user?.avatar || '',
    joined: user?.joined || null,
    bio: user?.bio || '',
    ...user // Include any other fields from the original user object
  };
  
  // Use safeUser instead of user throughout the component
  user = safeUser;
  
  // Handle 404 case
  if (userNotFound) {
    return (
      <Layout 
        title="User Not Found" 
        description="The requested user could not be found"
        categories={categories}
        isAuthenticated={isAuthenticated}
        user={currentUser}
      >
        <div className="empty">
          <div className="empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-user-off" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
              <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
              <path d="M8.18 8.189a4.01 4.01 0 0 0 2.616 2.627m3.507 -.545a4 4 0 1 0 -5.59 -5.552"></path>
              <path d="M6 21v-2a4 4 0 0 1 4 -4h4c.412 0 .81 .062 1.183 .178m2.633 2.618c.12 .38 .184 .785 .184 1.204v2"></path>
              <path d="M3 3l18 18"></path>
            </svg>
          </div>
          <p className="empty-title">User Not Found</p>
          <p className="empty-subtitle text-muted">
            The user you are looking for does not exist or has been removed.
          </p>
          <div className="empty-action">
            <Link href="/" className="btn btn-primary">
              Go to Home Page
            </Link>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Format date with improved error handling
  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    
    try {
      const date = new Date(dateString);
      
      // Check if date is valid
      if (isNaN(date.getTime())) {
        console.error('[ERROR] Invalid date:', dateString);
        return 'N/A';
      }
      
      return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      }).format(date);
    } catch (error) {
      console.error('[ERROR] Error formatting date:', error);
      return 'N/A';
    }
  };
  
  // Handle page change
  const handlePageChange = (page) => {
    router.push({
      pathname: router.pathname,
      query: { ...router.query, page }
    });
  };
  
  return (
    <Layout 
      title={isOwnProfile ? 'My Profile' : `${user.username}'s Profile`}
      description={isOwnProfile ? 'View and manage your profile' : `View ${user.username}'s schematics and profile information`}
      categories={categories}
      isAuthenticated={isAuthenticated}
      user={currentUser}
    >
      <div className="row">
        {/* User profile card */}
        <div className="col-lg-4">
          <div className="card">
            <div className="card-body p-4 text-center">
              {/* Debug user data */}
              {console.log('[DEBUG] Rendering avatar - user.avatar:', user.avatar)}
              
              {user.avatar ? (
                <span 
                  className="avatar avatar-xl mb-3 avatar-rounded" 
                  style={{ backgroundImage: `url(${user.avatar})` }}
                ></span>
              ) : (
                <span className="avatar avatar-xl mb-3 avatar-rounded">
                  {user.username ? user.username.charAt(0).toUpperCase() : '?'}
                </span>
              )}
              
              {/* Debug name data */}
              {console.log('[DEBUG] Rendering name - user.name:', user.name)}
              
              <h3 className="m-0 mb-1">{user.name || user.username || 'Unknown User'}</h3>
              <div className="text-muted">{user.username || 'No username'}</div>
              
              <div className="mt-3">
                <div className="row g-2 text-center">
                  <div className="col-6">
                    <div className="border rounded p-2">
                      <div className="h1 m-0">{totalItems}</div>
                      <div className="text-muted">Schematics</div>
                    </div>
                  </div>
                  <div className="col-6">
                    <div className="border rounded p-2">
                      {/* Debug joined date */}
                      {console.log('[DEBUG] Rendering joined date - user.joined:', user.joined)}
                      
                      <div className="h1 m-0">
                        {user.joined ? 
                          (() => {
                            try {
                              return formatDate(user.joined).split(' ')[2];
                            } catch (error) {
                              console.error('[ERROR] Failed to format joined date:', error);
                              return 'N/A';
                            }
                          })() 
                          : 'N/A'}
                      </div>
                      <div className="text-muted">Joined</div>
                    </div>
                  </div>
                </div>
              </div>
              
              {isOwnProfile && (
                <div className="mt-3">
                  <Link href="/settings" className="btn btn-primary w-100">
                    Edit Profile
                  </Link>
                </div>
              )}
            </div>
            
            <div className="d-flex">
              <a 
                href="#" 
                className={`card-link flex-fill text-center py-2 ${activeTab === 'schematics' ? 'active' : ''}`}
                onClick={(e) => {
                  e.preventDefault();
                  setActiveTab('schematics');
                }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="icon me-1" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <path d="M14 3v4a1 1 0 0 0 1 1h4" />
                  <path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z" />
                  <path d="M9 17h6" />
                  <path d="M9 13h6" />
                </svg>
                Schematics
              </a>
              <a 
                href="#" 
                className={`card-link flex-fill text-center py-2 ${activeTab === 'about' ? 'active' : ''}`}
                onClick={(e) => {
                  e.preventDefault();
                  setActiveTab('about');
                }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="icon me-1" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0" />
                  <path d="M12 8l.01 0" />
                  <path d="M11 12l1 0l0 4l1 0" />
                </svg>
                About
              </a>
            </div>
          </div>
        </div>
        
        {/* Content area */}
        <div className="col-lg-8">
          {activeTab === 'schematics' ? (
            <>
              {/* Schematics tab */}
              <div className="card">
                <div className="card-header">
                  <h3 className="card-title">{isOwnProfile ? 'My Schematics' : `${user.username}'s Schematics`}</h3>
                </div>
                <div className="card-body">
                  {schematics.length > 0 ? (
                    <div className="row row-cards">
                      {schematics.map((schematic) => (
                        <div className="col-sm-6" key={schematic.id}>
                          <SchematicCard schematic={schematic} />
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="empty">
                      <div className="empty-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                          <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                          <path d="M14 3v4a1 1 0 0 0 1 1h4" />
                          <path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z" />
                          <path d="M9 17h6" />
                          <path d="M9 13h6" />
                        </svg>
                      </div>
                      <p className="empty-title">No schematics yet</p>
                      <p className="empty-subtitle text-muted">
                        {isOwnProfile ? 'You haven\'t uploaded any schematics yet.' : `${user.username} hasn't uploaded any schematics yet.`}
                      </p>
                      {isOwnProfile && (
                        <div className="empty-action">
                          <Link href="/upload" className="btn btn-primary">
                            Upload a Schematic
                          </Link>
                        </div>
                      )}
                    </div>
                  )}
                  
                  {/* Pagination */}
                  {totalPages > 1 && (
                    <Pagination 
                      currentPage={currentPage}
                      totalPages={totalPages}
                      totalItems={totalItems}
                      perPage={12}
                      onPageChange={handlePageChange}
                    />
                  )}
                </div>
              </div>
            </>
          ) : (
            <>
              {/* About tab */}
              <div className="card">
                <div className="card-header">
                  <h3 className="card-title">About {isOwnProfile ? 'Me' : (user.username || 'User')}</h3>
                </div>
                <div className="card-body">
                  {/* Debug about tab data */}
                  {console.log('[DEBUG] Rendering About tab - user data:', {
                    username: user.username,
                    name: user.name,
                    joined: user.joined,
                    bio: user.bio
                  })}
                  
                  <div className="mb-3">
                    <div className="datagrid">
                      <div className="datagrid-item">
                        <div className="datagrid-title">Username</div>
                        <div className="datagrid-content">{user.username || 'N/A'}</div>
                      </div>
                      
                      {user.name && (
                        <div className="datagrid-item">
                          <div className="datagrid-title">Name</div>
                          <div className="datagrid-content">{user.name}</div>
                        </div>
                      )}
                      
                      <div className="datagrid-item">
                        <div className="datagrid-title">Joined</div>
                        <div className="datagrid-content">
                          {user.joined ? 
                            (() => {
                              try {
                                return formatDate(user.joined);
                              } catch (error) {
                                console.error('[ERROR] Failed to format joined date in About tab:', error);
                                return 'N/A';
                              }
                            })() 
                            : 'N/A'}
                        </div>
                      </div>
                      
                      <div className="datagrid-item">
                        <div className="datagrid-title">Schematics</div>
                        <div className="datagrid-content">{totalItems}</div>
                      </div>
                    </div>
                  </div>
                  
                  {user.bio && (
                    <div className="mt-4">
                      <h4>Bio</h4>
                      <p>{user.bio}</p>
                    </div>
                  )}
                </div>
              </div>
            </>
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
    const { username } = context.params;
    const page = parseInt(context.query.page || '1', 10);
    const perPage = 12;
    
    console.log(`[DEBUG] getServerSideProps - Fetching profile for username: ${username}`);
    
    // Check if user is authenticated and get current user data
    const { isAuthenticated, user: authUser } = await validateServerAuth(context.req);
    
    // Create a serialization-safe version of the current user object
    const currentUser = authUser ? {
      id: authUser.id,
      username: authUser.username,
      email: authUser.email || null, // Use null instead of undefined for serialization
      name: authUser.name || '',
      avatar: authUser.avatar || '',
      joined: authUser.created || null, // Use created as joined date if available
      bio: authUser.bio || '',
      verified: authUser.verified || false // Use false instead of undefined for serialization
    } : null;
    
    console.log('[DEBUG] getServerSideProps - Auth result:', { 
      isAuthenticated, 
      currentUserId: currentUser?.id,
      currentUsername: currentUser?.username,
      serializedUser: !!currentUser
    });
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Check if this is the user's own profile by comparing usernames (case-insensitive)
    const isOwnProfile = isAuthenticated && currentUser && 
                        currentUser.username.toLowerCase() === username.toLowerCase();
    console.log('[DEBUG] getServerSideProps - isOwnProfile calculation:', {
      isAuthenticated,
      hasCurrentUser: !!currentUser,
      currentUsername: currentUser?.username?.toLowerCase(),
      requestedUsername: username.toLowerCase(),
      isOwnProfile
    });
    
    // If viewing own profile, use the authenticated user's data directly
    let user;
    if (isOwnProfile && currentUser) {
      console.log('[DEBUG] getServerSideProps - Using authenticated user data for own profile');
      user = currentUser;
    } else {
      // Otherwise, try to fetch the user by username
      const userData = await getUserByUsername(username);
      console.log('[DEBUG] getServerSideProps - User data from API:', userData);
      
      // If user not found, return 404
      if (!userData.items || userData.items.length === 0) {
        return {
          props: {
            userNotFound: true,
            categories,
            isAuthenticated,
            currentUser
          }
        };
      }
      
      // Get the raw user data from the API response
      const rawUser = userData.items[0];
      
      // Create a serialization-safe version of the profile user object
      user = {
        id: rawUser.id,
        username: rawUser.username,
        name: rawUser.name || '',
        avatar: rawUser.avatar || '',
        joined: rawUser.created || rawUser.joined || null, // Use created or joined date if available
        bio: rawUser.bio || '',
        email: rawUser.email || null, // Use null instead of undefined for serialization
        verified: rawUser.verified || false // Use false instead of undefined for serialization
      };
    }
    
    console.log('[DEBUG] getServerSideProps - Processed user data:', {
      id: user.id,
      username: user.username,
      name: user.name,
      avatar: user.avatar ? 'Has avatar' : 'No avatar',
      joined: user.joined,
      bio: user.bio ? 'Has bio' : 'No bio',
      serialized: true,
      allFields: Object.keys(user)
    });
    
    
    // Get user's schematics
    const schematicsData = await getSchematicsByAuthor(user.id, {
      sort: '-created',
      filter: 'moderated=true',
      expand: 'author,categories,tags',
      page,
      perPage
    });
    console.log('[DEBUG] getServerSideProps - Schematics data:', {
      count: schematicsData.items?.length || 0,
      totalItems: schematicsData.totalItems || 0
    });
    
    const schematics = schematicsData.items || [];
    const totalItems = schematicsData.totalItems || 0;
    const totalPages = Math.ceil(totalItems / perPage);
    
    // Prepare the final props object
    const props = {
      user,
      schematics,
      categories,
      isAuthenticated,
      currentUser,
      isOwnProfile,
      userNotFound: false,
      totalItems,
      totalPages,
      currentPage: page
    };
    
    // Log the final props being returned (excluding large arrays)
    console.log('[DEBUG] getServerSideProps - Final props:', {
      user: {
        id: props.user?.id,
        username: props.user?.username,
        hasName: !!props.user?.name,
        hasAvatar: !!props.user?.avatar,
        hasJoined: !!props.user?.joined,
        hasBio: !!props.user?.bio,
        isEmpty: !props.user || Object.keys(props.user).length === 0
      },
      currentUser: props.currentUser ? {
        id: props.currentUser.id,
        username: props.currentUser.username,
        isEmpty: Object.keys(props.currentUser).length === 0
      } : null,
      isAuthenticated: props.isAuthenticated,
      isOwnProfile: props.isOwnProfile,
      userNotFound: props.userNotFound,
      schematicsCount: props.schematics?.length || 0,
      categoriesCount: props.categories?.length || 0,
      totalItems: props.totalItems,
      totalPages: props.totalPages,
      currentPage: props.currentPage
    });
    
    // Additional validation before returning props
    if (!props.user || Object.keys(props.user).length === 0) {
      console.error('[ERROR] getServerSideProps - User data is empty or undefined');
      
      // If this is supposed to be the user's own profile, use currentUser as a fallback
      if (isOwnProfile && currentUser) {
        console.log('[DEBUG] getServerSideProps - Using currentUser as fallback for empty user data');
        props.user = currentUser;
      }
    }
    
    // Final check to ensure we're not returning empty user data
    if (!props.user || Object.keys(props.user).length === 0) {
      console.error('[ERROR] getServerSideProps - Still have empty user data after fallback attempt');
      return {
        props: {
          userNotFound: true,
          categories,
          isAuthenticated,
          currentUser
        }
      };
    }
    
    return { props };
  } catch (error) {
    console.error('Error fetching user profile:', error);
    
    // Return 404 on error
    return {
      props: {
        userNotFound: true,
        categories: [],
        isAuthenticated: false,
        currentUser: null
      }
    };
  }
}