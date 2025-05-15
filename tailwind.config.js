/** @type {import('tailwindcss').Config} */

const colors = require('./config/colors.json').colors;
module.exports = {
    content: ["./templates/**/*.{html,tmpl}", "./config/colors.json"],
    plugins: [
        require('@tailwindcss/typography'),
    ],
}
