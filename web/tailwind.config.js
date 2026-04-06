/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      borderRadius: {
        lg: '0',
        md: '0',
        sm: '0',
      },
      boxShadow: {
        sm: '3px 3px 0px 0px #7c3aed',
        DEFAULT: '5px 5px 0px 0px #7c3aed',
        md: '6px 6px 0px 0px #7c3aed',
        lg: '8px 8px 0px 0px #7c3aed',
        xl: '12px 12px 0px 0px #7c3aed',
      },
      colors: {
        border: '#111827',
        input: '#111827',
        ring: '#7c3aed',
        background: '#f9fafb',
        foreground: '#111827',
        primary: {
          DEFAULT: '#111827',
          foreground: '#ffffff',
        },
        secondary: {
          DEFAULT: '#ffffff',
          foreground: '#111827',
        },
        muted: {
          DEFAULT: '#f3f4f6',
          foreground: '#6b7280',
        },
        card: {
          DEFAULT: '#ffffff',
          foreground: '#111827',
        },
        destructive: {
          DEFAULT: '#ef4444',
          foreground: '#ffffff',
        },
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}
