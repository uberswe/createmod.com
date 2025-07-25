import React, { useState } from 'react';
import Layout from '../../components/layout/Layout';
import SchematicCard from '../../components/schematics/SchematicCard';
import Pagination from '../../components/common/Pagination';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getUserByUsername, getSchematicsByAuthor, getCategories } from '../../lib/api';

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
  user, 
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
  
  // Format date
  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    }).format(date);
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
              {user.avatar ? (
                <span 
                  className="avatar avatar-xl mb-3 avatar-rounded" 
                  style={{ backgroundImage: `url(${user.avatar})` }}
                ></span>
              ) : (
                <span className="avatar avatar-xl mb-3 avatar-rounded">
                  {user.username.charAt(0).toUpperCase()}
                </span>
              )}
              <h3 className="m-0 mb-1">{user.name || user.username}</h3>
              <div className="text-muted">{user.username}</div>
              
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
                      <div className="h1 m-0">{user.joined ? formatDate(user.joined).split(' ')[2] : 'N/A'}</div>
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
                  <h3 className="card-title">About {isOwnProfile ? 'Me' : user.username}</h3>
                </div>
                <div className="card-body">
                  <div className="mb-3">
                    <div className="datagrid">
                      <div className="datagrid-item">
                        <div className="datagrid-title">Username</div>
                        <div className="datagrid-content">{user.username}</div>
                      </div>
                      
                      {user.name && (
                        <div className="datagrid-item">
                          <div className="datagrid-title">Name</div>
                          <div className="datagrid-content">{user.name}</div>
                        </div>
                      )}
                      
                      <div className="datagrid-item">
                        <div className="datagrid-title">Joined</div>
                        <div className="datagrid-content">{user.joined ? formatDate(user.joined) : 'N/A'}</div>
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
    
    // Check if user is authenticated
    const isAuthenticated = context.req.cookies['create-mod-auth'] !== undefined;
    let currentUser = null;
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Get user by username
    const userData = await getUserByUsername(username);
    
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
    
    const user = userData.items[0];
    
    // Check if this is the user's own profile
    const isOwnProfile = isAuthenticated && currentUser && currentUser.username.toLowerCase() === username.toLowerCase();
    
    // Get user's schematics
    const schematicsData = await getSchematicsByAuthor(user.id, {
      sort: '-created',
      filter: 'moderated=true',
      expand: 'author,categories,tags',
      page,
      perPage
    });
    
    const schematics = schematicsData.items || [];
    const totalItems = schematicsData.totalItems || 0;
    const totalPages = Math.ceil(totalItems / perPage);
    
    return {
      props: {
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
      }
    };
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