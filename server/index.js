#!/usr/bin/env node
import path from 'node:path';
import express from 'express';
import api from './api.js';

const PORT = process.env.PORT || 3000;
const dist = path.join(process.cwd(), "dist");

const app  = express();
app.get('/', (req, res) => {
  res.sendFile(path.join(dist , 'index.html'));
  res.end();
});
app.get('/card', (req, res) => {
  res.sendFile(path.join(dist, 'card', 'index.html'));
  res.end();
})

app.use('/api/v1', api());
app.use(express.static(dist));
app.listen(PORT, () => {
  console.log(`server running on http://localhost:${PORT}`);
});
