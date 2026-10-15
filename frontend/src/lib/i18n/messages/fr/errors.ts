export const errors = {
  'errors.generic': 'Une erreur est survenue : {detail}',
  'errors.unreachable': 'Impossible de joindre le serveur. Vérifiez que FLACidal est lancé et que l\'adresse est correcte.',
  'errors.unauthorized': 'Accès refusé. Saisissez le jeton d\'accès dans la fenêtre prévue, puis réessayez.',
  'errors.folder.relative': 'Le dossier doit être un chemin absolu (par exemple /home/vous/Musique) : {detail}',
  'errors.folder.root': 'Le dossier racine ne peut pas être utilisé. Choisissez un dossier dédié dans les Paramètres : {detail}',
  'errors.folder.missing': 'Ce dossier n\'existe pas. Créez-le d\'abord ou choisissez-en un autre dans les Paramètres : {detail}',
  'errors.folder.notDir': 'Ce chemin est un fichier, pas un dossier. Choisissez un dossier dans les Paramètres : {detail}',
  'errors.folder.notWritable': 'FLACidal ne peut pas écrire dans ce dossier. Vérifiez ses droits ou choisissez un autre dossier dans les Paramètres : {detail}',
  'errors.queue.noOutputDir': 'Aucun dossier de téléchargement n\'est défini. Choisissez-en un dans les Paramètres, puis réessayez.',
  'errors.queue.notReady': 'Le moteur de téléchargement n\'est pas prêt. Redémarrez l\'application et réessayez.',
};
