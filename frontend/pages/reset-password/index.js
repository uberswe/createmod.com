import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { getCategories, requestPasswordReset } from '../../lib/api';

/**
 * Password Reset page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 * @param {boolean} props.isAuthenticated - Whether user is already authenticated
 */
export default function ResetPassword({ categories = [], isAuthenticated = false }) {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);
  
  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      router.push('/');
    }
  }, [isAuthenticated, router]);
  
  /**
   * Handle email input change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleEmailChange = (e) => {
    setEmail(e.target.value);
    
    // Clear error when user types
    if (error) {
      setError('');
    }
  };
  
  /**
   * Validate email
   * @returns {boolean} - Whether email is valid
   */
  const validateEmail = () => {
    if (!email.trim()) {
      setError('Email is required');
      return false;
    }
    
    // Email format validation
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setError('Please enter a valid email address');
      return false;
    }
    
    return true;
  };
  
  /**
   * Handle form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateEmail()) {
      return;
    }
    
    setIsSubmitting(true);
    setError('');
    
    try {
      // Call the API to request password reset
      await requestPasswordReset(email);
      
      // Show success message
      setShowSuccess(true);
      
    } catch (error) {
      console.error('Password reset error:', error);
      setError(error.message || 'An error occurred. Please check your email and try again.');
    } finally {
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
  
  // If password reset request was successful, show success message
  if (showSuccess) {
    return (
      <Layout title="Password Reset Email Sent" description="Check your email to reset your password" categories={categories}>
        <div className="container-tight py-4">
          <div className="card">
            <div className="card-body">
              <div className="text-center mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="icon text-green icon-lg" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                  <path d="M22 7.535v9.465a3 3 0 0 1 -2.824 2.995l-.176 .005h-14a3 3 0 0 1 -2.995 -2.824l-.005 -.176v-9.465l9.445 6.297l.116 .066a1 1 0 0 0 .878 0l.116 -.066l9.445 -6.297z" />
                  <path d="M19 4c1.08 0 2.027 .57 2.555 1.427l-9.555 6.37l-9.555 -6.37a2.999 2.999 0 0 1 2.354 -1.42l.201 -.007h14z" />
                </svg>
                <h2>Check Your Email</h2>
                <p className="text-muted">
                  We've sent a password reset link to <strong>{email}</strong>.
                </p>
                <p className="text-muted">
                  Please check your email and follow the instructions to reset your password. The link will expire in 1 hour.
                </p>
                <p className="text-muted">
                  If you don't see the email, check your spam folder.
                </p>
              </div>
              <div className="text-center">
                <Link href="/login" className="btn btn-primary">
                  Return to Login
                </Link>
              </div>
            </div>
          </div>
        </div>
      </Layout>
    );
  }
  
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
            <h2 className="h2 text-center mb-4">Forgot Password</h2>
            <p className="text-muted text-center mb-4">
              Enter your email address and we'll send you a password reset link.
            </p>
            
            <form onSubmit={handleSubmit} noValidate>
              {error && (
                <div className="alert alert-danger" role="alert">
                  {error}
                </div>
              )}
              
              {/* Email */}
              <div className="mb-3">
                <label className="form-label">Email address</label>
                <input 
                  type="email" 
                  className={`form-control ${error ? 'is-invalid' : ''}`}
                  placeholder="your.email@example.com"
                  value={email}
                  onChange={handleEmailChange}
                  autoComplete="email"
                  required
                />
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
        
        {/* Login Link */}
        <div className="text-center text-muted mt-3">
          Remember your password? <Link href="/login" className="text-decoration-none">Sign in</Link>
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
    console.error('Error fetching data for password reset page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        isAuthenticated: false
      }
    };
  }
}