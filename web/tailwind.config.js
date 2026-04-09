/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      borderRadius: {
        sm: '4px',
        md: '6px',
        lg: '8px',
        xl: '8px',
        '2xl': '8px',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgb(15 23 42 / 0.06)',
        DEFAULT: '0 8px 24px -18px rgb(15 23 42 / 0.35)',
        md: '0 14px 36px -24px rgb(15 23 42 / 0.35)',
        lg: '0 22px 60px -34px rgb(15 23 42 / 0.45)',
        xl: '0 28px 80px -44px rgb(15 23 42 / 0.5)',
      },
      colors: {
        border: '#e5e7eb',
        input: '#d1d5db',
        ring: '#111827',
        background: '#f8fafc',
        foreground: '#0f172a',
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
