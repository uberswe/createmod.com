import fs, {existsSync, readFileSync} from 'fs'
import path, {resolve} from 'path'

export function goTemplateIgnorePlugin() {
    // A Map to store placeholders per file ID
    const placeholdersMap = new Map()
    const partialsDir = 'include'

    return {
        name: 'go-template-ignore',
        order: 'pre',

        load(id) {
            if (id.endsWith('.html')) {
                let code = fs.readFileSync(id, 'utf8')
                const placeholders = []

                code = code.replace(/{{\s*template\s+"([^"]+)"\s*\.\s*}}/g, (match, partialName) => {
                    const filePath = resolve(partialsDir, partialName);
                    if (existsSync(filePath)) {
                        try {
                            return readFileSync(filePath, 'utf8');
                        } catch (err) {
                            console.error(`Error reading partial ${filePath}:`, err);
                            return match;
                        }
                    } else {
                        console.warn(`Partial not found: ${filePath}`);
                        return match;
                    }
                });

                // Replace all occurrences of `{{ ... }}` with placeholders.
                code = code.replace(/\{\{[\s\S]*?\}\}/g, (match) => {
                    placeholders.push(match)
                    return `GO_TEMPLATE_PLACEHOLDER_${placeholders.length - 1}`
                })

                // Compute an absolute file id and embed it as a marker comment.
                const resolvedId = path.resolve(id)
                code += `\n<!-- __FILE_ID__:${resolvedId} -->`
                // Save the placeholders for later restoration.
                placeholdersMap.set(resolvedId, placeholders)
                return code
            }
        },

        transformIndexHtml: {
            order: 'post',
            handler(html, { filename }) {
                // Look for our marker comment.
                const match = html.match(/<!-- __FILE_ID__:(.*?) -->/)
                if (match) {
                    const fileId = match[1].trim()
                    // Remove the marker from the final HTML.
                    html = html.replace(/<!-- __FILE_ID__:(.*?) -->/, '')
                    // If we stored placeholders for this file, restore them.
                    if (placeholdersMap.has(fileId)) {
                        const placeholders = placeholdersMap.get(fileId)
                        html = html.replace(/GO_TEMPLATE_PLACEHOLDER_(\d+)/g, (_, index) => {
                            return placeholders[Number(index)]
                        })
                    }
                }
                return html
            }
        }
    }
}