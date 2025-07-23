import '../styles/globals.css';
import { useEffect } from 'react';
import ErrorBoundary from '../components/common/ErrorBoundary';

function MyApp({ Component, pageProps }) {
  // Initialize theme from localStorage on client-side
  useEffect(() => {
    const savedTheme = localStorage.getItem('createmodTheme');
    if (savedTheme) {
      document.documentElement.setAttribute('data-bs-theme', savedTheme);
    } else {
      // Default to light theme if no preference is saved
      document.documentElement.setAttribute('data-bs-theme', 'light');
    }
  }, []);

  return (
    <>
      <ErrorBoundary>
        <Component {...pageProps} />
      </ErrorBoundary>
    </>
  );
}

export default MyApp;