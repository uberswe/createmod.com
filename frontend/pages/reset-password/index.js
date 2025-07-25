import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getCategories, requestPasswordReset, confirmPasswordReset } from '../../lib/api';

/**
 * Reset Password page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether user is already authenticated
 */
export default function ResetPassword({ categories = [], isAuthenticated = false }) {
  const router = useRouter();
  const { token } = router.query;
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  
  // Redirect if already authenticated
  useEffect(() => {
    console.log('[RESET-PASSWORD] Authentication status changed:', isAuthenticated);
    
    // Check for auth cookie
    if (typeof document !== 'undefined') {
      const cookies = document.cookie.split(';').map(cookie => cookie.trim());
      const authCookie = cookies.find(cookie => cookie.startsWith('create-mod-auth='));
      console.log('[RESET-PASSWORD] Auth cookie present:', !!authCookie);
    }
    
    if (isAuthenticated) {
      console.log('[RESET-PASSWORD] User is already authenticated, redirecting to home page');
      router.push('/');
    } else {
      console.log('[RESET-PASSWORD] User is not authenticated, showing reset password form');
      console.log('[RESET-PASSWORD] Token in URL:', !!token);
    }
  }, [isAuthenticated, router, token]);
  
  /**
   * Validate form data for request password reset
   * @returns {boolean} - Whether form is valid
   */
  const validateRequestForm = () => {
    const newErrors = {};
    
    // Required fields
    if (!email.trim()) newErrors.email = 'Email is required';
    
    // Email validation
    if (email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      newErrors.email = 'Please enter a valid email address';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Validate form data for confirm password reset
   * @returns {boolean} - Whether form is valid
   */
  const validateConfirmForm = () => {
    const newErrors = {};
    
    // Required fields
    if (!password) newErrors.password = 'Password is required';
    if (!passwordConfirm) newErrors.passwordConfirm = 'Please confirm your password';
    
    // Password validation
    if (password && password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }
    
    // Password confirmation
    if (password && passwordConfirm && password !== passwordConfirm) {
      newErrors.passwordConfirm = 'Passwords do not match';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Handle request password reset form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleRequestSubmit = async (e) => {
    e.preventDefault();
    console.log('[RESET-PASSWORD] Password reset request form submitted for email:', email);
    
    if (!validateRequestForm()) {
      console.log('[RESET-PASSWORD] Form validation failed');
      return;
    }
    
    console.log('[RESET-PASSWORD] Form validation passed, proceeding with password reset request');
    setIsSubmitting(true);
    setSubmitError('');
    
    try {
      // Request password reset
      console.log('[RESET-PASSWORD] Calling requestPasswordReset API for email:', email);
      await requestPasswordReset(email);
      
      // Show success message
      console.log('[RESET-PASSWORD] Password reset request successful');
      console.log('[RESET-PASSWORD] Setting showSuccess to true and setting success message');
      setShowSuccess(true);
      setSuccessMessage(
        `Password reset instructions have been sent to ${email}. Please check your email and follow the instructions to reset your password.`
      );
      
    } catch (error) {
      console.error('[RESET-PASSWORD] Password reset request error:', error);
      console.log('[RESET-PASSWORD] Setting error message');
      setSubmitError('An error occurred while requesting password reset. Please try again.');
    } finally {
      console.log('[RESET-PASSWORD] Setting isSubmitting to false');
      setIsSubmitting(false);
    }
  };
  
  /**
   * Handle confirm password reset form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleConfirmSubmit = async (e) => {
    e.preventDefault();
    console.log('[RESET-PASSWORD] Password reset confirmation form submitted with token:', token?.substring(0, 8) + '...');
    
    if (!validateConfirmForm()) {
      console.log('[RESET-PASSWORD] Form validation failed');
      return;
    }
    
    console.log('[RESET-PASSWORD] Form validation passed, proceeding with password reset confirmation');
    setIsSubmitting(true);
    setSubmitError('');
    
    try {
      // Confirm password reset
      console.log('[RESET-PASSWORD] Calling confirmPasswordReset API with token:', token?.substring(0, 8) + '...');
      const response = await confirmPasswordReset(token, password, passwordConfirm);
      
      // Show success message
      console.log('[RESET-PASSWORD] Password reset confirmation successful');
      if (response && response.record) {
        console.log('[RESET-PASSWORD] User data after password reset:', {
          id: response.record.id,
          username: response.record.username,
          email: response.record.email
        });
      }
      
      console.log('[RESET-PASSWORD] Setting showSuccess to true and setting success message');
      setShowSuccess(true);
      setSuccessMessage(
        'Your password has been reset successfully. You can now log in with your new password.'
      );
      
      // Check for auth cookie after password reset
      const cookiesAfterReset = document.cookie.split(';').map(cookie => cookie.trim());
      const authCookieAfterReset = cookiesAfterReset.find(cookie => cookie.startsWith('create-mod-auth='));
      console.log('[RESET-PASSWORD] Auth cookie after password reset:', !!authCookieAfterReset);
      
      // Redirect to login page after a delay
      console.log('[RESET-PASSWORD] Setting timeout for redirect to login page');
      setTimeout(() => {
        console.log('[RESET-PASSWORD] Redirecting to login page');
        router.push('/login/');
      }, 3000);
      
    } catch (error) {
      console.error('[RESET-PASSWORD] Password reset confirmation error:', error);
      
      console.log('[RESET-PASSWORD] Analyzing error message:', error.message);
      if (error.message && error.message.includes('token')) {
        console.log('[RESET-PASSWORD] Invalid or expired token error');
        setSubmitError('Invalid or expired reset token. Please request a new password reset.');
      } else {
        console.log('[RESET-PASSWORD] Generic password reset error');
        setSubmitError('An error occurred while resetting your password. Please try again.');
      }
    } finally {
      console.log('[RESET-PASSWORD] Setting isSubmitting to false');
      setIsSubmitting(false);
    }
  };
  
  // If already authenticated, show loading until redirect happens
  if (isAuthenticated) {
    return (
      <Layout title="Reset Password" description="Reset your password" categories={categories}>
        <div className="d-flex justify-content-center align-items-center" style={{ height: '50vh' }}>
          <div className="spinner-border text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
        </div>
      </Layout>
    );
  }
  
  // If operation was successful, show success message
  if (showSuccess) {
    return (
      <Layout title="Password Reset" description="Reset your password" categories={categories}>
        <div className="container-tight py-4">
          <div className="card">
            <div className="card-body">
              <div className="text-center mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="icon text-green icon-lg" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <circle cx="12" cy="12" r="9" />
                  <path d="M9 12l2 2l4 -4" />
                </svg>
                <h2>Success!</h2>
                <p className="text-muted">
                  {successMessage}
                </p>
              </div>
              <div className="text-center">
                <Link href="/login/" className="btn btn-primary">
                  Go to Login Page
                </Link>
              </div>
            </div>
          </div>
        </div>
      </Layout>
    );
  }
  
  // If token is provided, show confirm password reset form
  if (token) {
    return (
      <Layout title="Reset Password" description="Reset your password" categories={categories}>
        <div className="container-tight py-4">
          <div className="text-center mb-4">
            <Link href="/">
              <img src="/logo.png" height="36" alt="CreateMod.com" />
            </Link>
          </div>
          
          <div className="card card-md">
            <div className="card-body">
              <h2 className="h2 text-center mb-4">Reset your password</h2>
              
              <form onSubmit={handleConfirmSubmit} noValidate>
                {submitError && (
                  <div className="alert alert-danger" role="alert">
                    {submitError}
                  </div>
                )}
                
                {/* New Password */}
                <div className="mb-3">
                  <label className="form-label required">New Password</label>
                  <input 
                    type="password" 
                    className={`form-control ${errors.password ? 'is-invalid' : ''}`}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="New Password"
                    autoComplete="new-password"
                    required
                  />
                  {errors.password && <div className="invalid-feedback">{errors.password}</div>}
                  <div className="form-hint">
                    Password must be at least 8 characters long.
                  </div>
                </div>
                
                {/* Confirm New Password */}
                <div className="mb-3">
                  <label className="form-label required">Confirm New Password</label>
                  <input 
                    type="password" 
                    className={`form-control ${errors.passwordConfirm ? 'is-invalid' : ''}`}
                    value={passwordConfirm}
                    onChange={(e) => setPasswordConfirm(e.target.value)}
                    placeholder="Confirm New Password"
                    autoComplete="new-password"
                    required
                  />
                  {errors.passwordConfirm && <div className="invalid-feedback">{errors.passwordConfirm}</div>}
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
                        Resetting password...
                      </>
                    ) : 'Reset Password'}
                  </button>
                </div>
              </form>
            </div>
          </div>
          
          <div className="text-center text-muted mt-3">
            Remember your password? <Link href="/login/" className="text-decoration-none">Sign in</Link>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Otherwise, show request password reset form
  return (
    <Layout title="Reset Password" description="Reset your password" categories={categories}>
      <div className="container-tight py-4">
        <div className="text-center mb-4">
          <Link href="/">
            <img src="/logo.png" height="36" alt="CreateMod.com" />
          </Link>
        </div>
        
        <div className="card card-md">
          <div className="card-body">
            <h2 className="h2 text-center mb-4">Forgot your password?</h2>
            <p className="text-muted text-center mb-4">
              Enter your email address and we'll send you a password reset link.
            </p>
            
            <form onSubmit={handleRequestSubmit} noValidate>
              {submitError && (
                <div className="alert alert-danger" role="alert">
                  {submitError}
                </div>
              )}
              
              {/* Email */}
              <div className="mb-3">
                <label className="form-label required">Email</label>
                <input 
                  type="email" 
                  className={`form-control ${errors.email ? 'is-invalid' : ''}`}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="your.email@example.com"
                  autoComplete="email"
                  required
                />
                {errors.email && <div className="invalid-feedback">{errors.email}</div>}
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
                      Sending reset link...
                    </>
                  ) : 'Send Reset Link'}
                </button>
              </div>
            </form>
          </div>
        </div>
        
        <div className="text-center text-muted mt-3">
          Remember your password? <Link href="/login/" className="text-decoration-none">Sign in</Link>
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
    
    console.log('Server-side auth check (reset-password) - Cookie exists:', isAuthenticated);
    
    // Validate the auth cookie if it exists
    let validAuth = false;
    if (isAuthenticated) {
      try {
        // In a real implementation, you would validate the token here
        // For now, we'll just assume it's valid if it exists
        validAuth = true;
      } catch (authError) {
        console.error('Auth validation error (reset-password):', authError);
      }
    }
    
    console.log('Server-side auth check (reset-password) - Valid auth:', validAuth);
    
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
    console.error('Error fetching data for reset password page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        isAuthenticated: false
      }
    };
  }
}