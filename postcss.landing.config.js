module.exports = ({ env }) => ({
  plugins: {
    'postcss-import': {},
    tailwindcss: { config: './tailwind.landing.config.js' },
    autoprefixer: {},
    ...(env === 'production' ? { cssnano: {} } : {}),
  },
})
