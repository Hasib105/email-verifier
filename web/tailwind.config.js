/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      borderRadius: {
        lg: '0.75rem',
        md: '0.5rem',
        sm: '0.375rem',
      },
      boxShadow: {
        sm: '0 1px 3px rgba(15, 23, 42, 0.09)',
        DEFAULT: '0 14px 28px rgba(15, 23, 42, 0.08)',
        md: '0 18px 34px rgba(15, 23, 42, 0.1)',
        lg: '0 24px 48px rgba(15, 23, 42, 0.11)',
        xl: '0 34px 64px rgba(15, 23, 42, 0.13)',
      },
      colors: {
        border: '#111827',
        input: '#111827',
        ring: '#1d4ed8',
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
