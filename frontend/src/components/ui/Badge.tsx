import type { Component, JSX } from 'solid-js';
import { mergeProps, Show, splitProps, Switch, Match } from 'solid-js';

export interface BadgeProps {
  variant?: 'neutral' | 'success' | 'warning' | 'danger' | 'info';
  size?: 'sm' | 'md' | 'lg';
  dot?: boolean;
  interactive?: boolean;
  children: JSX.Element;
  class?: string;
  [key: string]: unknown; // Allow any HTML attributes
}

export const Badge: Component<BadgeProps> = (props) => {
  const merged = mergeProps({ 
    variant: 'neutral' as const, 
    size: 'md' as const,
    interactive: false
  }, props);
  
  const [local, others] = splitProps(merged, [
    'variant', 'size', 'dot', 'interactive', 'children', 'class', 'onClick'
  ]);

  const badgeClass = () => {
    const classes = [
      'badge',
      `badge-${local.variant}`,
      local.size !== 'md' && `badge-${local.size}`,
      local.interactive && 'badge-interactive',
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  const dotClass = () => {
    const dotColors = {
      neutral: 'bg-ds-text-subtle',
      success: 'bg-ds-success-bold',
      warning: 'bg-ds-warning-bold',
      danger: 'bg-ds-danger-bold',
      info: 'bg-ds-information-bold'
    };
    
    return `badge-dot ${dotColors[local.variant]}`;
  };

  const content = () => (
    <>
      <Show when={local.dot}>
        <span class={dotClass()} />
      </Show>
      {local.children}
    </>
  );

  return (
    <Switch>
      <Match when={local.interactive}>
        <button
          class={badgeClass()}
          type="button"
          onClick={local.onClick}
          {...others}
        >
          {content()}
        </button>
      </Match>
      <Match when={!local.interactive}>
        <span class={badgeClass()} {...others}>
          {content()}
        </span>
      </Match>
    </Switch>
  );
};