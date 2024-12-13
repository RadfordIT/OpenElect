/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./templates/**/*.{html,tmpl}"],
    plugins: [
        require('@tailwindcss/typography'),
        require('daisyui'),
    ],
    daisyui: {
        themes: [
            {
                ...require("daisyui/src/theming/themes")["light"],
                primary: "blue",
                secondary: "teal",
            }
        ]
    },
}

