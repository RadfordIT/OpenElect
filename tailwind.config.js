/** @type {import('tailwindcss').Config} */
/*eslint-env node*/
module.exports = {
    content: ["./templates/**/*.{html,tmpl}"],
    darkMode: 'selector',
    theme: {
        extend: {},
    },
    plugins: [
        require('@tailwindcss/typography'),
        require('daisyui'),
    ],
}

