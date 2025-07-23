import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getCategories, authenticateUser, authWithOAuth2 } from '../../lib/api';

/**
 * Login page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether user is already authenticated
 */
export default function Login({ categories = [], isAuthenticated = false }) {
  const router = useRouter();
  const [formData, setFormData] = useState({
    identity: '',
    password: ''
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [authError, setAuthError] = useState('');
  
  // Get redirect URL from query params or default to home
  const redirectUrl = router.query.redirect || '/';
  
  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      router.push(redirectUrl);
    }
  }, [isAuthenticated, redirectUrl, router]);
  
  /**
   * Handle input change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // Clear auth error when user types
    if (authError) {
      setAuthError('');
    }
  };
  
  /**
   * Validate form data
   * @returns {boolean} - Whether form is valid
   */
  const validateForm = () => {
    const newErrors = {};
    
    // Required fields
    if (!formData.identity.trim()) newErrors.identity = 'Username or email is required';
    if (!formData.password) newErrors.password = 'Password is required';
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Handle form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setIsSubmitting(true);
    setAuthError('');
    
    try {
      // Use the authenticateUser function from lib/api.js
      const authData = await authenticateUser(
        formData.identity,
        formData.password
      );
      
      // The auth cookie is set by the server in the response
      // No need to manually set it here
      
      // Redirect to the requested page or home
      router.push(redirectUrl);
      
    } catch (error) {
      console.error('Login error:', error);
      setAuthError('Invalid username/email or password');
    } finally {
      setIsSubmitting(false);
    }
  };
  
  /**
   * Handle social login
   * @param {string} provider - Social provider (discord, github)
   */
  const handleSocialLogin = async (provider) => {
    try {
      // Use the authWithOAuth2 function from lib/api.js
      await authWithOAuth2(provider);
      
      // Note: The function above will redirect the user to the OAuth provider
      // and the page will be unloaded, so the code below won't execute
      
    } catch (error) {
      console.error(`${provider} login error:`, error);
      setAuthError(`Failed to authenticate with ${provider}. Please try again.`);
    }
  };
  
  // If already authenticated, show loading until redirect happens
  if (isAuthenticated) {
    return (
      <Layout title="Login" description="Log in to your account" categories={categories}>
        <div className="d-flex justify-content-center align-items-center" style={{ height: '50vh' }}>
          <div className="spinner-border text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
        </div>
      </Layout>
    );
  }
  
  return (
    <Layout title="Login" description="Log in to your account" categories={categories}>
      <div className="container-tight py-4">
        <div className="text-center mb-4">
          <Link href="/">
            <img src="/logo.png" height="36" alt="CreateMod.com" />
          </Link>
        </div>
        
        <div className="card card-md">
          <div className="card-body">
            <h2 className="h2 text-center mb-4">Login to your account</h2>
            
            <form onSubmit={handleSubmit} noValidate>
              {authError && (
                <div className="alert alert-danger" role="alert">
                  {authError}
                </div>
              )}
              
              {/* Username/Email */}
              <div className="mb-3">
                <label className="form-label">Username or Email</label>
                <input 
                  type="text" 
                  className={`form-control ${errors.identity ? 'is-invalid' : ''}`}
                  name="identity"
                  value={formData.identity}
                  onChange={handleInputChange}
                  placeholder="Your username or email"
                  autoComplete="username"
                  required
                />
                {errors.identity && <div className="invalid-feedback">{errors.identity}</div>}
              </div>
              
              {/* Password */}
              <div className="mb-2">
                <label className="form-label">
                  Password
                  <span className="form-label-description">
                    <Link href="/reset-password" className="text-decoration-none">
                      Forgot password?
                    </Link>
                  </span>
                </label>
                <input 
                  type="password" 
                  className={`form-control ${errors.password ? 'is-invalid' : ''}`}
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  placeholder="Your password"
                  autoComplete="current-password"
                  required
                />
                {errors.password && <div className="invalid-feedback">{errors.password}</div>}
              </div>
              
              {/* Remember me */}
              <div className="mb-2">
                <label className="form-check">
                  <input type="checkbox" className="form-check-input" />
                  <span className="form-check-label">Remember me on this device</span>
                </label>
              </div>
              
              {/* Submit Button */}
              <div className="form-footer">
                <button 
                  type="submit" 
                  className="btn btn-primary w-100"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? (
                    <>
                      <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                      Signing in...
                    </>
                  ) : 'Sign in'}
                </button>
              </div>
            </form>
          </div>
        </div>
        
        {/* Social Login */}
        <div className="card card-md mt-3">
          <div className="card-body">
            <h3 className="text-center mb-3">Or login with</h3>
            <div className="d-flex gap-2">
              <button 
                className="btn w-100" 
                onClick={() => handleSocialLogin('discord')}
                style={{ backgroundColor: '#5865F2', color: 'white' }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor" className="icon me-2">
                  <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515a.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0a12.64 12.64 0 0 0-.617-1.25a.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057a19.9 19.9 0 0 0 5.993 3.03a.078.078 0 0 0 .084-.028a14.09 14.09 0 0 0 1.226-1.994a.076.076 0 0 0-.041-.106a13.107 13.107 0 0 1-1.872-.892a.077.077 0 0 1-.008-.128a10.2 10.2 0 0 0 .372-.292a.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127a12.299 12.299 0 0 1-1.873.892a.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028a19.839 19.839 0 0 0 6.002-3.03a.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419c0-1.333.956-2.419 2.157-2.419c1.21 0 2.176 1.096 2.157 2.42c0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419c0-1.333.955-2.419 2.157-2.419c1.21 0 2.176 1.096 2.157 2.42c0 1.333-.946 2.418-2.157 2.418z"/>
                </svg>
                Discord
              </button>
              <button 
                className="btn w-100" 
                onClick={() => handleSocialLogin('github')}
                style={{ backgroundColor: '#24292e', color: 'white' }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor" className="icon me-2">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
                GitHub
              </button>
            </div>
          </div>
        </div>
        
        {/* Register Link */}
        <div className="text-center text-muted mt-3">
          Don't have an account yet? <Link href="/register" className="text-decoration-none">Sign up</Link>
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
    // Check if user is already authenticated
    const isAuthenticated = context.req.cookies['create-mod-auth'] !== undefined;
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    return {
      props: {
        categories,
        isAuthenticated
      }
    };
  } catch (error) {
    console.error('Error fetching data for login page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        isAuthenticated: false
      }
    };
  }
}