**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

La nouvelle version de l’application fonctionne dans une seule fenêtre et ne nécessite ni Python, ni Node.js, ni Wails. Elle peut servir de planificateur et de journal sans carte connectée, ou piloter un Arduino Nano par USB.

## Licence et responsabilité

Le code, le firmware, les schémas et la documentation sont publiquement disponibles pour un usage non commercial sous la licence **PolyForm Noncommercial 1.0.0**. Ils peuvent être utilisés, étudiés, modifiés et distribués dans les limites autorisées par cette licence, mais le projet ne peut pas être exploité commercialement sans autorisation distincte de l’auteur. Les détails se trouvent dans le fichier `LICENSE`.

Le projet est fourni pour des expérimentations personnelles et des usages DIY sans garantie. L’utilisateur est responsable du montage correct, des modifications et de la manière dont l’appareil est utilisé. L’auteur n’est pas responsable des dommages matériels, autres pertes ou conséquences d’un montage ou d’une utilisation incorrects et ne garantit aucun effet particulier sur la santé.

## Lancement de l’application

Lancez `Zapper.exe` depuis le dossier de la version portable. Les personnes persistantes et leurs identifiants sont stockés dans `data/persons.json`, les profils actifs dans `data/profiles.json`, et chaque exécution possède son propre fichier dans `data/progress/`. Les exécutions terminées sont déplacées vers des dossiers `data/archive/<id>/` contenant `profile.json` et `progress.json`. Les réglages de la carte sont stockés dans `data/device.json`, tandis que les réglages de l’application, y compris la langue détectée ou sélectionnée, sont stockés dans `data/settings.json`. Les sauvegardes restent dans des sous-dossiers locaux `backups/`. Tout se trouve à côté de l’EXE ; rien n’est écrit dans AppData, Documents ou le Registre Windows.

Dans la vue **Profils**, vous pouvez ajouter des personnes, générer un texte de contexte prêt à copier pour une IA et coller un JSON simplifié renvoyé par un modèle d’IA. Les fréquences dans ce format sont indiquées avec `frequency_hz` ; l’application valide le profil, affiche un aperçu et ne crée un nouveau `run_id` qu’après confirmation. L’exécution active précédente de cette personne est d’abord archivée.

Pendant une session de profil, le bouton **Pause** enregistre la partie restante de l’étape en cours et toutes les étapes suivantes dans la progression locale. La reprise envoie une séquence raccourcie au firmware inchangé et exige à nouveau une confirmation physique sur la carte. **Arrêter** annule la progression partielle et laisse la session complète disponible pour être exécutée de nouveau.

Les sessions ignorées restent dans la file d’attente comme étant en retard. Les règles du programme définissent le nombre de parties, les pauses à l’intérieur d’une série, l’intervalle entre les sessions complètes, le délai de récupération après une session et la compatibilité avec d’autres programmes le même jour. Un profil sans session en retard est archivé automatiquement une fois le plan terminé, tandis que **Terminer le programme** permet de le fermer plus tôt.

## Langue de l’application

Au démarrage, l’application lit la langue de Windows/WebView2 et l’associe à l’une des 30 langues prises en charge. Tant que le réglage reste en mode **Automatique (Windows)**, la détection est effectuée à chaque lancement. Un choix manuel dans le panneau de gauche est enregistré dans `data/settings.json` et désactive les changements automatiques jusqu’à ce que le mode automatique soit sélectionné à nouveau.

La langue de l’application est également la langue par défaut de la variante du firmware. Pour les systèmes d’écriture qu’un LCD1602/HD44780 standard ne peut pas afficher de manière portable, l’application choisit la variante de firmware correspondante avec des textes LCD en anglais ; l’interface de bureau continue d’utiliser la langue sélectionnée.

## Arduino et USB

Le firmware actuel se trouve dans `firmware/zapper_v5/zapper_v5.ino`, avec sa description dans `firmware/zapper_v5/README.md`. Après avoir flashé le firmware :

