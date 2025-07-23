import { useState, useEffect } from 'react';
import { fetchAPI } from '../lib/api';
import Layout from '../components/layout/Layout';

// List of endpoints to test
const TEST_ENDPOINTS = [
  { name: 'Health Check', endpoint: '/health' },
  { name: 'Categories', endpoint: '/collections/schematic_categories/records?sort=name' },
  { name: 'Schematics (Limited)', endpoint: '/collections/schematics/records?page=1&perPage=5' },
  { name: 'Public Collections', endpoint: '/collections' }
];

export default function ApiTest({ initialData = {}, initialError = null }) {
  const [loading, setLoading] = useState({});
  const [errors, setErrors] = useState({});
  const [results, setResults] = useState({});
  const [selectedEndpoint, setSelectedEndpoint] = useState(TEST_ENDPOINTS[0].endpoint);

  // Test client-side data fetching
  const fetchEndpoint = async (endpoint) => {
    setLoading(prev => ({ ...prev, [endpoint]: true }));
    setErrors(prev => ({ ...prev, [endpoint]: null }));
    
    try {
      const data = await fetchAPI(endpoint);
      setResults(prev => ({ ...prev, [endpoint]: data }));
      console.log(`Client-side data fetched successfully for ${endpoint}:`, data);
    } catch (err) {
      setErrors(prev => ({ ...prev, [endpoint]: err.message || 'An error occurred while fetching data' }));
      console.error(`Client-side fetch error for ${endpoint}:`, err);
    } finally {
      setLoading(prev => ({ ...prev, [endpoint]: false }));
    }
  };

  return (
    <Layout title="API Test Page">
      <div className="container py-4">
        <h1 className="mb-4">API Connectivity Test</h1>
        
        {/* Server-side fetched data */}
        <div className="card mb-4">
          <div className="card-header">
            <h2 className="card-title h5 mb-0">Server-Side Data Fetching</h2>
          </div>
          <div className="card-body">
            {initialError ? (
              <div className="alert alert-danger">
                <strong>Server-side Error:</strong> {initialError}
              </div>
            ) : Object.keys(initialData).length > 0 ? (
              <>
                <p className="text-success">✅ Successfully fetched data from the server during SSR!</p>
                <div className="mt-3">
                  <h3 className="h6">Raw Response:</h3>
                  <pre className="bg-light p-3 rounded">
                    {JSON.stringify(initialData, null, 2)}
                  </pre>
                </div>
              </>
            ) : (
              <p className="text-danger">❌ No data was fetched during server-side rendering.</p>
            )}
          </div>
        </div>
        
        {/* Client-side fetched data */}
        <div className="card mb-4">
          <div className="card-header">
            <h2 className="card-title h5 mb-0">Client-Side Data Fetching</h2>
          </div>
          <div className="card-body">
            <div className="mb-3">
              <label className="form-label">Select API Endpoint to Test:</label>
              <select 
                className="form-select mb-3"
                value={selectedEndpoint}
                onChange={(e) => setSelectedEndpoint(e.target.value)}
              >
                {TEST_ENDPOINTS.map((endpoint) => (
                  <option key={endpoint.endpoint} value={endpoint.endpoint}>
                    {endpoint.name} ({endpoint.endpoint})
                  </option>
                ))}
              </select>
              
              <button 
                className="btn btn-primary" 
                onClick={() => fetchEndpoint(selectedEndpoint)}
                disabled={loading[selectedEndpoint]}
              >
                {loading[selectedEndpoint] ? 'Loading...' : 'Test Selected Endpoint'}
              </button>
            </div>
            
            <div className="mt-4">
              <h3 className="h6 mb-3">Test Results:</h3>
              
              {TEST_ENDPOINTS.map((endpoint) => (
                <div key={endpoint.endpoint} className="mb-4">
                  <h4 className="h6 d-flex align-items-center">
                    <span className="badge bg-secondary me-2">{endpoint.name}</span>
                    <code className="small">{endpoint.endpoint}</code>
                    
                    <button 
                      className="btn btn-sm btn-outline-primary ms-auto"
                      onClick={() => fetchEndpoint(endpoint.endpoint)}
                      disabled={loading[endpoint.endpoint]}
                    >
                      {loading[endpoint.endpoint] ? 'Testing...' : 'Test'}
                    </button>
                  </h4>
                  
                  {errors[endpoint.endpoint] && (
                    <div className="alert alert-danger mt-2">
                      <strong>Error:</strong> {errors[endpoint.endpoint]}
                    </div>
                  )}
                  
                  {results[endpoint.endpoint] && (
                    <div className="mt-2">
                      <p className="text-success">✅ Success!</p>
                      <pre className="bg-light p-2 rounded small" style={{ maxHeight: '200px', overflow: 'auto' }}>
                        {JSON.stringify(results[endpoint.endpoint], null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
        
        {/* URL Construction Debug */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title h5 mb-0">URL Construction Debug</h2>
          </div>
          <div className="card-body">
            <p>Check the browser console for API Request URL logs.</p>
            <p>Expected URL format: <code>/api/[endpoint]</code> (client-side)</p>
            <p>Expected URL format: <code>http://localhost:8090/api/[endpoint]</code> (server-side)</p>
            
            <div className="alert alert-info mt-3">
              <h4 className="alert-heading h6">Authentication Note</h4>
              <p className="mb-0">
                Some endpoints may require authentication. If you see "Only superusers can perform this action" errors,
                it means the endpoint requires authentication with admin privileges.
              </p>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}

export async function getServerSideProps() {
  // Try to fetch data from the health endpoint first, as it's likely to be publicly accessible
  const testEndpoint = '/health';
  
  try {
    console.log(`Testing server-side API connectivity with endpoint: ${testEndpoint}`);
    const data = await fetchAPI(testEndpoint);
    console.log('Server-side data fetched successfully:', data);
    
    return {
      props: {
        initialData: data || {},
        initialError: null,
      },
    };
  } catch (error) {
    console.error('Server-side fetch error:', error);
    
    return {
      props: {
        initialData: {},
        initialError: error.message || 'An error occurred during server-side data fetching',
      },
    };
  }
}