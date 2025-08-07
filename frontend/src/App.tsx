import type { Component } from 'solid-js';
import './styles/globals.css';

const App: Component = () => {
  return (
    <div class="min-h-screen bg-ds-background">
      <div class="container-ds py-8">
        <h1 class="text-heading-xl mb-6">User Management System</h1>
        <div class="card">
          <div class="card-header">
            <h2 class="text-heading-sm">Welcome</h2>
          </div>
          <div class="card-body">
            <p class="text-body mb-4">
              Enterprise User Management Frontend built with SolidJS and Atlassian Design System.
            </p>
            <div class="flex gap-3">
              <button class="btn btn-primary">Get Started</button>
              <button class="btn btn-secondary">Learn More</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default App;