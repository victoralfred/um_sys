import { ParentComponent, onMount } from 'solid-js';
import { AuthProvider } from '../contexts/AuthContext';
import { UIProvider } from '../contexts/UIContext';
import { UsersProvider } from '../contexts/UsersContext';
import { uiActions } from '../stores/ui';

// Root provider that combines all context providers
export const RootProvider: ParentComponent = (props) => {
  // Set up global event listeners
  onMount(() => {
    // Handle keyboard shortcuts
    const handleKeyPress = (event: KeyboardEvent) => {
      uiActions.handleKeyPress(event);
    };

    // Handle unauthorized events from API
    const handleUnauthorized = () => {
      // This will be handled by the auth store
      console.log('Unauthorized event received');
    };

    // Add event listeners
    document.addEventListener('keydown', handleKeyPress);
    window.addEventListener('auth:unauthorized', handleUnauthorized);

    // Cleanup
    return () => {
      document.removeEventListener('keydown', handleKeyPress);
      window.removeEventListener('auth:unauthorized', handleUnauthorized);
    };
  });

  return (
    <UIProvider>
      <AuthProvider>
        <UsersProvider>
          {props.children}
        </UsersProvider>
      </AuthProvider>
    </UIProvider>
  );
};