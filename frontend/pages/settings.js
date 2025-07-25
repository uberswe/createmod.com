import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import Layout from '../components/layout/Layout';
import { updateRecord, getCategories, requestEmailVerification } from '../lib/api';
import { validateServerAuth } from '../lib/auth';

/**
 * User settings page component
 */
export default function Settings({ categories = [], isAuthenticated = false, user = null }) {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('profile');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState({ type: '', text: '' });
  
  // Form states
  const [profileData, setProfileData] = useState({
    username: user?.username || '',
    name: user?.name || '',
    email: user?.email || '',
    bio: user?.bio || ''
  });
  
  const [passwordData, setPasswordData] = useState({
    oldPassword: '',
    password: '',
    passwordConfirm: ''
  });
  
  // Redirect if not authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?redirect=/settings');
    }
  }, [isAuthenticated, router]);
  
  if (!isAuthenticated || !user) {
    return (
      <Layout 
        title="Settings" 
        description="Manage your account settings"
        categories={categories}
        isAuthenticated={isAuthenticated}
        user={user}
      >
        <div className="container-xl py-4">
          <div className="text-center">
            <div className="spinner-border" role="status">
              <span className="visually-hidden">Loading...</span>
            </div>
            <p className="mt-2">Redirecting to login...</p>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Handle profile form submission
  const handleProfileSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setMessage({ type: '', text: '' });
    
    try {
      // Update user record
      await updateRecord('users', user.id, {
        username: profileData.username,
        name: profileData.name,
        bio: profileData.bio
      });
      
      // Check if email was changed
      if (profileData.email !== user.email) {
        // Update email separately (requires verification)
        await updateRecord('users', user.id, {
          email: profileData.email,
          emailVisibility: false
        });
        
        // Request email verification
        await requestEmailVerification(profileData.email);
        
        setMessage({
          type: 'success',
          text: 'Profile updated successfully. A verification email has been sent to your new email address.'
        });
      } else {
        setMessage({
          type: 'success',
          text: 'Profile updated successfully.'
        });
      }
    } catch (error) {
      console.error('Error updating profile:', error);
      setMessage({
        type: 'error',
        text: error.message || 'An error occurred while updating your profile.'
      });
    } finally {
      setLoading(false);
    }
  };
  
  // Handle password form submission
  const handlePasswordSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setMessage({ type: '', text: '' });
    
    // Validate password match
    if (passwordData.password !== passwordData.passwordConfirm) {
      setMessage({
        type: 'error',
        text: 'New passwords do not match.'
      });
      setLoading(false);
      return;
    }
    
    try {
      // Update password
      await updateRecord('users', user.id, {
        oldPassword: passwordData.oldPassword,
        password: passwordData.password,
        passwordConfirm: passwordData.passwordConfirm
      });
      
      // Clear password fields
      setPasswordData({
        oldPassword: '',
        password: '',
        passwordConfirm: ''
      });
      
      setMessage({
        type: 'success',
        text: 'Password updated successfully.'
      });
    } catch (error) {
      console.error('Error updating password:', error);
      setMessage({
        type: 'error',
        text: error.message || 'An error occurred while updating your password.'
      });
    } finally {
      setLoading(false);
    }
  };
  
  // Handle input changes for profile form
  const handleProfileChange = (e) => {
    const { name, value } = e.target;
    setProfileData(prev => ({
      ...prev,
      [name]: value
    }));
  };
  
  // Handle input changes for password form
  const handlePasswordChange = (e) => {
    const { name, value } = e.target;
    setPasswordData(prev => ({
      ...prev,
      [name]: value
    }));
  };
  
  return (
    <Layout 
      title="Account Settings" 
      description="Manage your account settings"
      categories={categories}
      isAuthenticated={isAuthenticated}
      user={user}
    >
      <div className="container-xl py-4">
        <div className="page-header d-print-none">
          <div className="row align-items-center">
            <div className="col">
              <h2 className="page-title">Account Settings</h2>
              <div className="text-muted mt-1">Manage your account preferences</div>
            </div>
          </div>
        </div>
        
        <div className="row mt-3">
          <div className="col-md-3 mb-3">
            <div className="card">
              <div className="card-body">
                <div className="list-group list-group-transparent">
                  <a 
                    href="#" 
                    className={`list-group-item list-group-item-action d-flex align-items-center ${activeTab === 'profile' ? 'active' : ''}`}
                    onClick={(e) => {
                      e.preventDefault();
                      setActiveTab('profile');
                      setMessage({ type: '', text: '' });
                    }}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon me-2" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0" />
                      <path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" />
                    </svg>
                    Profile Information
                  </a>
                  <a 
                    href="#" 
                    className={`list-group-item list-group-item-action d-flex align-items-center ${activeTab === 'password' ? 'active' : ''}`}
                    onClick={(e) => {
                      e.preventDefault();
                      setActiveTab('password');
                      setMessage({ type: '', text: '' });
                    }}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon me-2" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M12 17v4" />
                      <path d="M10 20l4 0" />
                      <path d="M15 7v-2a3 3 0 0 0 -6 0v2" />
                      <path d="M9 11a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" />
                      <path d="M9 11h6" />
                    </svg>
                    Change Password
                  </a>
                  <a 
                    href="#" 
                    className={`list-group-item list-group-item-action d-flex align-items-center ${activeTab === 'avatar' ? 'active' : ''}`}
                    onClick={(e) => {
                      e.preventDefault();
                      setActiveTab('avatar');
                      setMessage({ type: '', text: '' });
                    }}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon me-2" width="24" height="24" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M15 8h.01" />
                      <path d="M3 6a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v12a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3v-12z" />
                      <path d="M3 16l5 -5c.928 -.893 2.072 -.893 3 0l5 5" />
                      <path d="M14 14l1 -1c.928 -.893 2.072 -.893 3 0l3 3" />
                    </svg>
                    Avatar
                  </a>
                </div>
              </div>
            </div>
          </div>
          
          <div className="col-md-9">
            <div className="card">
              <div className="card-body">
                {/* Alert message */}
                {message.text && (
                  <div className={`alert alert-${message.type === 'error' ? 'danger' : 'success'} alert-dismissible`} role="alert">
                    {message.text}
                    <button 
                      type="button" 
                      className="btn-close" 
                      onClick={() => setMessage({ type: '', text: '' })}
                      aria-label="Close"
                    ></button>
                  </div>
                )}
                
                {/* Profile Information Tab */}
                {activeTab === 'profile' && (
                  <form onSubmit={handleProfileSubmit}>
                    <div className="mb-3">
                      <label className="form-label required">Username</label>
                      <input 
                        type="text" 
                        className="form-control" 
                        name="username"
                        value={profileData.username}
                        onChange={handleProfileChange}
                        required
                      />
                      <div className="form-text text-muted">
                        Your username is visible to other users.
                      </div>
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label">Display Name</label>
                      <input 
                        type="text" 
                        className="form-control" 
                        name="name"
                        value={profileData.name}
                        onChange={handleProfileChange}
                      />
                      <div className="form-text text-muted">
                        Optional. Will be displayed instead of your username if provided.
                      </div>
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">Email</label>
                      <input 
                        type="email" 
                        className="form-control" 
                        name="email"
                        value={profileData.email}
                        onChange={handleProfileChange}
                        required
                      />
                      <div className="form-text text-muted">
                        Your email is not visible to other users. Changing your email will require verification.
                      </div>
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label">Bio</label>
                      <textarea 
                        className="form-control" 
                        name="bio"
                        value={profileData.bio}
                        onChange={handleProfileChange}
                        rows="4"
                      ></textarea>
                      <div className="form-text text-muted">
                        A brief description about yourself. This will be displayed on your profile page.
                      </div>
                    </div>
                    
                    <div className="form-footer">
                      <button 
                        type="submit" 
                        className="btn btn-primary"
                        disabled={loading}
                      >
                        {loading ? (
                          <>
                            <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                            Saving...
                          </>
                        ) : 'Save Changes'}
                      </button>
                    </div>
                  </form>
                )}
                
                {/* Password Tab */}
                {activeTab === 'password' && (
                  <form onSubmit={handlePasswordSubmit}>
                    <div className="mb-3">
                      <label className="form-label required">Current Password</label>
                      <input 
                        type="password" 
                        className="form-control" 
                        name="oldPassword"
                        value={passwordData.oldPassword}
                        onChange={handlePasswordChange}
                        required
                      />
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">New Password</label>
                      <input 
                        type="password" 
                        className="form-control" 
                        name="password"
                        value={passwordData.password}
                        onChange={handlePasswordChange}
                        required
                        minLength="8"
                      />
                      <div className="form-text text-muted">
                        Password must be at least 8 characters long.
                      </div>
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">Confirm New Password</label>
                      <input 
                        type="password" 
                        className="form-control" 
                        name="passwordConfirm"
                        value={passwordData.passwordConfirm}
                        onChange={handlePasswordChange}
                        required
                      />
                    </div>
                    
                    <div className="form-footer">
                      <button 
                        type="submit" 
                        className="btn btn-primary"
                        disabled={loading}
                      >
                        {loading ? (
                          <>
                            <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                            Updating...
                          </>
                        ) : 'Update Password'}
                      </button>
                    </div>
                  </form>
                )}
                
                {/* Avatar Tab */}
                {activeTab === 'avatar' && (
                  <div>
                    <div className="text-center mb-4">
                      {user.avatar ? (
                        <img 
                          src={user.avatar} 
                          alt={user.username} 
                          className="avatar avatar-xl"
                        />
                      ) : (
                        <div className="avatar avatar-xl">{user.username.charAt(0).toUpperCase()}</div>
                      )}
                    </div>
                    
                    <div className="alert alert-info" role="alert">
                      <div className="d-flex">
                        <div>
                          <svg xmlns="http://www.w3.org/2000/svg" className="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <path d="M12 9h.01"></path>
                            <path d="M11 12h1v4h1"></path>
                            <path d="M12 3c7.2 0 9 1.8 9 9s-1.8 9 -9 9s-9 -1.8 -9 -9s1.8 -9 9 -9z"></path>
                          </svg>
                        </div>
                        <div>
                          <h4 className="alert-title">Avatar Information</h4>
                          <div className="text-muted">
                            CreateMod.com uses <a href="https://gravatar.com" target="_blank" rel="noopener noreferrer">Gravatar</a> for user avatars. 
                            To change your avatar, please visit <a href="https://gravatar.com" target="_blank" rel="noopener noreferrer">Gravatar.com</a> and 
                            set up an avatar for the email address associated with your account: <strong>{user.email}</strong>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}

/**
 * Server-side data fetching
 */
export async function getServerSideProps(context) {
  try {
    // Validate authentication on the server side
    const { isAuthenticated, user } = await validateServerAuth(context.req);
    
    console.log('[SERVER] Settings page - Auth validation result:', { 
      isAuthenticated, 
      userId: user?.id,
      username: user?.username 
    });
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // If not authenticated, redirect to login page
    if (!isAuthenticated) {
      return {
        redirect: {
          destination: '/login?redirect=/settings',
          permanent: false,
        },
      };
    }
    
    return {
      props: {
        categories,
        isAuthenticated,
        user: user ? JSON.parse(JSON.stringify(user)) : null // Serialize user object for Next.js
      }
    };
  } catch (error) {
    console.error('[SERVER] Error in getServerSideProps for settings page:', error);
    
    return {
      props: {
        categories: [],
        isAuthenticated: false,
        user: null
      }
    };
  }
}