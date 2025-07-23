# Testing Instructions

This document provides instructions for testing the changes made to fix the API request and stylesheet issues.

## Testing API Connectivity

1. **Start the Backend Server**:
   ```bash
   go run ./cmd/server/main.go serve
   ```

2. **Start the Next.js Development Server**:
   ```bash
   cd frontend
   npm run dev
   ```

3. **Access the API Test Page**:
   - Open your browser and navigate to http://localhost:3000
   - Click on the "API Test Page" button on the home page
   - Alternatively, go directly to http://localhost:3000/api-test

4. **Verify API Requests**:
   - On the API Test page, click the "Test Selected Endpoint" button for different endpoints
   - Open your browser's developer console (F12 or Ctrl+Shift+I)
   - Check the console logs for "API Request URL" messages
   - Verify that all URLs include the `/api/` prefix:
     - Server-side: `http://localhost:8090/api/[endpoint]`
     - Client-side: `/api/[endpoint]`
   - Verify that the responses are valid JSON and display correctly on the page

5. **Test Different Endpoints**:
   - Use the dropdown to select different endpoints to test
   - Verify that each endpoint returns valid data
   - Check for any error messages in the console or on the page

## Testing Stylesheet Loading

1. **Check for Stylesheet Warnings**:
   - Start the Next.js development server as described above
   - Check the terminal output for any stylesheet-related warnings
   - Verify that there are no warnings about using stylesheets in next/head

2. **Verify Stylesheet Loading**:
   - Open your browser's developer tools
   - Go to the Network tab
   - Reload the page
   - Filter for CSS files
   - Verify that all stylesheets are loaded correctly:
     - `/assets/style-XHxDiORf.css`
     - `/libs/star-rating/dist/star-rating.min.css`
     - `/libs/plyr/dist/plyr.css`

3. **Check Page Styling**:
   - Verify that the page is styled correctly
   - Check that all components have the expected appearance
   - Test on different pages to ensure consistent styling

## Testing Error Handling

1. **Test Invalid API Endpoints**:
   - On the API Test page, try testing an invalid endpoint
   - Verify that errors are handled gracefully and displayed on the page
   - Check the console for detailed error messages

2. **Test Authentication Errors**:
   - Try accessing endpoints that require authentication
   - Verify that authentication errors are handled properly
   - Check for appropriate error messages

## Reporting Issues

If you encounter any issues during testing, please report them with the following information:

1. The specific test that failed
2. The expected behavior
3. The actual behavior
4. Any error messages from the console or terminal
5. Screenshots if applicable

This will help us identify and fix any remaining issues quickly.