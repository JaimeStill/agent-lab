import { resolve } from 'path'
import type { ClientConfig } from '../vite.client'
const root = __dirname

const config: ClientConfig = {
  name: 'app',
  aliases: {
    '@app': resolve(root, 'client'),
  },
}

export default config
