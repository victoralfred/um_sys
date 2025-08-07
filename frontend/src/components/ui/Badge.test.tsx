import { render, screen } from '@solidjs/testing-library';
import { describe, test, expect } from 'vitest';
import { Badge } from './Badge';

describe('Badge', () => {
  test('renders with default neutral variant', () => {
    render(() => <Badge>Default badge</Badge>);
    
    const badge = screen.getByText('Default badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('badge', 'badge-neutral');
  });

  test('renders success variant correctly', () => {
    render(() => <Badge variant="success">Active</Badge>);
    
    const badge = screen.getByText('Active');
    expect(badge).toHaveClass('badge-success');
    expect(badge).not.toHaveClass('badge-neutral');
  });

  test('renders warning variant correctly', () => {
    render(() => <Badge variant="warning">Pending</Badge>);
    
    const badge = screen.getByText('Pending');
    expect(badge).toHaveClass('badge-warning');
  });

  test('renders danger variant correctly', () => {
    render(() => <Badge variant="danger">Error</Badge>);
    
    const badge = screen.getByText('Error');
    expect(badge).toHaveClass('badge-danger');
  });

  test('renders info variant correctly', () => {
    render(() => <Badge variant="info">Info</Badge>);
    
    const badge = screen.getByText('Info');
    expect(badge).toHaveClass('badge-info');
  });

  test('applies size variants correctly', () => {
    render(() => <Badge size="sm">Small badge</Badge>);
    
    const badge = screen.getByText('Small badge');
    expect(badge).toHaveClass('badge-sm');
  });

  test('applies large size correctly', () => {
    render(() => <Badge size="lg">Large badge</Badge>);
    
    const badge = screen.getByText('Large badge');
    expect(badge).toHaveClass('badge-lg');
  });

  test('shows dot indicator when specified', () => {
    render(() => <Badge variant="success" dot>Online</Badge>);
    
    const badge = screen.getByText('Online');
    const dot = badge.querySelector('.badge-dot');
    
    expect(dot).toBeInTheDocument();
    expect(dot).toHaveClass('badge-dot', 'bg-ds-success-bold');
  });

  test('applies custom class names', () => {
    render(() => <Badge class="custom-badge">Custom</Badge>);
    
    const badge = screen.getByText('Custom');
    expect(badge).toHaveClass('badge', 'custom-badge');
  });

  test('supports interactive badges', () => {
    render(() => <Badge interactive>Clickable</Badge>);
    
    const badge = screen.getByText('Clickable');
    expect(badge).toHaveClass('badge-interactive');
    expect(badge.tagName).toBe('BUTTON');
  });

  test('renders as span by default', () => {
    render(() => <Badge>Static badge</Badge>);
    
    const badge = screen.getByText('Static badge');
    expect(badge.tagName).toBe('SPAN');
  });

  test('handles user status badges correctly', () => {
    render(() => <Badge variant="success" dot>Active</Badge>);
    
    const badge = screen.getByText('Active');
    const dot = badge.querySelector('.badge-dot');
    
    expect(badge).toHaveClass('badge-success');
    expect(dot).toHaveClass('bg-ds-success-bold');
  });

  test('supports accessibility attributes', () => {
    render(() => (
      <Badge 
        variant="warning" 
        aria-label="Account status: pending verification"
        role="status"
      >
        Pending
      </Badge>
    ));
    
    const badge = screen.getByRole('status');
    expect(badge).toHaveAttribute('aria-label', 'Account status: pending verification');
  });

  test('renders with icon when provided', () => {
    render(() => (
      <Badge variant="info">
        <svg data-testid="info-icon" class="w-3 h-3 mr-1">
          <circle cx="12" cy="12" r="10"/>
        </svg>
        With Icon
      </Badge>
    ));
    
    const badge = screen.getByText('With Icon');
    const icon = screen.getByTestId('info-icon');
    
    expect(badge).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
});