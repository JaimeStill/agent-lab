import { defineConfig } from 'vite';
import { merge } from './vite.client';
import appConfig from './app/client.config.ts';
import scalarConfig from './scalar/client.config.ts';

export default defineConfig(merge([
  appConfig,
  scalarConfig
]))
