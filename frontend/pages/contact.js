import React, { useState } from 'react';
import Layout from '../components/layout/Layout';
import { getCategories } from '../lib/api';

/**
 * Contact page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data for sidebar
 */
export default function Contact({ categories = [] }) {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: '',
    message: ''
  });
  
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  
  /**
   * Handle input change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
  };
  
  /**
   * Validate form data
   * @returns {boolean} - Whether form is valid
   */
  const validateForm = () => {
    const newErrors = {};
    
    // Required fields
    if (!formData.name.trim()) newErrors.name = 'Name is required';
    if (!formData.email.trim()) newErrors.email = 'Email is required';
    if (!formData.subject.trim()) newErrors.subject = 'Subject is required';
    if (!formData.message.trim()) newErrors.message = 'Message is required';
    
    // Email validation
    if (formData.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Please enter a valid email address';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Handle form submission
   * @param {React.FormEvent} e - Form event
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setIsSubmitting(true);
    
    try {
      // In a real implementation, this would send the form data to an API
      // For now, we'll just simulate a successful submission
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setSubmitSuccess(true);
      
      // Reset form
      setFormData({
        name: '',
        email: '',
        subject: '',
        message: ''
      });
    } catch (error) {
      console.error('Error submitting contact form:', error);
    } finally {
      setIsSubmitting(false);
    }
  };
  
  return (
    <Layout 
      title="Contact Us - CreateMod.com"
      description="Contact the CreateMod.com team with questions, feedback, or support requests"
      categories={categories}
    >
      <div className="container-xl py-4">
        <div className="row">
          {/* Contact Form */}
          <div className="col-lg-8">
            <div className="card">
              <div className="card-header">
                <h2 className="card-title">Contact Us</h2>
              </div>
              <div className="card-body">
                {submitSuccess ? (
                  <div className="alert alert-success" role="alert">
                    <h4 className="alert-title">Message Sent!</h4>
                    <p>Thank you for contacting us. We'll get back to you as soon as possible.</p>
                    <div className="mt-3">
                      <button 
                        className="btn btn-success" 
                        onClick={() => setSubmitSuccess(false)}
                      >
                        Send Another Message
                      </button>
                    </div>
                  </div>
                ) : (
                  <form onSubmit={handleSubmit}>
                    <div className="mb-3">
                      <label className="form-label required">Name</label>
                      <input 
                        type="text" 
                        className={`form-control ${errors.name ? 'is-invalid' : ''}`}
                        name="name"
                        value={formData.name}
                        onChange={handleInputChange}
                        placeholder="Your name"
                        required
                      />
                      {errors.name && <div className="invalid-feedback">{errors.name}</div>}
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">Email</label>
                      <input 
                        type="email" 
                        className={`form-control ${errors.email ? 'is-invalid' : ''}`}
                        name="email"
                        value={formData.email}
                        onChange={handleInputChange}
                        placeholder="your.email@example.com"
                        required
                      />
                      {errors.email && <div className="invalid-feedback">{errors.email}</div>}
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">Subject</label>
                      <input 
                        type="text" 
                        className={`form-control ${errors.subject ? 'is-invalid' : ''}`}
                        name="subject"
                        value={formData.subject}
                        onChange={handleInputChange}
                        placeholder="What is your message about?"
                        required
                      />
                      {errors.subject && <div className="invalid-feedback">{errors.subject}</div>}
                    </div>
                    
                    <div className="mb-3">
                      <label className="form-label required">Message</label>
                      <textarea 
                        className={`form-control ${errors.message ? 'is-invalid' : ''}`}
                        name="message"
                        value={formData.message}
                        onChange={handleInputChange}
                        rows="6"
                        placeholder="Your message..."
                        required
                      ></textarea>
                      {errors.message && <div className="invalid-feedback">{errors.message}</div>}
                    </div>
                    
                    <div className="form-footer">
                      <button 
                        type="submit" 
                        className="btn btn-primary"
                        disabled={isSubmitting}
                      >
                        {isSubmitting ? (
                          <>
                            <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                            Sending...
                          </>
                        ) : 'Send Message'}
                      </button>
                    </div>
                  </form>
                )}
              </div>
            </div>
          </div>
          
          {/* Contact Information */}
          <div className="col-lg-4">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">Contact Information</h3>
              </div>
              <div className="card-body">
                <p className="text-muted mb-4">
                  Have questions, feedback, or need support? We're here to help! Choose the most convenient way to reach us.
                </p>
                
                <div className="mb-3">
                  <div className="d-flex align-items-center mb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-mail me-2" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M3 7a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-10z" />
                      <path d="M3 7l9 6l9 -6" />
                    </svg>
                    <strong>Email</strong>
                  </div>
                  <p className="ms-4">
                    <a href="mailto:support@createmod.com">support@createmod.com</a>
                  </p>
                </div>
                
                <div className="mb-3">
                  <div className="d-flex align-items-center mb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-brand-discord me-2" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M8 12a1 1 0 1 0 2 0a1 1 0 0 0 -2 0" />
                      <path d="M14 12a1 1 0 1 0 2 0a1 1 0 0 0 -2 0" />
                      <path d="M8.5 17c0 1 -1.356 3 -1.832 3c-1.429 0 -2.698 -1.667 -3.333 -3c-.635 -1.667 -.476 -5.833 1.428 -11.5c1.388 -1.015 2.782 -1.34 4.237 -1.5l.975 1.923a11.913 11.913 0 0 1 4.053 0l.972 -1.923c1.5 .16 3.043 .485 4.5 1.5c2 5.667 2.167 9.833 1.5 11.5c-.667 1.333 -2 3 -3.5 3c-.5 0 -2 -2 -2 -3" />
                      <path d="M7 16.5c3.5 1 6.5 1 10 0" />
                    </svg>
                    <strong>Discord</strong>
                  </div>
                  <p className="ms-4">
                    <a href="https://discord.gg/createmod" target="_blank" rel="noopener noreferrer">Join our Discord server</a>
                  </p>
                </div>
                
                <div className="mb-3">
                  <div className="d-flex align-items-center mb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon icon-tabler icon-tabler-brand-github me-2" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" />
                    </svg>
                    <strong>GitHub</strong>
                  </div>
                  <p className="ms-4">
                    <a href="https://github.com/uberswe/createmod" target="_blank" rel="noopener noreferrer">Report issues on GitHub</a>
                  </p>
                </div>
                
                <div className="alert alert-info mt-4">
                  <div className="d-flex">
                    <div>
                      <svg xmlns="http://www.w3.org/2000/svg" className="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M12 9h.01"></path>
                        <path d="M11 12h1v4h1"></path>
                        <path d="M12 3c7.2 0 9 1.8 9 9s-1.8 9 -9 9s-9 -1.8 -9 -9s1.8 -9 9 -9z"></path>
                      </svg>
                    </div>
                    <div>
                      <h4 className="alert-title">Response Time</h4>
                      <p>We typically respond to inquiries within 1-2 business days. For urgent matters, please use Discord for faster assistance.</p>
                    </div>
                  </div>
                </div>
              </div>
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
    console.error('Error fetching data for contact page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: []
      }
    };
  }
}