import React from 'react';
import Link from 'next/link';

/**
 * Footer component with links, copyright information, and disclaimers
 */
export default function Footer() {
  const currentYear = new Date().getFullYear();
  
  return (
    <footer className="footer footer-transparent d-print-none">
      <div className="container-xl">
        <div className="row text-center align-items-center flex-row-reverse">
          <div className="col-lg-auto ms-lg-auto">
            <div><p><span data-ccpa-link="1"></span></p></div>
            <div id="ncmp-consent-link"></div>
          </div>
        </div>
        
        <div className="row text-center align-items-center flex-row-reverse">
          <div className="col-lg-auto ms-lg-auto">
            <ul className="list-inline list-inline-dots mb-0">
              <li className="list-inline-item">
                <Link href="/terms-of-service" className="link-secondary">
                  Terms Of Service
                </Link>
              </li>
              <li className="list-inline-item">
                <Link href="/privacy-policy" className="link-secondary">
                  Privacy Policy
                </Link>
              </li>
              <li className="list-inline-item">
                <a 
                  href="https://github.com/uberswe/createmod" 
                  target="_blank" 
                  rel="noopener noreferrer" 
                  className="link-secondary"
                >
                  Source code
                </a>
              </li>
            </ul>
          </div>
          
          <div className="col-12 col-lg-auto mt-3 mt-lg-0">
            <ul className="list-inline list-inline-dots mb-0">
              <li className="list-inline-item">
                Copyright &copy; {currentYear}
                <Link href="https://createmod.com" className="link-secondary">
                  {' CreateMod.com'}
                </Link>.
                All rights reserved.
              </li>
            </ul>
            
            <ul className="list-inline list-inline-dots mb-0">
              <li className="list-inline-item">
                NOT APPROVED BY OR ASSOCIATED WITH MOJANG OR MICROSOFT.
              </li>
            </ul>
            
            <ul className="list-inline list-inline-dots mb-0">
              <li className="list-inline-item">
                This site is <b>NOT</b> associated with the Create mod dev team.
              </li>
            </ul>
            
            <ul className="list-inline list-inline-dots mb-0">
              <li className="list-inline-item">
                This website does <b>NOT</b> own or claim to own any of the content posted onto it, all content
                has been provided by registered users.
              </li>
            </ul>
          </div>
        </div>
      </div>
      
      {/* Script for initializing PocketBase client on the client side */}
      <script
        dangerouslySetInnerHTML={{
          __html: `
            document.addEventListener("DOMContentLoaded", function() {
              // Initialize theme from localStorage
              const savedTheme = localStorage.getItem('createmodTheme');
              if (savedTheme) {
                document.documentElement.setAttribute('data-bs-theme', savedTheme);
              } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
                document.documentElement.setAttribute('data-bs-theme', 'dark');
              }
              
              // Add theme toggle event listeners
              const darkToggle = document.querySelector('.hide-theme-dark');
              const lightToggle = document.querySelector('.hide-theme-light');
              
              if (darkToggle) {
                darkToggle.addEventListener('click', function(e) {
                  e.preventDefault();
                  localStorage.setItem('createmodTheme', 'dark');
                  document.documentElement.setAttribute('data-bs-theme', 'dark');
                });
              }
              
              if (lightToggle) {
                lightToggle.addEventListener('click', function(e) {
                  e.preventDefault();
                  localStorage.setItem('createmodTheme', 'light');
                  document.documentElement.setAttribute('data-bs-theme', 'light');
                });
              }
            });
          `
        }}
      />
    </footer>
  );
}