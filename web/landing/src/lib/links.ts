/** Where the documentation is published. */
const DOCS_URL = 'https://docs.linktor.dev'

/** Link into the documentation site: docsUrl() for the home, docsUrl('/api') for a page. */
export function docsUrl(path = '') {
  return `${DOCS_URL}${path}`
}
