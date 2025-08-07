import type { Component, JSX } from 'solid-js';
import { mergeProps, Show, splitProps } from 'solid-js';

export interface CardProps extends JSX.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'elevated' | 'outlined';
  padding?: 'none' | 'sm' | 'md' | 'lg';
  children: JSX.Element;
}

export interface CardHeaderProps extends JSX.HTMLAttributes<HTMLDivElement> {
  title?: string;
  subtitle?: string;
  padding?: 'sm' | 'md' | 'lg';
  children?: JSX.Element;
}

export interface CardBodyProps extends JSX.HTMLAttributes<HTMLDivElement> {
  padding?: 'sm' | 'md' | 'lg';
  children: JSX.Element;
}

export interface CardFooterProps extends JSX.HTMLAttributes<HTMLDivElement> {
  padding?: 'sm' | 'md' | 'lg';
  children: JSX.Element;
}

export const Card: Component<CardProps> = (props) => {
  const merged = mergeProps({ variant: 'default' as const, padding: 'md' as const }, props);
  const [local, others] = splitProps(merged, ['variant', 'padding', 'children', 'class']);

  const cardClass = () => {
    const classes = [
      'card',
      local.variant !== 'default' && `card-${local.variant}`,
      local.padding !== 'md' && `card-padding-${local.padding}`,
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  return (
    <div class={cardClass()} {...others}>
      {local.children}
    </div>
  );
};

export const CardHeader: Component<CardHeaderProps> = (props) => {
  const merged = mergeProps({ padding: 'md' as const }, props);
  const [local, others] = splitProps(merged, ['title', 'subtitle', 'padding', 'children', 'class']);

  const headerClass = () => {
    const classes = [
      'card-header',
      local.padding !== 'md' && `card-header-padding-${local.padding}`,
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  return (
    <div class={headerClass()} {...others}>
      <Show when={local.title} fallback={local.children}>
        <div>
          <h3 class="text-heading-sm text-ds-text">{local.title}</h3>
          <Show when={local.subtitle}>
            <p class="text-body-sm text-ds-text-subtle mt-1">{local.subtitle}</p>
          </Show>
        </div>
      </Show>
    </div>
  );
};

export const CardBody: Component<CardBodyProps> = (props) => {
  const merged = mergeProps({ padding: 'md' as const }, props);
  const [local, others] = splitProps(merged, ['padding', 'children', 'class']);

  const bodyClass = () => {
    const classes = [
      'card-body',
      local.padding !== 'md' && `card-body-padding-${local.padding}`,
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  return (
    <div class={bodyClass()} {...others}>
      {local.children}
    </div>
  );
};

export const CardFooter: Component<CardFooterProps> = (props) => {
  const merged = mergeProps({ padding: 'md' as const }, props);
  const [local, others] = splitProps(merged, ['padding', 'children', 'class']);

  const footerClass = () => {
    const classes = [
      'card-footer',
      local.padding !== 'md' && `card-footer-padding-${local.padding}`,
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  return (
    <div class={footerClass()} {...others}>
      {local.children}
    </div>
  );
};