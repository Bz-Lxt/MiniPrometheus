/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        bg: '#07090c',
        panel: '#10151c',
        line: '#1d2733',
        amber: '#f5c16c',
        cyan: '#5ee0c5',
        rose: '#ff6b7a',
        mute: '#7d8b99',
      },
      fontFamily: {
        display: ['"IBM Plex Sans Condensed"', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
}
