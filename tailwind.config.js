/** @type {import('tailwindcss').Config} */
/*eslint-env node*/
import daisyui from "daisyui"
module.exports = {
  content: ["./templates/**/*.{html,tmpl}"],
  darkMode: 'selector',
  theme: {
    extend: {},
  },
  plugins: [
      daisyui,
  ],
}

