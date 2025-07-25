import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getCategories, registerUser, authenticateUser } from '../../lib/api';

/**
 * Register page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether user is already authenticated
 */
export default function Register({ categories = [], isAuthenticated = false }) {
  const router = useRouter();
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    passwordConfirm: '',
    terms: false
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  
  // Redirect if already authenticated
  useEffect(() => {
    console.log('[REGISTER] Authentication status changed:', isAuthenticated);
    
    // Check for auth cookie
    if (typeof document !== 'undefined') {
      const cookies = document.cookie.split(';').map(cookie => cookie.trim());
      const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
      console.log('[REGISTER] Auth cookie present:', !!authCookie);
    }
    
    if (isAuthenticated) {
      console.log('[REGISTER] User is already authenticated, redirecting to home page');
      router.push('/');
    }
  }, [isAuthenticated, router]);
  
  /**
   * Handle input change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (type === 'checkbox') {
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
    
    // Clear error for this field
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // Clear submit error when user types
    if (submitError) {
      setSubmitError('');
    }
  };
  
  /**
   * Validate form data
   * @returns {boolean} - Whether form is valid
   */
  const validateForm = () => {
    const newErrors = {};
    
    // Required fields
    if (!formData.username.trim()) newErrors.username = 'Username is required';
    if (!formData.email.trim()) newErrors.email = 'Email is required';
    if (!formData.password) newErrors.password = 'Password is required';
    if (!formData.passwordConfirm) newErrors.passwordConfirm = 'Please confirm your password';
    if (!formData.terms) newErrors.terms = 'You must agree to the Terms of Service';
    
    // Username validation
    if (formData.username.trim() && !/^[a-zA-Z0-9_]{3,}$/.test(formData.username)) {
      newErrors.username = 'Username must be at least 3 characters and contain only letters, numbers, and underscores';
    }
    
    // Email validation
    if (formData.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Please enter a valid email address';
    }
    
    // Password validation
    if (formData.password && formData.password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }
    
    // Password confirmation
    if (formData.password && formData.passwordConfirm && formData.password !== formData.passwordConfirm) {
      newErrors.passwordConfirm = 'Passwords do not match';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Handle form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    console.log('[REGISTER] Registration form submitted');
    
    if (!validateForm()) {
      console.log('[REGISTER] Form validation failed');
      return;
    }
    
    console.log('[REGISTER] Form validation passed, proceeding with registration');
    setIsSubmitting(true);
    setSubmitError('');
    
    try {
      // Prepare user data for registration
      const userData = {
        username: formData.username,
        email: formData.email,
        password: formData.password,
        passwordConfirm: formData.passwordConfirm,
        emailVisibility: true,
        verified: false
      };
      
      console.log('[REGISTER] Attempting to register user:', {
        username: userData.username,
        email: userData.email,
        emailVisibility: userData.emailVisibility,
        verified: userData.verified
      });
      
      // Register the user
      const registrationResult = await registerUser(userData);
      console.log('[REGISTER] Registration successful:', {
        userId: registrationResult.id,
        username: registrationResult.username,
        email: registrationResult.email,
        created: registrationResult.created
      });
      
      // Show success message
      console.log('[REGISTER] Setting showSuccess to true');
      setShowSuccess(true);
      
      // Check for auth cookie after registration
      const cookiesAfterRegistration = document.cookie.split(';').map(cookie => cookie.trim());
      const authCookieAfterRegistration = cookiesAfterRegistration.find(cookie => cookie.startsWith('create-mod-auth='));
      console.log('[REGISTER] Auth cookie after registration:', !!authCookieAfterRegistration);
      
      // Automatically log in the user after registration
      console.log('[REGISTER] Attempting auto-login after registration');
      try {
        console.log('[REGISTER] Calling authenticateUser with email:', formData.email);
        const authData = await authenticateUser(formData.email, formData.password);
        
        console.log('[REGISTER] Auto-login successful:', {
          userId: authData.record?.id,
          username: authData.record?.username,
          email: authData.record?.email
        });
        
        // Check for auth cookie after auto-login
        const cookiesAfterLogin = document.cookie.split(';').map(cookie => cookie.trim());
        const authCookieAfterLogin = cookiesAfterLogin.find(cookie => cookie.startsWith('create-mod-auth='));
        console.log('[REGISTER] Auth cookie after auto-login:', !!authCookieAfterLogin);
        
        // Redirect to home page after a delay
        console.log('[REGISTER] Setting timeout for redirect to home page');
        setTimeout(() => {
          console.log('[REGISTER] Redirecting to home page');
          router.push('/');
        }, 3000);
      } catch (loginError) {
        console.error('[REGISTER] Auto-login error after registration:', loginError);
        console.log('[REGISTER] Continuing to show success message despite auto-login failure');
        // Even if auto-login fails, we still show success since registration worked
      }
      
    } catch (error) {
      console.error('[REGISTER] Registration error:', error);
      
      // Handle specific error types
      console.log('[REGISTER] Analyzing error message:', error.message);
      if (error.message && error.message.includes('email')) {
        console.log('[REGISTER] Email already registered error');
        setSubmitError('This email is already registered. Please use a different email or try to log in.');
      } else if (error.message && error.message.includes('username')) {
        console.log('[REGISTER] Username already taken error');
        setSubmitError('This username is already taken. Please choose a different username.');
      } else {
        console.log('[REGISTER] Generic registration error');
        setSubmitError('An error occurred during registration. Please try again.');
      }
    } finally {
      console.log('[REGISTER] Setting isSubmitting to false');
      setIsSubmitting(false);
    }
  };
  
  /**
   * Handle social registration
   * @param {string} provider - Social provider (discord, github)
   */
  const handleSocialRegistration = async (provider) => {
    // In a real implementation, you would use PocketBase client to authenticate with OAuth
    // For now, we'll just simulate a redirect
    
    alert(`Redirecting to ${provider} registration... (This is a simulation)`);
    
    // In a real implementation, this would be:
    // await pb.collection('users').authWithOAuth2({ provider });
  };
  
  // If already authenticated, show loading until redirect happens
  if (isAuthenticated) {
    return (
      <Layout title="Register" description="Create a new account" categories={categories}>
        <div className="d-flex justify-content-center align-items-center" style={{ height: '50vh' }}>
          <div className="spinner-border text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
        </div>
      </Layout>
    );
  }
  
  // If registration was successful, show success message
  if (showSuccess) {
    return (
      <Layout title="Registration Successful" description="Your account has been created" categories={categories}>
        <div className="container-tight py-4">
          <div className="card">
            <div className="card-body">
              <div className="text-center mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="icon text-green icon-lg" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <circle cx="12" cy="12" r="9" />
                  <path d="M9 12l2 2l4 -4" />
                </svg>
                <h2>Registration Successful!</h2>
                <p className="text-muted">
                  Your account has been created successfully. You will be redirected to the home page shortly.
                </p>
                <p className="text-muted">
                  A verification email has been sent to your email address. Please check your inbox and follow the instructions to verify your account.
                </p>
              </div>
              <div className="text-center">
                <Link href="/" className="btn btn-primary">
                  Go to Home Page
                </Link>
              </div>
            </div>
          </div>
        </div>
      </Layout>
    );
  }
  
  return (
    <Layout title="Register" description="Create a new account" categories={categories}>
      <div className="container-tight py-4">
        <div className="text-center mb-4">
          <Link href="/">
            <img src="/logo.png" height="36" alt="CreateMod.com" />
          </Link>
        </div>
        
        <div className="card card-md">
          <div className="card-body">
            <h2 className="h2 text-center mb-4">Create new account</h2>
            
            <form onSubmit={handleSubmit} noValidate>
              {submitError && (
                <div className="alert alert-danger" role="alert">
                  {submitError}
                </div>
              )}
              
              {/* Username */}
              <div className="mb-3">
                <label className="form-label required">Username</label>
                <input 
                  type="text" 
                  className={`form-control ${errors.username ? 'is-invalid' : ''}`}
                  name="username"
                  value={formData.username}
                  onChange={handleInputChange}
                  placeholder="Your username"
                  autoComplete="username"
                  required
                />
                {errors.username && <div className="invalid-feedback">{errors.username}</div>}
                <div className="form-hint">
                  Your username will be visible to other users. Use only letters, numbers, and underscores.
                </div>
              </div>
              
              {/* Email */}
              <div className="mb-3">
                <label className="form-label required">Email</label>
                <input 
                  type="email" 
                  className={`form-control ${errors.email ? 'is-invalid' : ''}`}
                  name="email"
                  value={formData.email}
                  onChange={handleInputChange}
                  placeholder="your.email@example.com"
                  autoComplete="email"
                  required
                />
                {errors.email && <div className="invalid-feedback">{errors.email}</div>}
              </div>
              
              {/* Password */}
              <div className="mb-3">
                <label className="form-label required">Password</label>
                <input 
                  type="password" 
                  className={`form-control ${errors.password ? 'is-invalid' : ''}`}
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  placeholder="Password"
                  autoComplete="new-password"
                  required
                />
                {errors.password && <div className="invalid-feedback">{errors.password}</div>}
                <div className="form-hint">
                  Password must be at least 8 characters long.
                </div>
              </div>
              
              {/* Confirm Password */}
              <div className="mb-3">
                <label className="form-label required">Confirm Password</label>
                <input 
                  type="password" 
                  className={`form-control ${errors.passwordConfirm ? 'is-invalid' : ''}`}
                  name="passwordConfirm"
                  value={formData.passwordConfirm}
                  onChange={handleInputChange}
                  placeholder="Confirm Password"
                  autoComplete="new-password"
                  required
                />
                {errors.passwordConfirm && <div className="invalid-feedback">{errors.passwordConfirm}</div>}
              </div>
              
              {/* Terms of Service */}
              <div className="mb-3">
                <label className={`form-check ${errors.terms ? 'is-invalid' : ''}`}>
                  <input 
                    type="checkbox" 
                    className="form-check-input"
                    name="terms"
                    checked={formData.terms}
                    onChange={handleInputChange}
                    required
                  />
                  <span className="form-check-label">
                    I agree to the <Link href="/terms-of-service" target="_blank" className="text-decoration-none">Terms of Service</Link>
                  </span>
                </label>
                {errors.terms && <div className="invalid-feedback">{errors.terms}</div>}
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
                      Creating account...
                    </>
                  ) : 'Create account'}
                </button>
              </div>
            </form>
          </div>
        </div>
        
        {/* Social Registration */}
        <div className="card card-md mt-3">
          <div className="card-body">
            <h3 className="text-center mb-3">Or register with</h3>
            <div className="d-flex gap-2">
              <button 
                className="btn w-100" 
                onClick={() => handleSocialRegistration('discord')}
                style={{ backgroundColor: '#5865F2', color: 'white' }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor" className="icon me-2">
                  <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515a.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0a12.64 12.64 0 0 0-.617-1.25a.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057a19.9 19.9 0 0 0 5.993 3.03a.078.078 0 0 0 .084-.028a14.09 14.09 0 0 0 1.226-1.994a.076.076 0 0 0-.041-.106a13.107 13.107 0 0 1-1.872-.892a.077.077 0 0 1-.008-.128a10.2 10.2 0 0 0 .372-.292a.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127a12.299 12.299 0 0 1-1.873.892a.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028a19.839 19.839 0 0 0 6.002-3.03a.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419c0-1.333.956-2.419 2.157-2.419c1.21 0 2.176 1.096 2.157 2.42c0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419c0-1.333.955-2.419 2.157-2.419c1.21 0 2.176 1.096 2.157 2.42c0 1.333-.946 2.418-2.157 2.418z"/>
                </svg>
                Discord
              </button>
              <button 
                className="btn w-100" 
                onClick={() => handleSocialRegistration('github')}
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
        
        {/* Login Link */}
        <div className="text-center text-muted mt-3">
          Already have an account? <Link href="/login" className="text-decoration-none">Sign in</Link>
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
    const authCookie = context.req.cookies['create-mod-auth'];
    const isAuthenticated = authCookie !== undefined;
    
    console.log('Server-side auth check (register) - Cookie exists:', isAuthenticated);
    
    // Validate the auth cookie if it exists
    let validAuth = false;
    if (isAuthenticated) {
      try {
        // In a real implementation, you would validate the token here
        // For now, we'll just assume it's valid if it exists
        validAuth = true;
      } catch (authError) {
        console.error('Auth validation error (register):', authError);
      }
    }
    
    console.log('Server-side auth check (register) - Valid auth:', validAuth);
    
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    return {
      props: {
        categories,
        isAuthenticated: validAuth
      }
    };
  } catch (error) {
    console.error('Error fetching data for register page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        isAuthenticated: false
      }
    };
  }
}