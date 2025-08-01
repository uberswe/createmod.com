import { defineConfig } from 'vite';
import { resolve, dirname, normalize } from 'path';
import { readdirSync } from 'fs';
import { viteStaticCopy } from 'vite-plugin-static-copy';
import { goTemplateIgnorePlugin } from './plugin/goTemplateIgnorePlugin.js';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function getHtmlEntries(directory) {
    const fullPath = resolve(__dirname, directory);
    return readdirSync(fullPath).reduce((entries, file) => {
        if (file.endsWith('.html')) {
            // Remove the .html extension to get the entry name
            const name = file.replace(/\.html$/, '');
            // Create an entry with the file's absolute path
            entries[name] = resolve(fullPath, file);
        }
        return entries;
    }, {});
}

const promotionEntries = getHtmlEntries('./include/promotional');
const includeEntries = getHtmlEntries('./include');
const rootEntries = getHtmlEntries('.');
const inputEntries = { ...promotionEntries, ...includeEntries, ...rootEntries };

export default defineConfig({
    plugins: [
        goTemplateIgnorePlugin(),
        viteStaticCopy({
            targets: [
                { src: 'node_modules/tinymce/**/*',       dest: 'libs/tinymce' },
                { src: 'node_modules/pocketbase/**/*',    dest: 'libs/pocketbase' },
                { src: 'node_modules/tom-select/**/*',     dest: 'libs/tom-select' },
                { src: 'node_modules/fslightbox/**/*',     dest: 'libs/fslightbox' },
                { src: 'node_modules/plyr/**/*',           dest: 'libs/plyr' },
                { src: 'node_modules/star-rating.js/**/*', dest: 'libs/star-rating' },
                { src: 'node_modules/masonry-layout/dist/**/*', dest: 'libs/masonry-layout' }
            ]
        })
    ],
    build: {
        rollupOptions: {
            input: inputEntries
        }
    },
    optimizeDeps: {
        include: [
            "autosize > commonjs-external",
            "imask > commonjs-external"
        ]
    },
});