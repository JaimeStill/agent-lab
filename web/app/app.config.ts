import { resolve } from 'path'
import type { ClientConfig } from '../vite.client'
const root = __dirname

const config: ClientConfig = {
  name: 'app',
  aliases: {
    '@app/design': resolve(root, 'client/design'),
    '@app/core': resolve(root, 'client/core'),
    '@app/components': resolve(root, 'client/components'),
  },
}

export default config
