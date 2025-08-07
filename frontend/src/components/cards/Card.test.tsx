import { render, screen } from '@solidjs/testing-library';
import { describe, test, expect } from 'vitest';
import { Card, CardHeader, CardBody, CardFooter } from './Card';

describe('Card', () => {
  test('renders basic card with content', () => {
    render(() => <Card>Card content</Card>);
    
    const card = screen.getByText('Card content');
    expect(card).toBeInTheDocument();
    expect(card).toHaveClass('card');
  });

  test('applies custom class names', () => {
    render(() => <Card class="custom-card">Content</Card>);
    
    const card = screen.getByText('Content');
    expect(card).toHaveClass('card', 'custom-card');
  });

  test('supports different variants', () => {
    render(() => <Card variant="elevated">Elevated card</Card>);
    
    const card = screen.getByText('Elevated card');
    expect(card).toHaveClass('card-elevated');
  });

  test('applies padding variants correctly', () => {
    render(() => <Card padding="lg">Large padding</Card>);
    
    const card = screen.getByText('Large padding');
    expect(card).toHaveClass('card-padding-lg');
  });
});

describe('CardHeader', () => {
  test('renders header with correct classes', () => {
    render(() => <CardHeader>Header content</CardHeader>);
    
    const header = screen.getByText('Header content');
    expect(header).toBeInTheDocument();
    expect(header).toHaveClass('card-header');
  });

  test('supports title and subtitle props', () => {
    render(() => <CardHeader title="Card Title" subtitle="Card subtitle" />);
    
    const title = screen.getByText('Card Title');
    const subtitle = screen.getByText('Card subtitle');
    
    expect(title).toBeInTheDocument();
    expect(subtitle).toBeInTheDocument();
    expect(title).toHaveClass('text-heading-sm');
    expect(subtitle).toHaveClass('text-body-sm', 'text-ds-text-subtle');
  });

  test('renders children when title prop not provided', () => {
    render(() => <CardHeader><h3>Custom header</h3></CardHeader>);
    
    const header = screen.getByRole('heading', { level: 3 });
    expect(header).toBeInTheDocument();
    expect(header.textContent).toBe('Custom header');
  });
});

describe('CardBody', () => {
  test('renders body with correct classes', () => {
    render(() => <CardBody>Body content</CardBody>);
    
    const body = screen.getByText('Body content');
    expect(body).toBeInTheDocument();
    expect(body).toHaveClass('card-body');
  });

  test('applies custom padding', () => {
    render(() => <CardBody padding="sm">Small padding body</CardBody>);
    
    const body = screen.getByText('Small padding body');
    expect(body).toHaveClass('card-body-padding-sm');
  });
});

describe('CardFooter', () => {
  test('renders footer with correct classes', () => {
    render(() => <CardFooter>Footer content</CardFooter>);
    
    const footer = screen.getByText('Footer content');
    expect(footer).toBeInTheDocument();
    expect(footer).toHaveClass('card-footer');
  });
});

describe('Card composition', () => {
  test('renders complete card with all sections', () => {
    render(() => (
      <Card>
        <CardHeader title="Test Card" subtitle="A test card" />
        <CardBody>
          <p>This is the card body with some content.</p>
        </CardBody>
        <CardFooter>
          <button>Action</button>
        </CardFooter>
      </Card>
    ));
    
    expect(screen.getByText('Test Card')).toBeInTheDocument();
    expect(screen.getByText('A test card')).toBeInTheDocument();
    expect(screen.getByText('This is the card body with some content.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
  });

  test('card sections have proper accessibility structure', () => {
    render(() => (
      <Card role="article">
        <CardHeader>
          <h2>Accessible Card</h2>
        </CardHeader>
        <CardBody>
          Card content
        </CardBody>
      </Card>
    ));
    
    const card = screen.getByRole('article');
    const heading = screen.getByRole('heading', { level: 2 });
    
    expect(card).toBeInTheDocument();
    expect(heading).toBeInTheDocument();
  });
});