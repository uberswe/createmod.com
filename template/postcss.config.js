import { createRequire } from 'module';
const require = createRequire(import.meta.url);

import postcssImport from 'postcss-import';
import autoprefixer from 'autoprefixer';

// Use createRequire to load the CommonJS module.
const purgecss = require('@fullhuman/postcss-purgecss').default;

export default {
    plugins: [
        postcssImport,
        autoprefixer,
        purgecss({
            content: ['./**/*.html', './src/**/*.js'],
        }),
    ],
};