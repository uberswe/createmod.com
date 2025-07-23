import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import Layout from '../components/layout/Layout';

/**
 * Auth callback page for handling OAuth redirects
 * This page is the redirect target for OAuth authentication
 */
export default function AuthCallback() {
  const router = useRouter();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Only run this effect when the router is ready and we have query parameters
    if (!router.isReady) return;

    const handleCallback = async () => {
      try {
        // Extract the authorization code and state from the URL
        const { code, error, state } = router.query;

        // If there's an error parameter, show it
        if (error) {
          setError(`Authentication error: ${error}`);
          setLoading(false);
          return;
        }

        // If there's no code, show an error
        if (!code) {
          setError('No authorization code received');
          setLoading(false);
          return;
        }

        // The code is present in the URL, but PocketBase handles the token exchange automatically
        // through cookies. We just need to redirect the user to the home page or the page they
        // were trying to access before authentication.

        // Get the redirect URL from localStorage if it exists
        const redirectUrl = localStorage.getItem('authRedirectUrl') || '/';
        
        // Clear the redirect URL from localStorage
        localStorage.removeItem('authRedirectUrl');
        
        // Redirect to the appropriate page
        router.push(redirectUrl);
      } catch (err) {
        console.error('Error handling OAuth callback:', err);
        setError('An error occurred during authentication. Please try again.');
        setLoading(false);
      }
    };

    handleCallback();
  }, [router.isReady, router.query]);

  return (
    <Layout title="Authentication">
      <div className="container py-5">
        <div className="row justify-content-center">
          <div className="col-md-6">
            <div className="card">
              <div className="card-body text-center">
                {loading ? (
                  <>
                    <h2 className="mb-4">Completing Authentication</h2>
                    <div className="spinner-border text-primary mb-3" role="status">
                      <span className="visually-hidden">Loading...</span>
                    </div>
                    <p className="text-muted">Please wait while we complete the authentication process...</p>
                  </>
                ) : error ? (
                  <>
                    <h2 className="mb-4 text-danger">Authentication Failed</h2>
                    <p className="text-danger">{error}</p>
                    <a href="/login" className="btn btn-primary mt-3">
                      Return to Login
                    </a>
                  </>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}