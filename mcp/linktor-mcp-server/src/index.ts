#!/usr/bin/env node
// ============================================
// Linktor MCP Server - Entry Point
// ============================================

import { runServer, createServer } from './server.js';

// Re-export for programmatic usage
export { runServer, createServer } from './server.js';
export { createHttpServer, startHttpServer } from './http-server.js';

// Run stdio server if called directly
const isMain = import.meta.url === `file://${process.argv[1]}`;
if (isMain) {
  runServer().catch((error) => {
    console.error('Failed to start Linktor MCP Server:', error);
    process.exit(1);
  });
}
