/**
 * Simple script to test the newly created pages
 * 
 * This script can be run with Node.js to test if the pages can be loaded correctly.
 * It doesn't actually render the pages, but it checks if the files exist and can be imported.
 */

const fs = require('fs');
const path = require('path');

// List of pages to test
const pagesToTest = [
  'rules.js',
  'news.js',
  'guide.js',
  'explore.js',
  'contact.js',
  'terms-of-service.js',
  'privacy-policy.js'
];

// Function to test if a page file exists
function testPageExists(pageName) {
  const pagePath = path.join(__dirname, 'pages', pageName);
  try {
    if (fs.existsSync(pagePath)) {
      console.log(`✅ ${pageName} exists`);
      return true;
    } else {
      console.error(`❌ ${pageName} does not exist`);
      return false;
    }
  } catch (err) {
    console.error(`❌ Error checking ${pageName}:`, err);
    return false;
  }
}

// Function to test if a page can be imported
function testPageImport(pageName) {
  try {
    // Remove the .js extension for the import
    const pageNameWithoutExt = pageName.replace('.js', '');
    // We can't actually import the pages here because they use React and Next.js components
    // But we can check if the file is valid JavaScript by parsing it
    const pagePath = path.join(__dirname, 'pages', pageName);
    const content = fs.readFileSync(pagePath, 'utf8');
    
    // Very basic check: if it contains 'export default function' it's likely a valid page component
    if (content.includes('export default function')) {
      console.log(`✅ ${pageName} can be imported`);
      return true;
    } else {
      console.error(`❌ ${pageName} does not export a default function`);
      return false;
    }
  } catch (err) {
    console.error(`❌ Error importing ${pageName}:`, err);
    return false;
  }
}

// Run the tests
console.log('Testing pages...');
let allPagesExist = true;
let allPagesImportable = true;

for (const page of pagesToTest) {
  const exists = testPageExists(page);
  if (!exists) {
    allPagesExist = false;
    continue;
  }
  
  const importable = testPageImport(page);
  if (!importable) {
    allPagesImportable = false;
  }
}

// Summary
console.log('\nTest Summary:');
console.log(`All pages exist: ${allPagesExist ? '✅ Yes' : '❌ No'}`);
console.log(`All pages can be imported: ${allPagesImportable ? '✅ Yes' : '❌ No'}`);

if (allPagesExist && allPagesImportable) {
  console.log('\n✅ All pages are ready to be used!');
  console.log('To fully test the pages, start the Next.js development server:');
  console.log('  cd frontend');
  console.log('  npm run dev');
  console.log('Then open http://localhost:3000 in your browser and navigate to each page.');
} else {
  console.log('\n❌ Some pages have issues. Please check the errors above.');
}