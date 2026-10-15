export const errors = {
  'errors.generic': 'Something went wrong: {detail}',
  'errors.unreachable': 'Cannot reach the server. Check that FLACidal is running and that the address is correct.',
  'errors.unauthorized': 'Access denied. Enter the access token in the prompt, then try again.',
  'errors.folder.relative': 'The folder must be an absolute path (for example /home/you/Music): {detail}',
  'errors.folder.root': 'The root folder cannot be used. Choose a dedicated folder in Settings: {detail}',
  'errors.folder.missing': 'This folder does not exist. Create it first or choose another one in Settings: {detail}',
  'errors.folder.notDir': 'This path is a file, not a folder. Choose a folder in Settings: {detail}',
  'errors.folder.notWritable': 'FLACidal cannot write in this folder. Check its permissions or choose another folder in Settings: {detail}',
  'errors.queue.noOutputDir': 'No download folder is set. Choose one in Settings, then try again.',
  'errors.queue.notReady': 'The download engine is not ready. Restart the application and try again.',
};
