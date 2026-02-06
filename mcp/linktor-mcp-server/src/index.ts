#!/usr/bin/env node
// ============================================
// Linktor MCP Server - Entry Point
// ============================================

import { runServer } from './server.js';

runServer().catch((error) => {
  console.error('Failed to start Linktor MCP Server:', error);
  process.exit(1);
});
