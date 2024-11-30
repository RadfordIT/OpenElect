/** @type {import('tailwindcss').Config} */
/*eslint-env node*/
import daisyui from "daisyui"
export default {
  content: ["./templates/**/*.tmpl"],
  theme: {
    extend: {},
  },
  plugins: [
      daisyui,
  ],
}

