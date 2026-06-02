/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        brand: { DEFAULT: '#0e7490', dark: '#155e75' },
      },
    },
  },
  plugins: [],
}
