import './design/index.css';

import { Router } from '@app/router';

import './views/home-view';
import './views/not-found-view';

const router = new Router('app-content');
router.start();