1. Ouvrez la vue **Appareil**.
2. Sélectionnez le port COM et cliquez sur **Connecter**.
3. Attendez l’état **Prêt**.
4. Envoyez la session du jour ou démarrez une valeur unique en mode manuel.
5. Vérifiez les connexions sur la carte, puis appuyez sur son bouton physique ; la sortie ne démarre qu’à ce moment-là.

Le port sélectionné est mémorisé dans le fichier local `data/device.json`. Les sessions de profil stockent des `device_steps` séparés et précis ; une description telle que « 30 kHz » reste du texte lisible par l’utilisateur, tandis que la carte reçoit `30000000` millihertz et la durée en millisecondes.

### Langues du firmware LCD

Le firmware 5.1.0 possède 30 variantes linguistiques distinctes générées à partir d’une seule base de code. Chaque sketch Arduino ne contient qu’un seul jeu de textes LCD. Les langues utilisant l’alphabet latin disposent de leurs propres textes courts enregistrés en ASCII sûr. Pour le cyrillique et les autres écritures qu’un LCD1602/HD44780 typique ne peut pas afficher de manière portable, la variante correspondante utilise une interface LCD en anglais. La liste complète se trouve dans `firmware/LANGUAGES.md`.

La commande `go run ./tools/firmware_i18n` génère tous les sketches dans `build/generated/firmware/`. Le processus normal `build.ps1` le fait automatiquement et inclut les variantes dans la version portable.

### Flash du firmware depuis l’application

La section **Appareil → Firmware** affiche la version détectée, la dernière version, la langue de la variante du firmware et la langue LCD. L’utilisateur choisit l’ancien ou le nouveau bootloader de l’Arduino Nano et clique explicitement sur **Flasher le firmware** ; l’application ne flashe jamais automatiquement la carte au démarrage.

La compilation et l’envoi sont gérés par `arduino-cli`. Zapper le recherche dans `tools/arduino-cli/`, à côté de l’EXE, dans `PATH` et dans les emplacements habituels d’Arduino IDE. Si l’outil n’est pas disponible, l’application l’indique clairement et le bouton de flash reste désactivé. La compilation nécessite également que le core `arduino:avr` et la bibliothèque `LiquidCrystal_I2C` soient disponibles pour l’installation `arduino-cli` utilisée.

### Détection de la langue et sélection du firmware

Au démarrage, l’application lit la langue de l’environnement WebView2/Windows (`navigator.languages`) et l’associe à l’un des 30 codes pris en charge. Si la langue du système n’est pas prise en charge, l’anglais est sélectionné. En mode **Automatique (Windows)**, la langue est vérifiée à chaque lancement ; une sélection manuelle est enregistrée dans `data/settings.json` jusqu’à ce que le mode automatique soit réactivé.

Le même code de langue est utilisé par défaut sur l’écran de flash du firmware. Pour les langues non prises en charge par LCD1602, l’application sélectionne tout de même la variante correspondant à la langue de l’utilisateur, mais indique que le texte LCD sera en anglais. Le firmware n’est jamais flashé automatiquement au démarrage de l’application ; le flash exige un clic explicite de l’utilisateur afin de ne pas écraser accidentellement un autre programme déjà présent sur l’Arduino.

## Compilation

Go est requis. Le plus simple est d’exécuter ceci à la racine du projet :

```text
build.bat
```

Ou, dans PowerShell :

```powershell
.\build.ps1
```

Le script exécute les tests et l’analyse du code, construit `build/generated/Zapper-dev.exe` et prépare le portable `build/Zapper/Zapper.exe` sans fenêtre de console.

## Structure du projet

- `app/` — code Go, interface HTML/CSS/JS, guide et base de fréquences.
- `firmware/zapper_v5/` — firmware Arduino actuel.
- `data/` — profils actifs, progression, archives, réglages de l’appareil et sauvegardes automatiques.
- `locales/` — traductions versionnées de l’interface et du guide, utilisées en développement et copiées dans les versions publiées.
- `build/Zapper/` — version portable prête à être copiée sur un autre ordinateur.