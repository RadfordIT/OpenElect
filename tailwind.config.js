/** @type {import('tailwindcss').Config} */

const colors = require('./config/colors.json')
module.exports = {
    content: ["./templates/**/*.{html,tmpl}"],
    plugins: [
        require('@tailwindcss/typography'),
        require('daisyui'),
    ],
    "daisyui": {
        "themes": [
            colors
        ]
    }
}
