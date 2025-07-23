import React, { Component } from 'react';
import Link from 'next/link';

/**
 * Error Boundary component to catch JavaScript errors anywhere in the child component tree
 * and display a fallback UI instead of crashing the whole app
 */
class ErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { 
      hasError: false,
      error: null,
      errorInfo: null
    };
  }

  static getDerivedStateFromError(error) {
    // Update state so the next render will show the fallback UI
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    // You can also log the error to an error reporting service
    console.error('Error caught by ErrorBoundary:', error, errorInfo);
    this.setState({ errorInfo });
  }

  render() {
    if (this.state.hasError) {
      // You can render any custom fallback UI
      return (
        <div className="container py-5">
          <div className="card">
            <div className="card-body">
              <div className="text-center mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-alert-triangle mb-2" width="40" height="40" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                  <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                  <path d="M12 9v2m0 4v.01"></path>
                  <path d="M5 19h14a2 2 0 0 0 1.84 -2.75l-7.1 -12.25a2 2 0 0 0 -3.5 0l-7.1 12.25a2 2 0 0 0 1.75 2.75"></path>
                </svg>
                <h2 className="card-title">Something went wrong</h2>
                <p className="text-muted">
                  We're sorry, but an error occurred while rendering this page.
                </p>
              </div>
              
              <div className="alert alert-danger">
                <h4 className="alert-title">Error details</h4>
                <p>{this.state.error && this.state.error.toString()}</p>
                {this.state.errorInfo && (
                  <details className="mt-2">
                    <summary>Component stack trace</summary>
                    <pre className="mt-2 bg-dark text-light p-3 rounded">
                      {this.state.errorInfo.componentStack}
                    </pre>
                  </details>
                )}
              </div>
              
              <div className="d-flex justify-content-center mt-4">
                <Link href="/" className="btn btn-primary me-2">
                  Go to Home Page
                </Link>
                <button 
                  className="btn btn-outline-secondary"
                  onClick={() => window.location.reload()}
                >
                  Reload Page
                </button>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;