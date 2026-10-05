/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./*.html",
    "./admin/*.html",
    "./user/*.html",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#FFF7ED',
          100: '#FFEDD5',
          200: '#FED7AA',
          500: '#F6821F',
          600: '#E06F0E',
          700: '#C25E0A',
        },
        ink: '#1a1d21',
        muted: '#667085',
        line: '#e8eaf0',
        wash: '#fafafa',
        // Material-style colors (from admin dashboard)
        primary: '#F6821F',
        'primary-fixed': '#FFF7ED',
        'on-primary': '#ffffff',
        'on-surface': '#191c1e',
        secondary: '#565e74',
        tertiary: '#006242',
        surface: '#f7f9fb',
        'surface-variant': '#e0e3e5',
        'surface-container': '#eceef0',
        'surface-container-low': '#f2f4f6',
        'surface-container-high': '#e6e8ea',
        outline: '#737686',
        'outline-variant': '#c3c6d7',
      },
      boxShadow: {
        card: '0 1px 3px rgba(0,0,0,0.08)',
        pop: '0 8px 24px rgba(0,0,0,0.12)',
      },
      fontFamily: {
        headline: ['Sora', 'sans-serif'],
        body: ['Inter', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
