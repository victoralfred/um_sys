import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { uiState, uiActions } from '../ui';

// Mock createUniqueId
vi.mock('solid-js', async () => {
  const actual = await vi.importActual('solid-js');
  return {
    ...actual,
    createUniqueId: vi.fn(() => 'mock-id-' + Math.random().toString(36).substr(2, 9)),
  };
});

describe('UI Store', () => {
  beforeEach(() => {
    // Clear UI state before each test
    uiState.notifications = [];
    uiState.modals = [];
    uiState.isLoading = false;
    uiState.loadingMessage = undefined;
    
    // Clear all timers
    vi.clearAllTimers();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  describe('Initial State', () => {
    it('should have correct initial state', () => {
      expect(uiState.notifications).toEqual([]);
      expect(uiState.modals).toEqual([]);
      expect(uiState.isLoading).toBe(false);
      expect(uiState.loadingMessage).toBeUndefined();
    });
  });

  describe('Notifications', () => {
    it('should add notification', () => {
      const id = uiActions.addNotification('success', 'Test Title', 'Test Message');

      expect(uiState.notifications).toHaveLength(1);
      expect(uiState.notifications[0]).toMatchObject({
        id,
        type: 'success',
        title: 'Test Title',
        message: 'Test Message',
        duration: 5000,
      });
      expect(uiState.notifications[0].createdAt).toBeInstanceOf(Date);
    });

    it('should add notification with custom duration', () => {
      uiActions.addNotification('info', 'Test', undefined, 3000);

      expect(uiState.notifications[0].duration).toBe(3000);
    });

    it('should set error notifications to persist by default', () => {
      uiActions.addNotification('error', 'Error Message');

      expect(uiState.notifications[0].duration).toBe(0);
      expect(uiState.notifications[0].type).toBe('error');
    });

    it('should auto-dismiss notifications with duration', () => {
      uiActions.addNotification('success', 'Auto Dismiss', undefined, 1000);

      expect(uiState.notifications).toHaveLength(1);

      // Fast-forward time
      vi.advanceTimersByTime(1000);

      expect(uiState.notifications).toHaveLength(0);
    });

    it('should not auto-dismiss notifications with duration 0', () => {
      uiActions.addNotification('error', 'Persist', undefined, 0);

      expect(uiState.notifications).toHaveLength(1);

      // Fast-forward time
      vi.advanceTimersByTime(5000);

      expect(uiState.notifications).toHaveLength(1);
    });

    it('should remove notification by id', () => {
      const id1 = uiActions.addNotification('success', 'First');
      const id2 = uiActions.addNotification('info', 'Second');

      expect(uiState.notifications).toHaveLength(2);

      uiActions.removeNotification(id1);

      expect(uiState.notifications).toHaveLength(1);
      expect(uiState.notifications[0].id).toBe(id2);
    });

    it('should clear all notifications', () => {
      uiActions.addNotification('success', 'First');
      uiActions.addNotification('error', 'Second');
      uiActions.addNotification('warning', 'Third');

      expect(uiState.notifications).toHaveLength(3);

      uiActions.clearAllNotifications();

      expect(uiState.notifications).toHaveLength(0);
    });

    it('should use convenience methods for different types', () => {
      const successId = uiActions.notifySuccess('Success!');
      const errorId = uiActions.notifyError('Error!');
      const warningId = uiActions.notifyWarning('Warning!');
      const infoId = uiActions.notifyInfo('Info!');

      expect(uiState.notifications).toHaveLength(4);
      expect(uiState.notifications.find(n => n.id === successId)?.type).toBe('success');
      expect(uiState.notifications.find(n => n.id === errorId)?.type).toBe('error');
      expect(uiState.notifications.find(n => n.id === warningId)?.type).toBe('warning');
      expect(uiState.notifications.find(n => n.id === infoId)?.type).toBe('info');
    });

    it('should add notification with action', () => {
      const mockAction = vi.fn();
      uiActions.addNotification(
        'info',
        'Test',
        'Message',
        5000,
        { label: 'Click me', onClick: mockAction }
      );

      expect(uiState.notifications[0].action).toEqual({
        label: 'Click me',
        onClick: mockAction,
      });
    });
  });

  describe('Modals', () => {
    const MockComponent = 'MockComponent';

    it('should open modal with default options', () => {
      const id = uiActions.openModal(MockComponent);

      expect(uiState.modals).toHaveLength(1);
      expect(uiState.modals[0]).toMatchObject({
        id,
        component: MockComponent,
        props: {},
        size: 'md',
        closable: true,
      });
    });

    it('should open modal with custom options', () => {
      const onClose = vi.fn();
      const props = { title: 'Test Modal' };
      
      const id = uiActions.openModal(MockComponent, props, {
        size: 'lg',
        closable: false,
        onClose,
      });

      expect(uiState.modals[0]).toMatchObject({
        id,
        component: MockComponent,
        props,
        size: 'lg',
        closable: false,
        onClose,
      });
    });

    it('should close modal by id', () => {
      const onClose = vi.fn();
      const id1 = uiActions.openModal(MockComponent, {}, { onClose });
      const id2 = uiActions.openModal(MockComponent);

      expect(uiState.modals).toHaveLength(2);

      uiActions.closeModal(id1);

      expect(onClose).toHaveBeenCalled();
      expect(uiState.modals).toHaveLength(1);
      expect(uiState.modals[0].id).toBe(id2);
    });

    it('should close top modal', () => {
      const onClose1 = vi.fn();
      const onClose2 = vi.fn();
      
      uiActions.openModal(MockComponent, {}, { onClose: onClose1 });
      uiActions.openModal(MockComponent, {}, { onClose: onClose2 });

      expect(uiState.modals).toHaveLength(2);

      uiActions.closeTopModal();

      expect(onClose2).toHaveBeenCalled();
      expect(onClose1).not.toHaveBeenCalled();
      expect(uiState.modals).toHaveLength(1);
    });

    it('should close all modals', () => {
      const onClose1 = vi.fn();
      const onClose2 = vi.fn();
      
      uiActions.openModal(MockComponent, {}, { onClose: onClose1 });
      uiActions.openModal(MockComponent, {}, { onClose: onClose2 });

      expect(uiState.modals).toHaveLength(2);

      uiActions.closeAllModals();

      expect(onClose1).toHaveBeenCalled();
      expect(onClose2).toHaveBeenCalled();
      expect(uiState.modals).toHaveLength(0);
    });

    it('should handle closing modal with no onClose callback', () => {
      const id = uiActions.openModal(MockComponent);

      expect(() => {
        uiActions.closeModal(id);
      }).not.toThrow();

      expect(uiState.modals).toHaveLength(0);
    });
  });

  describe('Global Loading', () => {
    it('should set global loading state', () => {
      uiActions.setGlobalLoading(true, 'Loading data...');

      expect(uiState.isLoading).toBe(true);
      expect(uiState.loadingMessage).toBe('Loading data...');
    });

    it('should clear global loading state', () => {
      uiState.isLoading = true;
      uiState.loadingMessage = 'Loading...';

      uiActions.setGlobalLoading(false);

      expect(uiState.isLoading).toBe(false);
      expect(uiState.loadingMessage).toBeUndefined();
    });
  });

  describe('Keyboard Shortcuts', () => {
    const MockComponent = 'MockComponent';

    it('should close top modal on Escape key', () => {
      const onClose = vi.fn();
      uiActions.openModal(MockComponent, {}, { onClose });

      const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
      uiActions.handleKeyPress(escapeEvent);

      expect(onClose).toHaveBeenCalled();
      expect(uiState.modals).toHaveLength(0);
    });

    it('should not close modal on Escape if not closable', () => {
      uiActions.openModal(MockComponent, {}, { closable: false });

      const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
      uiActions.handleKeyPress(escapeEvent);

      expect(uiState.modals).toHaveLength(1);
    });

    it('should not close modal on other keys', () => {
      uiActions.openModal(MockComponent);

      const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' });
      uiActions.handleKeyPress(enterEvent);

      expect(uiState.modals).toHaveLength(1);
    });

    it('should handle Escape when no modals are open', () => {
      expect(() => {
        const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
        uiActions.handleKeyPress(escapeEvent);
      }).not.toThrow();
    });
  });

  describe('Stack Behavior', () => {
    const MockComponent = 'MockComponent';

    it('should maintain modal stack order', () => {
      const id1 = uiActions.openModal(MockComponent, { title: 'First' });
      const id2 = uiActions.openModal(MockComponent, { title: 'Second' });
      const id3 = uiActions.openModal(MockComponent, { title: 'Third' });

      expect(uiState.modals).toHaveLength(3);
      expect(uiState.modals[0].id).toBe(id1);
      expect(uiState.modals[1].id).toBe(id2);
      expect(uiState.modals[2].id).toBe(id3);

      // Close top modal
      uiActions.closeTopModal();
      expect(uiState.modals).toHaveLength(2);
      expect(uiState.modals[1].id).toBe(id2); // Second is now top
    });

    it('should maintain notification order', () => {
      const id1 = uiActions.addNotification('success', 'First');
      const id2 = uiActions.addNotification('info', 'Second');
      const id3 = uiActions.addNotification('warning', 'Third');

      expect(uiState.notifications).toHaveLength(3);
      expect(uiState.notifications[0].id).toBe(id1);
      expect(uiState.notifications[1].id).toBe(id2);
      expect(uiState.notifications[2].id).toBe(id3);
    });
  });
});