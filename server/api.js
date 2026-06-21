import path from 'node:path';
import express from 'express';
import fs from 'node:fs/promises';

async function readCardMeta(filename) {
  const data = await fs.readFile(filename);
  return JSON.parse(data);
}

export default function() {
  const api = new express.Router();
  api.get('/meta', async (req, res) => {
    const filename = path.join(process.cwd(), 'card', 'card.json');
    try {
      const meta = await readCardMeta(filename);
      res.status(200);
      res.json({
        timestamp: new Date().toISOString(),
        ...meta,
      });
    } catch(error) {
      res.status(500);
      res.json({
        message: `failed to load card meta from ${filename}: ${error}`,
      });
    }
    res.end();
  });
  return api;
}
