import React from 'react';
import Layout from '../components/layout/Layout';
import Link from 'next/link';
import { getCategories } from '../lib/api';

/**
 * Privacy Policy page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function PrivacyPolicy({ categories = [] }) {
  return (
    <Layout 
      title="Privacy Policy - CreateMod.com"
      description="Privacy Policy for CreateMod.com"
      categories={categories}
    >
      <div className="container-xl py-4">
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Privacy Policy</h2>
          </div>
          <div className="card-body">
            <div className="markdown">
              <p className="text-muted">
                Last updated: July 23, 2025
              </p>
              
              <h3>1. Introduction</h3>
              <p>
                At CreateMod.com, we respect your privacy and are committed to protecting your personal data. 
                This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you visit our website.
              </p>
              <p>
                Please read this Privacy Policy carefully. If you do not agree with the terms of this Privacy Policy, 
                please do not access the site.
              </p>
              
              <h3>2. Information We Collect</h3>
              
              <h4>2.1 Personal Data</h4>
              <p>
                We may collect personal identification information from you in various ways, including, but not limited to:
              </p>
              <ul>
                <li>When you register on our site</li>
                <li>When you upload content</li>
                <li>When you fill out a form</li>
                <li>When you leave comments</li>
                <li>When you contact us</li>
              </ul>
              <p>
                The personal information we may collect includes:
              </p>
              <ul>
                <li>Name</li>
                <li>Email address</li>
                <li>Username</li>
                <li>Profile information</li>
              </ul>
              
              <h4>2.2 Non-Personal Data</h4>
              <p>
                We may also collect non-personal identification information about users whenever they interact with our site. 
                This may include:
              </p>
              <ul>
                <li>Browser name</li>
                <li>Type of computer or device</li>
                <li>Technical information about the user's connection to our site</li>
                <li>IP address</li>
                <li>Referring site</li>
                <li>Pages visited</li>
              </ul>
              
              <h4>2.3 Cookies and Tracking Technologies</h4>
              <p>
                Our website may use "cookies" to enhance user experience. Cookies are small files that a site transfers to your 
                computer's hard drive through your web browser. They enable the site to recognize your browser and capture and 
                remember certain information.
              </p>
              <p>
                We use cookies to:
              </p>
              <ul>
                <li>Understand and save user preferences</li>
                <li>Keep track of advertisements</li>
                <li>Compile aggregate data about site traffic and interactions</li>
                <li>Enhance and personalize your experience on our site</li>
              </ul>
              <p>
                You can choose to have your computer warn you each time a cookie is being sent, or you can choose to turn off all cookies 
                through your browser settings. Since each browser is different, look at your browser's Help Menu to learn the correct way 
                to modify your cookies.
              </p>
              
              <h3>3. How We Use Your Information</h3>
              <p>
                We may use the information we collect from you for the following purposes:
              </p>
              <ul>
                <li>To personalize your experience and deliver content most relevant to you</li>
                <li>To improve our website based on the feedback and information we receive from you</li>
                <li>To process transactions and manage your account</li>
                <li>To administer contests, promotions, surveys, or other site features</li>
                <li>To send periodic emails regarding your account or other products and services</li>
                <li>To respond to your inquiries and provide customer support</li>
                <li>To protect our rights and the rights of others</li>
              </ul>
              
              <h3>4. Information Sharing and Disclosure</h3>
              <p>
                We do not sell, trade, or rent users' personal identification information to others. We may share generic 
                aggregated demographic information not linked to any personal identification information regarding visitors 
                and users with our business partners, trusted affiliates, and advertisers.
              </p>
              <p>
                We may disclose your information in the following circumstances:
              </p>
              <ul>
                <li>To comply with legal obligations</li>
                <li>To protect and defend our rights and property</li>
                <li>To prevent or investigate possible wrongdoing in connection with the service</li>
                <li>To protect the personal safety of users of the service or the public</li>
                <li>To protect against legal liability</li>
              </ul>
              
              <h3>5. Data Security</h3>
              <p>
                We implement appropriate data collection, storage, processing practices, and security measures to protect against 
                unauthorized access, alteration, disclosure, or destruction of your personal information, username, password, 
                transaction information, and data stored on our site.
              </p>
              <p>
                However, please be aware that no method of transmission over the Internet or method of electronic storage is 100% 
                secure and we cannot guarantee the absolute security of your data.
              </p>
              
              <h3>6. Third-Party Links</h3>
              <p>
                Our website may contain links to third-party websites. We have no control over the content, privacy policies, 
                or practices of any third-party sites or services. We encourage you to review the privacy policies of any sites 
                you visit.
              </p>
              
              <h3>7. Children's Privacy</h3>
              <p>
                Our service is not directed to anyone under the age of 13. We do not knowingly collect personal information from 
                children under 13. If you are a parent or guardian and you are aware that your child has provided us with personal 
                information, please contact us. If we discover that a child under 13 has provided us with personal information, 
                we will delete such information from our servers.
              </p>
              
              <h3>8. Your Rights</h3>
              <p>
                Depending on your location, you may have certain rights regarding your personal information, including:
              </p>
              <ul>
                <li>The right to access the personal information we have about you</li>
                <li>The right to request correction of inaccurate personal information</li>
                <li>The right to request deletion of your personal information</li>
                <li>The right to object to processing of your personal information</li>
                <li>The right to data portability</li>
                <li>The right to withdraw consent</li>
              </ul>
              <p>
                To exercise these rights, please <Link href="/contact">contact us</Link>.
              </p>
              
              <h3>9. Changes to This Privacy Policy</h3>
              <p>
                We may update our Privacy Policy from time to time. We will notify you of any changes by posting the new 
                Privacy Policy on this page and updating the "Last updated" date at the top of this page. You are advised 
                to review this Privacy Policy periodically for any changes.
              </p>
              
              <h3>10. Contact Us</h3>
              <p>
                If you have any questions about this Privacy Policy, please <Link href="/contact">contact us</Link>.
              </p>
            </div>
          </div>
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
    // Get categories for sidebar
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    return {
      props: {
        categories
      }
    };
  } catch (error) {
    console.error('Error fetching data for privacy policy page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: []
      }
    };
  }
}