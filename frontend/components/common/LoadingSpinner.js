import React from 'react';

/**
 * Loading spinner component for consistent loading states across the application
 * 
 * @param {Object} props - Component props
 * @param {string} props.size - Size of the spinner ('sm', 'md', 'lg')
 * @param {string} props.color - Color of the spinner ('primary', 'secondary', 'success', 'danger', 'warning', 'info')
 * @param {string} props.text - Text to display below the spinner
 * @param {boolean} props.fullPage - Whether to display the spinner centered on the full page
 */
export default function LoadingSpinner({ 
  size = 'md', 
  color = 'primary', 
  text = 'Loading...', 
  fullPage = false 
}) {
  // Determine spinner size class
  const sizeClass = size === 'sm' ? 'spinner-border-sm' : 
                   size === 'lg' ? 'spinner-border-lg' : '';
  
  // Create the spinner element
  const spinner = (
    <div className="d-flex flex-column align-items-center">
      <div 
        className={`spinner-border text-${color} ${sizeClass}`} 
        role="status"
      >
        <span className="visually-hidden">Loading...</span>
      </div>
      {text && <p className="mt-2 text-muted">{text}</p>}
    </div>
  );
  
  // If fullPage is true, center the spinner on the page
  if (fullPage) {
    return (
      <div className="d-flex justify-content-center align-items-center" style={{ height: '80vh' }}>
        {spinner}
      </div>
    );
  }
  
  return spinner;
}