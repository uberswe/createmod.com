# Subpages Implementation

## Overview

This document describes the implementation of several subpages that were previously not working in the CreateMod.com frontend. The following pages have been created:

1. Rules (`/rules`)
2. News (`/news`)
3. Guide (`/guide`)
4. Explore (`/explore`)
5. Contact (`/contact`)
6. Terms of Service (`/terms-of-service`)
7. Privacy Policy (`/privacy-policy`)

## Implementation Details

### Common Structure

All pages follow a common structure:

1. They use the `Layout` component from `../components/layout/Layout`
2. They fetch categories data for the sidebar using `getCategories` from `../lib/api`
3. They implement server-side data fetching with `getServerSideProps`
4. They include appropriate metadata (title, description)
5. They have a consistent UI structure with cards and sections

### Page-Specific Details

#### Rules Page (`/rules`)

- Displays community rules and guidelines
- Organized into sections covering general rules, content guidelines, schematic quality, communication, account usage, and moderation
- Includes information about rule changes

#### News Page (`/news`)

- Displays news and announcements
- Shows a list of news items with titles, dates, and content
- Includes a fallback UI for when there are no news items
- Has "Read More" links for each news item

#### Guide Page (`/guide`)

- Provides a comprehensive guide for using the Create mod and sharing schematics
- Covers installation, basic concepts, using schematics, creating and sharing schematics, advanced techniques, troubleshooting, and additional resources
- Includes links to relevant pages and external resources

#### Explore Page (`/explore`)

- Showcases featured schematics, creators, and categories
- Includes a hero section with a brief introduction and call-to-action buttons
- Displays schematics using the `SchematicCard` component
- Shows featured creators with their profiles
- Provides a categories showcase for browsing by category

#### Contact Page (`/contact`)

- Provides a contact form for users to send messages
- Includes form validation and error handling
- Shows a success message after form submission
- Displays contact information with email, Discord, and GitHub links

#### Terms of Service Page (`/terms-of-service`)

- Displays the terms of service for the website
- Covers acceptance of terms, description of service, user accounts, user content, intellectual property, prohibited activities, disclaimers, limitations of liability, indemnification, modifications to terms, governing law, termination, and contact information
- Includes links to related pages like rules and contact

#### Privacy Policy Page (`/privacy-policy`)

- Displays the privacy policy for the website
- Covers information collection, use, sharing, security, third-party links, children's privacy, user rights, and policy changes
- Includes links to the contact page for users who have questions or want to exercise their rights

## Testing

All pages have been tested and confirmed to exist and be importable. You can run the included test script to verify this:

```bash
cd frontend
node test-pages.js
```

To fully test the pages in a browser:

1. Start the Next.js development server:
   ```bash
   cd frontend
   npm run dev
   ```

2. Open http://localhost:3000 in your browser

3. Navigate to each page using the sidebar navigation or by directly entering the URLs:
   - http://localhost:3000/rules
   - http://localhost:3000/news
   - http://localhost:3000/guide
   - http://localhost:3000/explore
   - http://localhost:3000/contact
   - http://localhost:3000/terms-of-service
   - http://localhost:3000/privacy-policy

## Future Improvements

While the basic functionality of these pages has been implemented, there are several potential improvements that could be made:

1. **Dynamic Content**: Replace the static content with data fetched from the API
2. **Enhanced Interactivity**: Add more interactive elements to improve user experience
3. **Responsive Design Improvements**: Further optimize the pages for different screen sizes
4. **Accessibility Enhancements**: Ensure all pages meet accessibility standards
5. **Performance Optimization**: Implement code splitting and other performance improvements
6. **SEO Optimization**: Add more metadata and structured data for better search engine visibility

## Conclusion

The implementation of these subpages completes the basic structure of the CreateMod.com frontend. Users can now access important information about the site, browse featured content, contact the site administrators, and read legal documents. These pages enhance the overall user experience and provide necessary functionality for a complete web application.