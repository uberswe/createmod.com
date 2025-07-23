import { useState, useEffect } from 'react';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';

export default function Layout({ children, title = 'CreateMod.com', description = '', categories = [] }) {
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState(null);
  const [theme, setTheme] = useState('light');

  useEffect(() => {
    // Check if user is authenticated
    const checkAuth = async () => {
      try {
        const response = await fetch('/api/collections/users/auth-refresh', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
        });
        
        if (response.ok) {
          const userData = await response.json();
          setIsAuthenticated(true);
          setUser(userData.record);
        } else {
          setIsAuthenticated(false);
          setUser(null);
        }
      } catch (error) {
        console.error('Error checking authentication:', error);
        setIsAuthenticated(false);
        setUser(null);
      }
    };

    // Check for theme preference
    const savedTheme = localStorage.getItem('createmodTheme') || 'light';
    setTheme(savedTheme);
    document.documentElement.setAttribute('data-bs-theme', savedTheme);

    checkAuth();
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('createmodTheme', newTheme);
    document.documentElement.setAttribute('data-bs-theme', newTheme);
  };

  const handleLogout = async () => {
    try {
      await fetch('/api/collections/users/auth-refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });
      
      // Clear cookie
      document.cookie = 'create-mod-auth=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
      
      setIsAuthenticated(false);
      setUser(null);
      router.push('/');
    } catch (error) {
      console.error('Error logging out:', error);
    }
  };

  return (
    <div className="page">
      <Head>
        <title>{title}</title>
        <meta name="description" content={description} />
        <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
        <link rel="shortcut icon" href="/favicon.ico" type="image/x-icon" />
      </Head>

      {/* Sidebar */}
      <aside className="navbar navbar-vertical navbar-expand-lg" data-bs-theme="dark">
        <div className="container-fluid">
          <button className="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#sidebar-menu" aria-controls="sidebar-menu" aria-expanded="false" aria-label="Toggle navigation">
            <span className="navbar-toggler-icon"></span>
          </button>
          <h1 className="navbar-brand navbar-brand-autodark ms-3">
            <Link href="/">
              <img alt="CreateMod.com logo" src="/logo.png" />
            </Link>
          </h1>
          
          {/* Mobile nav items */}
          <div className="navbar-nav flex-row d-lg-none">
            <div className="nav-item dropdown auth-section">
              {isAuthenticated ? (
                <a href="#" className="nav-link d-flex lh-1 text-reset p-0 dropdown" data-bs-toggle="dropdown" aria-label="Open user menu">
                  {user?.avatar && <img className="avatar avatar-sm auth-avatar" src={user.avatar} alt={user.username} />}
                  <div className="d-none d-xl-block ps-2">
                    <div className="auth-username">{user?.username}</div>
                  </div>
                </a>
              ) : (
                <Link href="/login" className="nav-link">Login</Link>
              )}
              
              {isAuthenticated && (
                <div className="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
                  <Link href={`/author/${user?.username?.toLowerCase()}`} className="dropdown-item">Profile</Link>
                  <div className="dropdown-divider"></div>
                  <Link href="/settings" className="dropdown-item">Settings</Link>
                  <a className="dropdown-item logout-button" onClick={handleLogout}>Logout</a>
                </div>
              )}
            </div>
          </div>
          
          {/* Sidebar menu */}
          <div className="collapse navbar-collapse" id="sidebar-menu">
            <ul className="navbar-nav pt-lg-3">
              <li className="nav-item">
                <Link href="/" className={`nav-link ${router.pathname === '/' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M5 12l-2 0l9 -9l9 9l-2 0" />
                      <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7" />
                      <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Home</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/search/?sort=6&rating=-1&category=all&tag=all" className={`nav-link ${router.pathname.startsWith('/search') ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0" />
                      <path d="M21 21l-6 -6" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Search</span>
                </Link>
              </li>
              
              {categories.map(category => (
                <li key={category.id} className="nav-item">
                  <Link href={`/search?category=${category.key}&sort=6`} className="nav-link" style={{ marginLeft: '15px' }}>
                    <span className="nav-link-icon d-md-none d-lg-inline-block">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" className="bi bi-dash-lg" viewBox="0 0 16 16">
                        <path fill-rule="evenodd" d="M2 8a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11A.5.5 0 0 1 2 8"/>
                      </svg>
                    </span>
                    <span className="nav-link-title">{category.name}</span>
                  </Link>
                </li>
              ))}
              
              <li className="nav-item">
                <Link href="/upload" className={`nav-link ${router.pathname === '/upload' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M9 11l3 3l8 -8" />
                      <path d="M20 12v6a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12a2 2 0 0 1 2 -2h9" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Upload</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/rules" className={`nav-link ${router.pathname === '/rules' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M17 3l4 4l-14 14l-4 -4z" />
                      <path d="M16 7l-1.5 -1.5" />
                      <path d="M13 10l-1.5 -1.5" />
                      <path d="M10 13l-1.5 -1.5" />
                      <path d="M7 16l-1.5 -1.5" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Rules</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/news" className={`nav-link ${router.pathname === '/news' || router.pathname.startsWith('/news/') ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11" />
                      <path d="M8 8l4 0" />
                      <path d="M8 12l4 0" />
                      <path d="M8 16l4 0" />
                    </svg>
                  </span>
                  <span className="nav-link-title">News</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/guide" className={`nav-link ${router.pathname === '/guide' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M3.5 5.5l1.5 1.5l2.5 -2.5" />
                      <path d="M3.5 11.5l1.5 1.5l2.5 -2.5" />
                      <path d="M3.5 17.5l1.5 1.5l2.5 -2.5" />
                      <path d="M11 6l9 0" />
                      <path d="M11 12l9 0" />
                      <path d="M11 18l9 0" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Guide</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/explore" className={`nav-link ${router.pathname === '/explore' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M7 3m0 2.667a2.667 2.667 0 0 1 2.667 -2.667h8.666a2.667 2.667 0 0 1 2.667 2.667v8.666a2.667 2.667 0 0 1 -2.667 2.667h-8.666a2.667 2.667 0 0 1 -2.667 -2.667z" />
                      <path d="M4.012 7.26a2.005 2.005 0 0 0 -1.012 1.737v10c0 1.1 .9 2 2 2h10c.75 0 1.158 -.385 1.5 -1" />
                      <path d="M17 7h.01" />
                      <path d="M7 13l3.644 -3.644a1.21 1.21 0 0 1 1.712 0l3.644 3.644" />
                      <path d="M15 12l1.644 -1.644a1.21 1.21 0 0 1 1.712 0l2.644 2.644" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Explore</span>
                </Link>
              </li>
              
              <li className="nav-item">
                <Link href="/contact" className={`nav-link ${router.pathname === '/contact' ? 'active' : ''}`}>
                  <span className="nav-link-icon d-md-none d-lg-inline-block">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                      <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                      <path d="M3 7a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-10z" />
                      <path d="M3 7l9 6l9 -6" />
                    </svg>
                  </span>
                  <span className="nav-link-title">Contact</span>
                </Link>
              </li>
              
              {!isAuthenticated ? (
                <li className="nav-item">
                  <Link href="/login" className={`nav-link ${router.pathname === '/login' ? 'active' : ''}`}>
                    <span className="nav-link-icon d-md-none d-lg-inline-block">
                      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M15 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2" />
                        <path d="M21 12h-13l3 -3" />
                        <path d="M11 15l-3 -3" />
                      </svg>
                    </span>
                    <span className="nav-link-title">Login</span>
                  </Link>
                </li>
              ) : (
                <>
                  <li className="nav-item d-xl-none d-inline-block">
                    <Link href="/settings" className={`nav-link ${router.pathname === '/settings' ? 'active' : ''}`}>
                      <span className="nav-link-icon d-md-none d-lg-inline-block">
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                          <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                          <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z" />
                          <path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" />
                        </svg>
                      </span>
                      <span className="nav-link-title">Settings</span>
                    </Link>
                  </li>
                  
                  <li className="nav-item d-xl-none d-inline-block">
                    <Link href={`/author/${user?.username?.toLowerCase()}`} className="nav-link">
                      <span className="nav-link-icon d-md-none d-lg-inline-block">
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                          <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                          <path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0" />
                          <path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" />
                        </svg>
                      </span>
                      <span className="nav-link-title">Profile</span>
                    </Link>
                  </li>
                  
                  <li className="nav-item">
                    <a className="nav-link" onClick={handleLogout}>
                      <span className="nav-link-icon d-md-none d-lg-inline-block">
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="icon">
                          <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                          <path d="M15 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2" />
                          <path d="M21 12h-13l3 -3" />
                          <path d="M11 15l-3 -3" />
                        </svg>
                      </span>
                      <span className="nav-link-title">Logout</span>
                    </a>
                  </li>
                </>
              )}
            </ul>
          </div>
        </div>
      </aside>
      
      {/* Main content */}
      <div className="page-wrapper">
        {/* Header */}
        <div className="page-header d-print-none">
          <div className="container-xl">
            <div className="row g-2 align-items-center">
              <div className="col">
                <div className="my-2 my-md-0 flex-grow-1 flex-md-grow-0 order-first order-md-last d-none d-md-block">
                  <form action="/search" method="post" autoComplete="off" id="search-form" noValidate>
                    <div className="input-icon">
                      <span className="input-icon-addon">
                        <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                          <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                          <path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0"/>
                          <path d="M21 21l-6 -6"/>
                        </svg>
                      </span>
                      <input id="search-field" type="text" name="advanced-search-term" className="form-control" placeholder="Search…" aria-label="Search CreateMod.com" />
                    </div>
                  </form>
                </div>
              </div>
              
              <div className="col-auto">
                <div className="d-none d-md-flex">
                  <a href="#" onClick={toggleTheme} className="nav-link px-0" data-bs-toggle="tooltip" data-bs-placement="bottom" aria-label={theme === 'light' ? 'Enable dark mode' : 'Enable light mode'}>
                    {theme === 'light' ? (
                      <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z" />
                      </svg>
                    ) : (
                      <svg xmlns="http://www.w3.org/2000/svg" className="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <path d="M12 12m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0" />
                        <path d="M3 12h1m8 -9v1m8 8h1m-9 8v1m-6.4 -15.4l.7 .7m12.1 -.7l-.7 .7m0 11.4l.7 .7m-12.1 -.7l-.7 .7" />
                      </svg>
                    )}
                  </a>
                </div>
              </div>
              
              <div className="col col-auto d-none d-lg-block auth-section">
                {isAuthenticated ? (
                  <div className="nav-item dropdown">
                    <a href="#" className="nav-link d-flex lh-1 text-reset p-0" data-bs-toggle="dropdown" aria-label="Open user menu">
                      {user?.avatar && <img className="avatar avatar-sm auth-avatar" src={user.avatar} alt={user.username} />}
                      <div className="d-none d-xl-block ps-2">
                        <div className="auth-username">{user?.username}</div>
                      </div>
                    </a>
                    <div className="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
                      <Link href={`/author/${user?.username?.toLowerCase()}`} className="dropdown-item">Profile</Link>
                      <div className="dropdown-divider"></div>
                      <Link href="/settings" className="dropdown-item">Settings</Link>
                      <a className="dropdown-item logout-button" onClick={handleLogout}>Logout</a>
                    </div>
                  </div>
                ) : (
                  <Link href="/login" className="nav-link">Login</Link>
                )}
              </div>
            </div>
          </div>
        </div>
        
        {/* Page body */}
        <div className="page-body">
          {children}
        </div>
        
        {/* Footer */}
        <footer className="footer footer-transparent d-print-none">
          <div className="container-xl">
            <div className="row text-center align-items-center flex-row-reverse">
              <div className="col-lg-auto ms-lg-auto">
                <ul className="list-inline list-inline-dots mb-0">
                  <li className="list-inline-item"><Link href="/terms-of-service" className="link-secondary">Terms Of Service</Link></li>
                  <li className="list-inline-item"><Link href="/privacy-policy" className="link-secondary">Privacy Policy</Link></li>
                  <li className="list-inline-item"><a href="https://github.com/uberswe/createmod" target="_blank" className="link-secondary" rel="noopener">Source code</a></li>
                </ul>
              </div>
              <div className="col-12 col-lg-auto mt-3 mt-lg-0">
                <ul className="list-inline list-inline-dots mb-0">
                  <li className="list-inline-item">
                    Copyright &copy; {new Date().getFullYear()}
                    <a href="https://createmod.com" className="link-secondary">CreateMod.com</a>.
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
        </footer>
      </div>
    </div>
  );
}