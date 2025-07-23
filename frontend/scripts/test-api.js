/**
 * Test script to verify communication between Next.js frontend and PocketBase backend
 * 
 * Usage:
 * 1. Make sure the backend server is running at http://localhost:8090
 * 2. Run this script with: node scripts/test-api.js
 */

const fetch = require('node-fetch');

// Configuration
const API_BASE_URL = 'http://localhost:8090/api';
const ENDPOINTS = [
  '/collections/schematics/records',
  '/collections/schematic_categories/records',
  '/collections/users/records',
  '/health'
];

// ANSI color codes for console output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

/**
 * Test a specific API endpoint
 * @param {string} endpoint - API endpoint to test
 * @returns {Promise<boolean>} - Whether the test passed
 */
async function testEndpoint(endpoint) {
  const url = `${API_BASE_URL}${endpoint}`;
  console.log(`${colors.blue}Testing endpoint:${colors.reset} ${url}`);
  
  try {
    const response = await fetch(url);
    
    if (!response.ok) {
      console.log(`${colors.red}✗ Failed with status:${colors.reset} ${response.status} ${response.statusText}`);
      return false;
    }
    
    const data = await response.json();
    console.log(`${colors.green}✓ Success!${colors.reset} Received data with ${Object.keys(data).length} keys`);
    
    // Log a sample of the data
    if (data.items && data.items.length > 0) {
      console.log(`${colors.cyan}Sample data:${colors.reset} ${data.items.length} items found`);
    } else {
      console.log(`${colors.cyan}Response data:${colors.reset}`, JSON.stringify(data).substring(0, 100) + '...');
    }
    
    return true;
  } catch (error) {
    console.log(`${colors.red}✗ Error:${colors.reset} ${error.message}`);
    return false;
  }
}

/**
 * Test URL structure mapping
 */
async function testUrlStructure() {
  console.log(`\n${colors.magenta}Testing URL structure mapping:${colors.reset}`);
  
  const urlMappings = [
    { path: '/schematics', expectedComponent: 'pages/schematics/index.js' },
    { path: '/schematics/example-schematic', expectedComponent: 'pages/schematics/[name].js' },
    { path: '/author/username', expectedComponent: 'pages/author/[username].js' },
    { path: '/search', expectedComponent: 'pages/search/index.js' }
  ];
  
  let allPassed = true;
  
  for (const mapping of urlMappings) {
    console.log(`${colors.blue}Testing URL:${colors.reset} ${mapping.path}`);
    console.log(`${colors.blue}Expected component:${colors.reset} ${mapping.expectedComponent}`);
    
    // Check if the component file exists
    const fs = require('fs');
    const path = require('path');
    
    const componentPath = path.join(__dirname, '..', mapping.expectedComponent);
    const alternativePath = componentPath.replace('.js', '.jsx');
    
    if (fs.existsSync(componentPath) || fs.existsSync(alternativePath)) {
      console.log(`${colors.green}✓ Component exists!${colors.reset}`);
    } else {
      console.log(`${colors.red}✗ Component not found!${colors.reset}`);
      allPassed = false;
    }
  }
  
  return allPassed;
}

/**
 * Main test function
 */
async function runTests() {
  console.log(`${colors.magenta}Starting API communication tests${colors.reset}`);
  console.log(`${colors.blue}Backend API URL:${colors.reset} ${API_BASE_URL}\n`);
  
  // Test backend connectivity
  let allPassed = true;
  
  // Test each endpoint
  for (const endpoint of ENDPOINTS) {
    const passed = await testEndpoint(endpoint);
    allPassed = allPassed && passed;
    console.log(); // Add a blank line between tests
  }
  
  // Test URL structure
  const urlStructurePassed = await testUrlStructure();
  allPassed = allPassed && urlStructurePassed;
  
  // Summary
  console.log(`\n${colors.magenta}Test Summary:${colors.reset}`);
  if (allPassed) {
    console.log(`${colors.green}✓ All tests passed!${colors.reset}`);
    console.log(`${colors.green}✓ The Next.js frontend can communicate with the PocketBase backend${colors.reset}`);
    console.log(`${colors.green}✓ URL structure is correctly mapped${colors.reset}`);
  } else {
    console.log(`${colors.red}✗ Some tests failed!${colors.reset}`);
    console.log(`${colors.yellow}Please check the output above for details${colors.reset}`);
  }
}

// Run the tests
runTests().catch(error => {
  console.error(`${colors.red}Unhandled error:${colors.reset}`, error);
  process.exit(1);
});