import { Router, Route, Navigate } from '@solidjs/router';
import { Component, lazy } from 'solid-js';
import { AuthGuard } from '../contexts/AuthContext';
import { RootProvider } from '../providers/RootProvider';

// Lazy load pages for better performance
const LoginPage = lazy(() => import('../features/auth/LoginPage').then(m => ({ default: m.LoginPage })));
const RegisterPage = lazy(() => import('../features/auth/RegisterPage').then(m => ({ default: m.RegisterPage })));
const ProfilePage = lazy(() => import('../features/auth/ProfilePage').then(m => ({ default: m.ProfilePage })));

// Example pages (will be moved to proper locations later)
const AuthExample = lazy(() => import('../examples/AuthExample').then(m => ({ default: m.AuthExample })));
const UsersExample = lazy(() => import('../examples/UsersExample').then(m => ({ default: m.UsersExample })));
const UIExample = lazy(() => import('../examples/UIExample').then(m => ({ default: m.UIExample })));

// Layout components
const AppLayout: Component<{ children: any }> = (props) => {
  return (
    <div class="min-h-screen" style={{ "background-color": "#F7F8F9" }}>
      <RootProvider>
        {props.children}
      </RootProvider>
    </div>
  );
};

const AuthLayout: Component<{ children: any }> = (props) => {
  return (
    <AuthGuard requireAuth={false} fallback={<Navigate href="/dashboard" />}>
      <AppLayout>
        {props.children}
      </AppLayout>
    </AuthGuard>
  );
};

const DashboardLayout: Component<{ children: any }> = (props) => {
  return (
    <AuthGuard requireAuth={true} fallback={<Navigate href="/login" />}>
      <AppLayout>
        <div class="flex min-h-screen">
          {/* Sidebar Navigation */}
          <nav 
            class="w-64 bg-white border-r"
            style={{ "border-color": "#DFE1E6" }}
          >
            <div class="p-6">
              <h2 class="text-heading-md mb-6">UManager</h2>
              <ul class="space-y-2">
                <li>
                  <a 
                    href="/dashboard" 
                    class="flex items-center p-3 rounded hover:bg-ds-background-subtle"
                  >
                    Dashboard
                  </a>
                </li>
                <li>
                  <a 
                    href="/users" 
                    class="flex items-center p-3 rounded hover:bg-ds-background-subtle"
                  >
                    Users
                  </a>
                </li>
                <li>
                  <a 
                    href="/profile" 
                    class="flex items-center p-3 rounded hover:bg-ds-background-subtle"
                  >
                    Profile
                  </a>
                </li>
                <li>
                  <a 
                    href="/examples" 
                    class="flex items-center p-3 rounded hover:bg-ds-background-subtle"
                  >
                    Examples
                  </a>
                </li>
              </ul>
            </div>
          </nav>

          {/* Main Content */}
          <main class="flex-1">
            {props.children}
          </main>
        </div>
      </AppLayout>
    </AuthGuard>
  );
};

// Dashboard placeholder
const Dashboard: Component = () => {
  return (
    <div class="p-8">
      <h1 class="text-heading-lg mb-4">Dashboard</h1>
      <p>Welcome to UManager! This is your dashboard.</p>
    </div>
  );
};

// Examples navigation page
const ExamplesPage: Component = () => {
  return (
    <div class="p-8">
      <h1 class="text-heading-lg mb-6">Examples</h1>
      <div class="grid gap-4" style={{ "grid-template-columns": "repeat(auto-fit, minmax(300px, 1fr))" }}>
        <div class="card p-6">
          <h3 class="text-heading-sm mb-2">Authentication Example</h3>
          <p class="mb-4" style={{ color: "#6B778C" }}>
            Complete authentication flow with login, register, and logout.
          </p>
          <a href="/examples/auth" class="btn btn-primary">
            View Example
          </a>
        </div>

        <div class="card p-6">
          <h3 class="text-heading-sm mb-2">Users Management Example</h3>
          <p class="mb-4" style={{ color: "#6B778C" }}>
            User management interface with CRUD operations, filtering, and pagination.
          </p>
          <a href="/examples/users" class="btn btn-primary">
            View Example
          </a>
        </div>

        <div class="card p-6">
          <h3 class="text-heading-sm mb-2">UI State Example</h3>
          <p class="mb-4" style={{ color: "#6B778C" }}>
            Notifications, modals, and loading states demonstration.
          </p>
          <a href="/examples/ui" class="btn btn-primary">
            View Example
          </a>
        </div>
      </div>
    </div>
  );
};

// Main App Router
export const AppRouter: Component = () => {
  return (
    <Router>
      {/* Public Routes (Auth) */}
      <Route path="/login" component={() => (
        <AuthLayout>
          <LoginPage />
        </AuthLayout>
      )} />

      <Route path="/register" component={() => (
        <AuthLayout>
          <RegisterPage />
        </AuthLayout>
      )} />

      {/* Protected Routes (Dashboard) */}
      <Route path="/dashboard" component={() => (
        <DashboardLayout>
          <Dashboard />
        </DashboardLayout>
      )} />

      <Route path="/users" component={() => (
        <DashboardLayout>
          <UsersExample />
        </DashboardLayout>
      )} />

      <Route path="/profile" component={() => (
        <DashboardLayout>
          <ProfilePage />
        </DashboardLayout>
      )} />

      {/* Examples Routes */}
      <Route path="/examples" component={() => (
        <DashboardLayout>
          <ExamplesPage />
        </DashboardLayout>
      )} />

      <Route path="/examples/auth" component={() => (
        <AppLayout>
          <div class="p-8">
            <div class="mb-4">
              <a href="/examples" class="btn btn-ghost">← Back to Examples</a>
            </div>
            <AuthExample />
          </div>
        </AppLayout>
      )} />

      <Route path="/examples/users" component={() => (
        <DashboardLayout>
          <UsersExample />
        </DashboardLayout>
      )} />

      <Route path="/examples/ui" component={() => (
        <AppLayout>
          <div class="p-8">
            <div class="mb-4">
              <a href="/examples" class="btn btn-ghost">← Back to Examples</a>
            </div>
            <UIExample />
          </div>
        </AppLayout>
      )} />

      {/* Redirects */}
      <Route path="/" component={() => <Navigate href="/dashboard" />} />
      
      {/* 404 - Catch all */}
      <Route path="/*all" component={() => (
        <AppLayout>
          <div class="min-h-screen flex items-center justify-center">
            <div class="text-center">
              <h1 class="text-heading-lg mb-4">Page Not Found</h1>
              <p class="mb-6" style={{ color: "#6B778C" }}>
                The page you're looking for doesn't exist.
              </p>
              <a href="/dashboard" class="btn btn-primary">
                Go to Dashboard
              </a>
            </div>
          </div>
        </AppLayout>
      )} />
    </Router>
  );
};