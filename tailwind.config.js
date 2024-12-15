/** @type {import('tailwindcss').Config} */

const colors = require('./config/colors.json').colors;
module.exports = {
    content: ["./templates/**/*.{html,tmpl}"],
    plugins: [
        require('@tailwindcss/typography'),
        require('daisyui'),
    ],
    daisyui: {
        themes: [
            {
                mytheme: {
                    "primary": colors.primary,
                    "primary-content": colors.primaryContent,
                    "secondary": colors.secondary,
                    "secondary-content": colors.secondaryContent,
                    "accent": colors.accent,
                    "accent-content": colors.accentContent,
                    "neutral": colors.neutral,
                    "neutral-content": colors.neutralContent,
                    "base-100": colors.base100,
                    "base-200": colors.base200,
                    "base-300": colors.base300,
                    "base-content": colors.baseContent,
                    "info": colors.info,
                    "info-content": colors.infoContent,
                    "success": colors.success,
                    "success-content": colors.successContent,
                    "warning": colors.warning,
                    "warning-content": colors.warningContent,
                    "error": colors.error,
                    "error-content": colors.errorContent
                }
            }
        ]
    }
}
