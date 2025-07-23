import React from 'react';
import Layout from '../components/layout/Layout';
import { getCategories } from '../lib/api';

/**
 * Rules page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function Rules({ categories = [] }) {
  return (
    <Layout 
      title="Rules - CreateMod.com"
      description="Community rules and guidelines for CreateMod.com"
      categories={categories}
    >
      <div className="container-xl py-4">
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Community Rules</h2>
          </div>
          <div className="card-body">
            <div className="markdown">
              <h3>General Rules</h3>
              <p>
                Welcome to CreateMod.com! To ensure a positive experience for everyone, please follow these rules when using our platform.
              </p>
              
              <h4>1. Respect Other Users</h4>
              <p>
                Treat all members of our community with respect. Harassment, hate speech, discrimination, or bullying will not be tolerated.
              </p>
              
              <h4>2. Content Guidelines</h4>
              <p>
                All schematics and content uploaded to CreateMod.com must:
              </p>
              <ul>
                <li>Be your own creation or have permission from the original creator</li>
                <li>Not contain inappropriate, offensive, or adult content</li>
                <li>Not violate any copyright, trademark, or intellectual property laws</li>
                <li>Include proper attribution for any collaborative work</li>
              </ul>
              
              <h4>3. Schematic Quality</h4>
              <p>
                When uploading schematics, please ensure:
              </p>
              <ul>
                <li>Your schematic works as described</li>
                <li>You provide clear descriptions and instructions</li>
                <li>You specify the Minecraft and Create mod versions required</li>
                <li>You list any dependencies or additional mods needed</li>
              </ul>
              
              <h4>4. Communication</h4>
              <p>
                When commenting or messaging:
              </p>
              <ul>
                <li>Stay on topic and be constructive</li>
                <li>Avoid spamming or excessive self-promotion</li>
                <li>Use appropriate language</li>
                <li>Report issues rather than engaging in arguments</li>
              </ul>
              
              <h4>5. Account Usage</h4>
              <p>
                Each user should:
              </p>
              <ul>
                <li>Maintain only one account</li>
                <li>Not share account credentials</li>
                <li>Not impersonate other users or staff</li>
                <li>Not attempt to bypass any site restrictions</li>
              </ul>
              
              <h3>Moderation</h3>
              <p>
                Our moderation team works to ensure these rules are followed. Violations may result in:
              </p>
              <ul>
                <li>Content removal</li>
                <li>Temporary restrictions</li>
                <li>Account suspension or termination</li>
              </ul>
              
              <p>
                We strive to be fair and consistent in our moderation. If you believe a mistake has been made, please contact us through the appropriate channels.
              </p>
              
              <h3>Changes to Rules</h3>
              <p>
                These rules may be updated periodically. Significant changes will be announced, but it's your responsibility to stay informed about the current rules.
              </p>
              
              <p>
                Thank you for being part of our community and helping to make CreateMod.com a positive and creative space for everyone!
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
    console.error('Error fetching data for rules page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: []
      }
    };
  }
}