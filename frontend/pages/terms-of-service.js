import React from 'react';
import Layout from '../components/layout/Layout';
import Link from 'next/link';
import { getCategories } from '../lib/api';

/**
 * Terms of Service page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function TermsOfService({ categories = [] }) {
  return (
    <Layout 
      title="Terms of Service - CreateMod.com"
      description="Terms of Service for CreateMod.com"
      categories={categories}
    >
      <div className="container-xl py-4">
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Terms of Service</h2>
          </div>
          <div className="card-body">
            <div className="markdown">
              <p className="text-muted">
                Last updated: July 23, 2025
              </p>
              
              <h3>1. Acceptance of Terms</h3>
              <p>
                Welcome to CreateMod.com. By accessing or using our website, you agree to be bound by these Terms of Service ("Terms"). 
                If you do not agree to these Terms, please do not use our services.
              </p>
              
              <h3>2. Description of Service</h3>
              <p>
                CreateMod.com is a platform for sharing and discovering schematics for the Create mod in Minecraft. 
                Our services include, but are not limited to:
              </p>
              <ul>
                <li>Browsing and downloading schematics</li>
                <li>Uploading and sharing your own schematics</li>
                <li>Commenting on and rating schematics</li>
                <li>Creating and managing a user profile</li>
              </ul>
              
              <h3>3. User Accounts</h3>
              <p>
                To access certain features of our service, you may need to create an account. You are responsible for:
              </p>
              <ul>
                <li>Maintaining the confidentiality of your account information</li>
                <li>All activities that occur under your account</li>
                <li>Notifying us immediately of any unauthorized use of your account</li>
              </ul>
              <p>
                We reserve the right to terminate accounts that violate our Terms or remain inactive for extended periods.
              </p>
              
              <h3>4. User Content</h3>
              <p>
                By uploading content to CreateMod.com, you:
              </p>
              <ul>
                <li>Retain ownership of your content</li>
                <li>Grant us a non-exclusive, worldwide, royalty-free license to use, display, and distribute your content on our platform</li>
                <li>Represent that you have all necessary rights to the content you upload</li>
                <li>Agree not to upload content that violates our <Link href="/rules">Community Rules</Link></li>
              </ul>
              <p>
                We reserve the right to remove any content that violates these Terms or our Community Rules.
              </p>
              
              <h3>5. Intellectual Property</h3>
              <p>
                CreateMod.com respects intellectual property rights. Our policies regarding intellectual property include:
              </p>
              <ul>
                <li>Users may only upload content they have created or have permission to share</li>
                <li>We will respond to notices of alleged copyright infringement</li>
                <li>The CreateMod.com name, logo, and website design are our trademarks and may not be used without permission</li>
              </ul>
              <p>
                CreateMod.com is not affiliated with Mojang, Microsoft, or the Create mod development team. 
                Minecraft is a trademark of Mojang Synergies AB.
              </p>
              
              <h3>6. Prohibited Activities</h3>
              <p>
                When using our services, you agree not to:
              </p>
              <ul>
                <li>Violate any laws or regulations</li>
                <li>Infringe on the rights of others</li>
                <li>Upload malicious code or attempt to hack our systems</li>
                <li>Impersonate others or misrepresent your affiliation</li>
                <li>Collect user information without consent</li>
                <li>Use our services for commercial purposes without permission</li>
                <li>Engage in any activity that interferes with our services</li>
              </ul>
              
              <h3>7. Disclaimer of Warranties</h3>
              <p>
                Our services are provided "as is" and "as available" without warranties of any kind, either express or implied. 
                We do not guarantee that:
              </p>
              <ul>
                <li>Our services will always be available or error-free</li>
                <li>Defects will be corrected</li>
                <li>Our services are free of viruses or other harmful components</li>
                <li>User content is accurate, complete, or suitable for any purpose</li>
              </ul>
              
              <h3>8. Limitation of Liability</h3>
              <p>
                To the maximum extent permitted by law, CreateMod.com and its operators shall not be liable for:
              </p>
              <ul>
                <li>Any indirect, incidental, special, consequential, or punitive damages</li>
                <li>Loss of profits, data, use, or goodwill</li>
                <li>Cost of procuring substitute services</li>
                <li>Any damages arising from your use of or inability to use our services</li>
              </ul>
              
              <h3>9. Indemnification</h3>
              <p>
                You agree to indemnify, defend, and hold harmless CreateMod.com and its operators from and against any claims, 
                liabilities, damages, losses, and expenses arising from:
              </p>
              <ul>
                <li>Your use of our services</li>
                <li>Your violation of these Terms</li>
                <li>Your violation of any rights of another</li>
                <li>Your user content</li>
              </ul>
              
              <h3>10. Modifications to Terms</h3>
              <p>
                We may modify these Terms at any time. Changes will be effective immediately upon posting to our website. 
                Your continued use of our services after changes are posted constitutes your acceptance of the modified Terms.
              </p>
              
              <h3>11. Governing Law</h3>
              <p>
                These Terms shall be governed by and construed in accordance with the laws of the jurisdiction in which 
                CreateMod.com is operated, without regard to its conflict of law provisions.
              </p>
              
              <h3>12. Termination</h3>
              <p>
                We reserve the right to terminate or suspend your account and access to our services at our sole discretion, 
                without notice, for conduct that we believe violates these Terms or is harmful to other users, us, or third parties, 
                or for any other reason.
              </p>
              
              <h3>13. Contact Information</h3>
              <p>
                If you have any questions about these Terms, please <Link href="/contact">contact us</Link>.
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
    console.error('Error fetching data for terms of service page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: []
      }
    };
  }
}