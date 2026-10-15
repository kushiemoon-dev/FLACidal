import type { MessageKey } from '../..';
import { common } from './common';
import { errors } from './errors';
import { files } from './files';
import { history } from './history';
import { home } from './home';
import { modals } from './modals';
import { nav } from './nav';
import { queue } from './queue';
import { search } from './search';
import { settings } from './settings';
import { shell } from './shell';
import { about } from './about';
import { terminal } from './terminal';
import { tools } from './tools';

export const de: Record<MessageKey, string> = { ...common, ...errors, ...files, ...history, ...home, ...modals, ...nav, ...queue, ...search, ...settings, ...shell, ...about, ...terminal, ...tools };
