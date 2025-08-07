/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Atlassian Design System Colors
        'ds-background': {
          DEFAULT: '#FFFFFF',
          subtle: '#F7F8F9',
          sunken: '#F1F2F4',
          disabled: '#091E4208',
          selected: '#E9F2FF',
          brand: '#0052CC',
        },
        'ds-text': {
          DEFAULT: '#172B4D',
          subtle: '#6B778C',
          subtlest: '#8993A4',
          disabled: '#A5ADBA',
          inverse: '#FFFFFF',
          brand: '#0052CC',
          warning: '#FF8B00',
          danger: '#DE350B',
          success: '#00875A',
        },
        'ds-border': {
          DEFAULT: '#DFE1E6',
          bold: '#8993A4',
          inverse: '#FFFFFF',
          focused: '#0065FF',
          input: '#DFE1E6',
          disabled: '#F4F5F7',
        },
        'ds-icon': {
          DEFAULT: '#42526E',
          subtle: '#6B778C',
          subtlest: '#8993A4',
          disabled: '#A5ADBA',
          inverse: '#FFFFFF',
          brand: '#0052CC',
          warning: '#FF8B00',
          danger: '#DE350B',
          success: '#00875A',
        },
        'ds-surface': {
          DEFAULT: '#FFFFFF',
          overlay: '#FFFFFF',
          raised: '#FFFFFF',
          sunken: '#F4F5F7',
        },
        'ds-brand': {
          'bold': '#0052CC',
          'boldest': '#0747A6',
          'subtle': '#DEEBFF',
          'subtlest': '#F4F8FF',
        },
        'ds-danger': {
          'bold': '#DE350B',
          'boldest': '#BF2600',
          'subtle': '#FFEBE6',
          'subtlest': '#FFF4F1',
        },
        'ds-warning': {
          'bold': '#FF8B00',
          'boldest': '#FF6900',
          'subtle': '#FFF4E6',
          'subtlest': '#FFFAE6',
        },
        'ds-success': {
          'bold': '#00875A',
          'boldest': '#006644',
          'subtle': '#E3FCEF',
          'subtlest': '#F1FDF7',
        },
        'ds-information': {
          'bold': '#0065FF',
          'boldest': '#0747A6',
          'subtle': '#DEEBFF',
          'subtlest': '#F4F8FF',
        },
        'ds-discovery': {
          'bold': '#8777D9',
          'boldest': '#5E4DB2',
          'subtle': '#EAE6FF',
          'subtlest': '#F3F0FF',
        },
      },
      fontFamily: {
        // Atlassian uses system fonts
        'sans': [
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          '"Roboto"',
          '"Oxygen"',
          '"Ubuntu"',
          '"Fira Sans"',
          '"Droid Sans"',
          '"Helvetica Neue"',
          'sans-serif'
        ],
        'mono': [
          '"SFMono-Regular"',
          '"Consolas"',
          '"Liberation Mono"',
          '"Menlo"',
          'monospace'
        ]
      },
      fontSize: {
        // Atlassian typography scale
        'ds-11': ['11px', { lineHeight: '16px' }],
        'ds-12': ['12px', { lineHeight: '16px' }],
        'ds-14': ['14px', { lineHeight: '20px' }],
        'ds-16': ['16px', { lineHeight: '24px' }],
        'ds-20': ['20px', { lineHeight: '24px' }],
        'ds-24': ['24px', { lineHeight: '28px' }],
        'ds-29': ['29px', { lineHeight: '32px' }],
        'ds-35': ['35px', { lineHeight: '40px' }],
      },
      spacing: {
        // Atlassian 4px grid system
        '1': '4px',
        '2': '8px',
        '3': '12px',
        '4': '16px',
        '5': '20px',
        '6': '24px',
        '8': '32px',
        '10': '40px',
        '12': '48px',
        '16': '64px',
        '20': '80px',
        '24': '96px',
      },
      borderRadius: {
        'ds-sm': '3px',
        'ds': '4px',
        'ds-md': '6px',
        'ds-lg': '8px',
      },
      boxShadow: {
        'ds-card': '0 1px 1px rgba(9, 30, 66, 0.25), 0 0 1px rgba(9, 30, 66, 0.31)',
        'ds-overlay': '0 4px 8px -2px rgba(9, 30, 66, 0.25), 0 0 1px rgba(9, 30, 66, 0.31)',
        'ds-raised': '0 1px 1px rgba(9, 30, 66, 0.25), 0 0 1px rgba(9, 30, 66, 0.31)',
      },
      animation: {
        'fade-in': 'fadeIn 0.15s ease-in-out',
        'slide-up': 'slideUp 0.2s ease-out',
        'slide-down': 'slideDown 0.2s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(4px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        slideDown: {
          '0%': { transform: 'translateY(-4px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
}