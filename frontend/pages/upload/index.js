import React, { useState, useEffect } from 'react';
import Layout from '../../components/layout/Layout';
import { getCategories, getTags, getMinecraftVersions, getCreateModVersions, createRecord, uploadFile } from '../../lib/api';
import { useRouter } from 'next/router';
import { validateServerAuth } from '../../lib/auth';

/**
 * Upload page component
 * 
 * @param {Object} props - Component props from getServerSideProps
 * @param {Array} props.categories - Categories data
 * @param {Array} props.tags - Tags data
 * @param {Array} props.minecraftVersions - Minecraft versions data
 * @param {Array} props.createmodVersions - Create mod versions data
 * @param {boolean} props.isAuthenticated - Whether user is authenticated
 * @param {Object} props.user - User data if authenticated
 */
export default function Upload({ 
  categories = [], 
  tags = [], 
  minecraftVersions = [], 
  createmodVersions = [],
  isAuthenticated = false,
  user = null
}) {
  const router = useRouter();
  const [formData, setFormData] = useState({
    title: '',
    name: '',
    content: '',
    minecraft_version: '',
    createmod_version: '',
    categories: [],
    tags: [],
    has_dependencies: false,
    dependencies: '',
    video: ''
  });
  
  const [files, setFiles] = useState({
    featured_image: null,
    gallery: [],
    schematic_file: null
  });
  
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const [submitError, setSubmitError] = useState('');
  
  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?redirect=/upload');
    }
  }, [isAuthenticated, router]);
  
  /**
   * Handle input change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (type === 'checkbox') {
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else if (name === 'categories' || name === 'tags') {
      // Handle multi-select
      const options = Array.from(e.target.selectedOptions).map(option => option.value);
      setFormData(prev => ({ ...prev, [name]: options }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
      
      // Generate slug from title
      if (name === 'title') {
        const slug = value
          .toLowerCase()
          .replace(/[^a-z0-9 -]/g, '')
          .replace(/\s+/g, '-')
          .replace(/-+/g, '-');
        
        setFormData(prev => ({ ...prev, name: slug }));
      }
    }
    
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
   * Handle file change
   * @param {React.ChangeEvent} e - Change event
   */
  const handleFileChange = (e) => {
    const { name, files: fileList } = e.target;
    
    if (name === 'gallery') {
      // Handle multiple files
      const galleryFiles = Array.from(fileList);
      setFiles(prev => ({ ...prev, [name]: galleryFiles }));
    } else {
      // Handle single file
      setFiles(prev => ({ ...prev, [name]: fileList[0] }));
    }
    
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
    if (!formData.title.trim()) newErrors.title = 'Title is required';
    if (!formData.name.trim()) newErrors.name = 'Name is required';
    if (!formData.content.trim()) newErrors.content = 'Description is required';
    if (!formData.minecraft_version) newErrors.minecraft_version = 'Minecraft version is required';
    if (!formData.createmod_version) newErrors.createmod_version = 'Create mod version is required';
    if (formData.categories.length === 0) newErrors.categories = 'At least one category is required';
    
    // Required files
    if (!files.featured_image) newErrors.featured_image = 'Featured image is required';
    if (!files.schematic_file) newErrors.schematic_file = 'Schematic file is required';
    
    // File type validation
    if (files.featured_image && !files.featured_image.type.startsWith('image/')) {
      newErrors.featured_image = 'Featured image must be an image file';
    }
    
    if (files.gallery.length > 0) {
      const invalidGalleryFiles = files.gallery.filter(file => !file.type.startsWith('image/'));
      if (invalidGalleryFiles.length > 0) {
        newErrors.gallery = 'All gallery files must be images';
      }
    }
    
    if (files.schematic_file && !files.schematic_file.name.endsWith('.nbt')) {
      newErrors.schematic_file = 'Schematic file must be a .nbt file';
    }
    
    // URL validation
    if (formData.video && !isValidYouTubeUrl(formData.video)) {
      newErrors.video = 'Please enter a valid YouTube video URL or ID';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  /**
   * Validate YouTube URL
   * @param {string} url - URL to validate
   * @returns {boolean} - Whether URL is valid
   */
  const isValidYouTubeUrl = (url) => {
    if (!url) return true; // Empty is valid (optional field)
    
    // Extract video ID from URL or use as is if it's just an ID
    const videoIdRegex = /^[a-zA-Z0-9_-]{11}$/;
    const urlRegex = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})$/;
    
    return videoIdRegex.test(url) || urlRegex.test(url);
  };
  
  /**
   * Extract YouTube video ID from URL
   * @param {string} url - YouTube URL
   * @returns {string} - Video ID
   */
  const extractYouTubeId = (url) => {
    if (!url) return '';
    
    // If it's already just an ID
    if (/^[a-zA-Z0-9_-]{11}$/.test(url)) {
      return url;
    }
    
    // Extract from URL
    const match = url.match(/^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/);
    return match ? match[4] : '';
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
    setSubmitError('');
    
    try {
      // Process YouTube URL to extract ID
      const videoId = formData.video ? extractYouTubeId(formData.video) : '';
      
      // Create initial schematic record without files
      const schematicData = {
        title: formData.title,
        name: formData.name,
        content: formData.content,
        minecraft_version: formData.minecraft_version,
        createmod_version: formData.createmod_version,
        categories: formData.categories,
        tags: formData.tags,
        has_dependencies: formData.has_dependencies,
        dependencies: formData.dependencies,
        video: videoId,
        // Set author to current user (handled by the server)
        moderated: false, // Requires moderation before being public
        deleted: null
      };
      
      // Create the schematic record
      const newSchematic = await createRecord('schematics', schematicData);
      
      // Upload featured image
      await uploadFile('schematics', newSchematic.id, 'featured_image', files.featured_image);
      
      // Upload schematic file
      await uploadFile('schematics', newSchematic.id, 'schematic_file', files.schematic_file);
      
      // Upload gallery images (if any)
      if (files.gallery.length > 0) {
        for (const galleryFile of files.gallery) {
          await uploadFile('schematics', newSchematic.id, 'gallery', galleryFile);
        }
      }
      
      setSubmitSuccess(true);
      
      // Redirect to the new schematic page after a delay
      setTimeout(() => {
        router.push(`/schematics/${formData.name}`);
      }, 2000);
      
    } catch (error) {
      console.error('Error uploading schematic:', error);
      setSubmitError(error.message || 'An error occurred while uploading your schematic. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };
  
  // If not authenticated, show loading until redirect happens
  if (!isAuthenticated) {
    return (
      <Layout title="Upload Schematic" description="Upload a new schematic" categories={categories}>
        <div className="d-flex justify-content-center align-items-center" style={{ height: '50vh' }}>
          <div className="spinner-border text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
        </div>
      </Layout>
    );
  }
  
  return (
    <Layout 
      title="Upload Schematic" 
      description="Upload a new schematic for the Create mod"
      categories={categories}
      isAuthenticated={isAuthenticated}
      user={user}
    >
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-header">
              <h3 className="card-title">Upload Schematic</h3>
            </div>
            <div className="card-body">
              {submitSuccess ? (
                <div className="alert alert-success" role="alert">
                  <h4 className="alert-title">Success!</h4>
                  <div className="text-muted">Your schematic has been uploaded successfully. Redirecting to your schematic page...</div>
                </div>
              ) : (
                <form onSubmit={handleSubmit}>
                  {submitError && (
                    <div className="alert alert-danger" role="alert">
                      <h4 className="alert-title">Error</h4>
                      <div className="text-muted">{submitError}</div>
                    </div>
                  )}
                  
                  {/* Title */}
                  <div className="mb-3">
                    <label className="form-label required">Title</label>
                    <input 
                      type="text" 
                      className={`form-control ${errors.title ? 'is-invalid' : ''}`}
                      name="title"
                      value={formData.title}
                      onChange={handleInputChange}
                      placeholder="Enter a title for your schematic"
                      required
                    />
                    {errors.title && <div className="invalid-feedback">{errors.title}</div>}
                  </div>
                  
                  {/* Name (slug) */}
                  <div className="mb-3">
                    <label className="form-label required">URL Name</label>
                    <div className="input-group">
                      <span className="input-group-text">createmod.com/schematics/</span>
                      <input 
                        type="text" 
                        className={`form-control ${errors.name ? 'is-invalid' : ''}`}
                        name="name"
                        value={formData.name}
                        onChange={handleInputChange}
                        placeholder="url-friendly-name"
                        required
                      />
                    </div>
                    {errors.name && <div className="invalid-feedback">{errors.name}</div>}
                    <small className="form-hint">This will be used in the URL. Use only letters, numbers, and hyphens.</small>
                  </div>
                  
                  {/* Description */}
                  <div className="mb-3">
                    <label className="form-label required">Description</label>
                    <textarea 
                      className={`form-control ${errors.content ? 'is-invalid' : ''}`}
                      name="content"
                      value={formData.content}
                      onChange={handleInputChange}
                      rows="6"
                      placeholder="Describe your schematic in detail"
                      required
                    ></textarea>
                    {errors.content && <div className="invalid-feedback">{errors.content}</div>}
                    <small className="form-hint">HTML is allowed for formatting.</small>
                  </div>
                  
                  {/* Featured Image */}
                  <div className="mb-3">
                    <label className="form-label required">Featured Image</label>
                    <input 
                      type="file" 
                      className={`form-control ${errors.featured_image ? 'is-invalid' : ''}`}
                      name="featured_image"
                      onChange={handleFileChange}
                      accept="image/*"
                      required
                    />
                    {errors.featured_image && <div className="invalid-feedback">{errors.featured_image}</div>}
                    <small className="form-hint">This will be the main image shown for your schematic.</small>
                  </div>
                  
                  {/* Gallery */}
                  <div className="mb-3">
                    <label className="form-label">Gallery Images</label>
                    <input 
                      type="file" 
                      className={`form-control ${errors.gallery ? 'is-invalid' : ''}`}
                      name="gallery"
                      onChange={handleFileChange}
                      accept="image/*"
                      multiple
                    />
                    {errors.gallery && <div className="invalid-feedback">{errors.gallery}</div>}
                    <small className="form-hint">Optional additional images for your schematic gallery.</small>
                  </div>
                  
                  {/* Schematic File */}
                  <div className="mb-3">
                    <label className="form-label required">Schematic File (.nbt)</label>
                    <input 
                      type="file" 
                      className={`form-control ${errors.schematic_file ? 'is-invalid' : ''}`}
                      name="schematic_file"
                      onChange={handleFileChange}
                      accept=".nbt"
                      required
                    />
                    {errors.schematic_file && <div className="invalid-feedback">{errors.schematic_file}</div>}
                    <small className="form-hint">The .nbt file exported from the Create mod.</small>
                  </div>
                  
                  <div className="row">
                    {/* Minecraft Version */}
                    <div className="col-md-6 mb-3">
                      <label className="form-label required">Minecraft Version</label>
                      <select 
                        className={`form-select ${errors.minecraft_version ? 'is-invalid' : ''}`}
                        name="minecraft_version"
                        value={formData.minecraft_version}
                        onChange={handleInputChange}
                        required
                      >
                        <option value="">Select Minecraft Version</option>
                        {minecraftVersions.map(version => (
                          <option key={version.id} value={version.id}>
                            {version.version}
                          </option>
                        ))}
                      </select>
                      {errors.minecraft_version && <div className="invalid-feedback">{errors.minecraft_version}</div>}
                    </div>
                    
                    {/* Create Mod Version */}
                    <div className="col-md-6 mb-3">
                      <label className="form-label required">Create Mod Version</label>
                      <select 
                        className={`form-select ${errors.createmod_version ? 'is-invalid' : ''}`}
                        name="createmod_version"
                        value={formData.createmod_version}
                        onChange={handleInputChange}
                        required
                      >
                        <option value="">Select Create Mod Version</option>
                        {createmodVersions.map(version => (
                          <option key={version.id} value={version.id}>
                            {version.version}
                          </option>
                        ))}
                      </select>
                      {errors.createmod_version && <div className="invalid-feedback">{errors.createmod_version}</div>}
                    </div>
                  </div>
                  
                  <div className="row">
                    {/* Categories */}
                    <div className="col-md-6 mb-3">
                      <label className="form-label required">Categories</label>
                      <select 
                        className={`form-select ${errors.categories ? 'is-invalid' : ''}`}
                        name="categories"
                        value={formData.categories}
                        onChange={handleInputChange}
                        multiple
                        size="5"
                        required
                      >
                        {categories.map(category => (
                          <option key={category.id} value={category.id}>
                            {category.name}
                          </option>
                        ))}
                      </select>
                      {errors.categories && <div className="invalid-feedback">{errors.categories}</div>}
                      <small className="form-hint">Hold Ctrl/Cmd to select multiple categories.</small>
                    </div>
                    
                    {/* Tags */}
                    <div className="col-md-6 mb-3">
                      <label className="form-label">Tags</label>
                      <select 
                        className="form-select"
                        name="tags"
                        value={formData.tags}
                        onChange={handleInputChange}
                        multiple
                        size="5"
                      >
                        {tags.map(tag => (
                          <option key={tag.id} value={tag.id}>
                            {tag.name}
                          </option>
                        ))}
                      </select>
                      <small className="form-hint">Optional. Hold Ctrl/Cmd to select multiple tags.</small>
                    </div>
                  </div>
                  
                  {/* Dependencies */}
                  <div className="mb-3">
                    <div className="form-check form-switch">
                      <input 
                        className="form-check-input" 
                        type="checkbox" 
                        name="has_dependencies"
                        checked={formData.has_dependencies}
                        onChange={handleInputChange}
                      />
                      <label className="form-check-label">This schematic has dependencies</label>
                    </div>
                  </div>
                  
                  {formData.has_dependencies && (
                    <div className="mb-3">
                      <label className="form-label">Dependencies</label>
                      <textarea 
                        className="form-control"
                        name="dependencies"
                        value={formData.dependencies}
                        onChange={handleInputChange}
                        rows="3"
                        placeholder="List the mods required for this schematic"
                      ></textarea>
                      <small className="form-hint">HTML is allowed for formatting.</small>
                    </div>
                  )}
                  
                  {/* YouTube Video */}
                  <div className="mb-3">
                    <label className="form-label">YouTube Video</label>
                    <input 
                      type="text" 
                      className={`form-control ${errors.video ? 'is-invalid' : ''}`}
                      name="video"
                      value={formData.video}
                      onChange={handleInputChange}
                      placeholder="YouTube video URL or ID (optional)"
                    />
                    {errors.video && <div className="invalid-feedback">{errors.video}</div>}
                    <small className="form-hint">Optional. Enter a YouTube video URL or just the video ID.</small>
                  </div>
                  
                  {/* Submit Button */}
                  <div className="form-footer">
                    <button 
                      type="submit" 
                      className="btn btn-primary"
                      disabled={isSubmitting}
                    >
                      {isSubmitting ? (
                        <>
                          <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                          Uploading...
                        </>
                      ) : 'Upload Schematic'}
                    </button>
                  </div>
                </form>
              )}
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
    // Validate authentication on the server side
    const { isAuthenticated, user } = await validateServerAuth(context.req);
    
    console.log('[SERVER] Upload page - Auth validation result:', { 
      isAuthenticated, 
      userId: user?.id,
      username: user?.username 
    });
    
    // Get categories
    const categoriesData = await getCategories();
    const categories = categoriesData.items || [];
    
    // Get tags
    const tagsData = await getTags();
    const tags = tagsData.items || [];
    
    // Get Minecraft versions
    const minecraftVersionsData = await getMinecraftVersions();
    const minecraftVersions = minecraftVersionsData.items || [];
    
    // Get Create mod versions
    const createmodVersionsData = await getCreateModVersions();
    const createmodVersions = createmodVersionsData.items || [];
    
    // If not authenticated, redirect to login page
    if (!isAuthenticated) {
      return {
        redirect: {
          destination: '/login?redirect=/upload',
          permanent: false,
        },
      };
    }
    
    return {
      props: {
        categories,
        tags,
        minecraftVersions,
        createmodVersions,
        isAuthenticated,
        user: user ? JSON.parse(JSON.stringify(user)) : null // Serialize user object for Next.js
      }
    };
  } catch (error) {
    console.error('[SERVER] Error in getServerSideProps for upload page:', error);
    
    // Return empty data on error
    return {
      props: {
        categories: [],
        tags: [],
        minecraftVersions: [],
        createmodVersions: [],
        isAuthenticated: false,
        user: null
      }
    };
  }
}