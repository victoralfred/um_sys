import { Component } from 'solid-js';
import { useUI } from '../contexts/UIContext';
import { Button } from '../components/buttons/Button';
import { Card } from '../components/cards/Card';

// Example component demonstrating UI state management
export const UIExample: Component = () => {
  const ui = useUI();

  const showSuccessNotification = () => {
    ui.notifySuccess('Success!', 'This is a success notification');
  };

  const showErrorNotification = () => {
    ui.notifyError('Error!', 'This is an error notification that persists');
  };

  const showWarningNotification = () => {
    ui.notifyWarning('Warning!', 'This is a warning notification');
  };

  const showInfoNotification = () => {
    ui.notifyInfo('Information', 'This is an info notification');
  };

  const showCustomNotification = () => {
    ui.addNotification(
      'info',
      'Custom Notification',
      'This notification has a custom action',
      10000, // 10 seconds
      {
        label: 'Take Action',
        onClick: () => {
          ui.notifySuccess('Action Taken!', 'You clicked the notification action');
        }
      }
    );
  };

  const showModal = () => {
    const modalId = ui.openModal(
      'div', // In a real app, this would be a proper component
      { 
        children: 'This is a modal content. Press Escape to close or click the button below.',
        style: { padding: '20px', 'text-align': 'center' }
      },
      {
        size: 'md',
        onClose: () => {
          ui.notifyInfo('Modal Closed', 'The modal was closed');
        }
      }
    );

    // Add a close button after 2 seconds (just for demo)
    if (typeof window !== 'undefined') {
      window.setTimeout(() => {
        ui.closeModal(modalId);
      }, 3000);
    }
  };

  const showLoadingState = () => {
    ui.setGlobalLoading(true, 'Processing your request...');
    
    // Simulate async operation
    if (typeof window !== 'undefined') {
      window.setTimeout(() => {
        ui.setGlobalLoading(false);
        ui.notifySuccess('Complete!', 'The operation finished successfully');
      }, 3000);
    }
  };

  const clearAllNotifications = () => {
    ui.clearAllNotifications();
  };

  return (
    <div style={{ "max-width": "800px", margin: "0 auto", padding: "20px" }}>
      <h1>UI State Examples</h1>
      <p style={{ color: "#6B778C", "margin-bottom": "24px" }}>
        This page demonstrates various UI state management features including notifications, modals, and loading states.
      </p>

      {/* Notifications Section */}
      <Card class="mb-4">
        <div class="p-4">
          <h2 class="mb-4">Notifications</h2>
          <div class="flex gap-2 mb-4 flex-wrap">
            <Button variant="primary" onClick={showSuccessNotification}>
              Success Notification
            </Button>
            <Button variant="danger" onClick={showErrorNotification}>
              Error Notification
            </Button>
            <Button variant="secondary" onClick={showWarningNotification}>
              Warning Notification
            </Button>
            <Button variant="ghost" onClick={showInfoNotification}>
              Info Notification
            </Button>
          </div>
          <div class="flex gap-2 flex-wrap">
            <Button variant="secondary" onClick={showCustomNotification}>
              Custom with Action
            </Button>
            <Button variant="ghost" onClick={clearAllNotifications}>
              Clear All Notifications
            </Button>
          </div>
        </div>
      </Card>

      {/* Modals Section */}
      <Card class="mb-4">
        <div class="p-4">
          <h2 class="mb-4">Modals</h2>
          <div class="flex gap-2">
            <Button variant="primary" onClick={showModal}>
              Show Modal (Auto-closes in 3s)
            </Button>
          </div>
          <p class="mt-2" style={{ "font-size": "14px", color: "#6B778C" }}>
            The modal will demonstrate escape key handling and programmatic closing.
          </p>
        </div>
      </Card>

      {/* Loading States Section */}
      <Card class="mb-4">
        <div class="p-4">
          <h2 class="mb-4">Loading States</h2>
          <div class="flex gap-2">
            <Button 
              variant="primary" 
              onClick={showLoadingState}
              disabled={ui.state.isLoading}
            >
              Show Global Loading (3s)
            </Button>
          </div>
          <p class="mt-2" style={{ "font-size": "14px", color: "#6B778C" }}>
            This will show a global loading state with a custom message.
          </p>
        </div>
      </Card>

      {/* Current State Display */}
      <Card>
        <div class="p-4">
          <h2 class="mb-4">Current UI State</h2>
          <div class="grid" style={{ "grid-template-columns": "1fr 1fr", gap: "16px" }}>
            <div>
              <h3 style={{ "margin-bottom": "8px" }}>Notifications ({ui.state.notifications.length})</h3>
              <div style={{ "max-height": "200px", overflow: "auto" }}>
                {ui.state.notifications.length === 0 ? (
                  <p style={{ color: "#6B778C", "font-size": "14px" }}>No active notifications</p>
                ) : (
                  <div class="space-y-2">
                    {ui.state.notifications.map((notification) => (
                      <div 
                        class="p-2"
                        style={{ 
                          "background-color": "#F7F8F9", 
                          "border-radius": "4px",
                          "border-left": `4px solid ${
                            notification.type === 'success' ? '#00875A' :
                            notification.type === 'error' ? '#DE350B' :
                            notification.type === 'warning' ? '#FF8B00' : '#0052CC'
                          }`
                        }}
                      >
                        <div style={{ "font-weight": "500", "font-size": "12px" }}>
                          {notification.type.toUpperCase()}: {notification.title}
                        </div>
                        <div style={{ "font-size": "11px", color: "#6B778C" }}>
                          {notification.message}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
            <div>
              <h3 style={{ "margin-bottom": "8px" }}>Modals ({ui.state.modals.length})</h3>
              <div style={{ "max-height": "200px", overflow: "auto" }}>
                {ui.state.modals.length === 0 ? (
                  <p style={{ color: "#6B778C", "font-size": "14px" }}>No active modals</p>
                ) : (
                  <div class="space-y-2">
                    {ui.state.modals.map((modal) => (
                      <div 
                        class="p-2"
                        style={{ "background-color": "#F7F8F9", "border-radius": "4px" }}
                      >
                        <div style={{ "font-weight": "500", "font-size": "12px" }}>
                          Modal ID: {modal.id}
                        </div>
                        <div style={{ "font-size": "11px", color: "#6B778C" }}>
                          Size: {modal.size}, Closable: {modal.closable ? 'Yes' : 'No'}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
          <div class="mt-4">
            <h3 style={{ "margin-bottom": "8px" }}>Global Loading</h3>
            <p style={{ "font-size": "14px", color: ui.state.isLoading ? "#DE350B" : "#6B778C" }}>
              Status: {ui.state.isLoading ? 'Loading' : 'Idle'}
              {ui.state.loadingMessage && ` - ${ui.state.loadingMessage}`}
            </p>
          </div>
        </div>
      </Card>

      {/* Instructions */}
      <div class="mt-4 p-4" style={{ "background-color": "#F4F8FF", "border-radius": "4px", "border": "1px solid #DEEBFF" }}>
        <h3 style={{ color: "#0052CC", "margin-bottom": "8px" }}>💡 Pro Tips</h3>
        <ul style={{ "margin": 0, "padding-left": "20px", "font-size": "14px", color: "#172B4D" }}>
          <li>Error notifications persist by default (duration = 0)</li>
          <li>Other notification types auto-dismiss after 5 seconds</li>
          <li>Press Escape key to close the topmost modal</li>
          <li>Notifications can include custom actions</li>
          <li>Global loading state is useful for page-level operations</li>
        </ul>
      </div>
    </div>
  );
};