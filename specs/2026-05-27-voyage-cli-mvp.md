# Spec: voyage-cli MVP (moteur CLI de navigation relationnelle Markdown)

## 1. Meta
- Date: 2026-05-27
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: MVP CLI (sans TUI, sans watch, sans DB)

## 2. Contexte / Problème
Le besoin est un CLI scriptable orienté relations entre notes Markdown, analogue à `ls` mais appliqué à un graphe de notes. Aujourd’hui, le parcours relationnel n’existe pas: il faut scanner un répertoire récursivement, extraire les relations, et lister rapidement les notes liées à une note cible.

## 3. Objectifs
- Fournir une commande `vo <path-note>` qui affiche les notes liées sortantes.
- Construire un index en mémoire à partir du répertoire racine de la note cible (dossier parent + sous-répertoires).
- Supporter wikilinks et tags au parsing, avec backlinks calculés.
- Fournir une sortie scriptable avec options de formatage et tri.
- Concevoir une architecture extensible pour futures stratégies de relations et futurs adapters FS.

## 4. Non-objectifs
- Pas de TUI (BubbleTea reporté).
- Pas de mode watch (`-w/--watch` reporté).
- Pas d’édition de notes.
- Pas de DB/SQLite/cache persistant.
- Pas de sync cloud, IA, semantic search, plugins.

## 5. Scope
In:
- CLI Go `vo`.
- Entrée cible: chemin de fichier Markdown uniquement.
- Scan récursif depuis le répertoire parent de la note cible.
- Parsing Markdown:
  - frontmatter YAML (title, tags) avec warning si invalide,
  - wikilinks `[[...]]`.
- Résolution wikilinks:
  - par `title` et par nom de fichier (sans extension),
  - avec règle de priorité explicite (voir section approche).
- Calcul backlinks en mémoire.
- Affichage par défaut des liens sortants.
- Affichage des liens non résolus par défaut.
- Option “silencieux dangling links”.
- Ordre par défaut: découverte.
- Option de tri alphabétique.
- Options de format de sortie (liste simple + format détaillé humain lisible).
- Niveaux de logs (`silent`, `warn`, `debug`).
- Compat cible MVP: Linux/macOS.
- Architecture découplée (cœur indépendant CLI et FS concret).

Out:
- Acceptation d’identifiants logiques en entrée (hors chemin).
- Sélection runtime de stratégies relations autres que sortants.
- Watch FS en temps réel.
- Support Windows garanti.
- TUI, Neovim plugin.

## 6. Risques et Impacts
- Ambiguïté de résolution wikilinks si plusieurs notes candidates (même title ou même basename).
- Frontmatter invalide fréquent: risque de bruit logs/warnings.
- Performance sur gros corpus Markdown: coût scan + parsing startup.
- Dépendance implicite au FS local si abstractions mal posées.

Mitigations:
- Politique de résolution déterministe + warning en cas d’ambiguïté.
- Parsing tolérant: extraire ce qui est possible et continuer.
- API interne orientée interfaces (hexagonal): parser/index/service/FS.

## 7. Approche proposée
Architecture (hexagonale légère):
- Domain:
  - `Note` (ID, Title, Path, Tags, Links, Backlinks, DanglingLinks éventuel).
  - `GraphIndex` (accès notes + relations).
- Ports:
  - `NoteRepository` (list/read notes depuis une racine).
  - `Logger`.
  - `OutputFormatter`.
  - `RelationStrategy` (MVP: `OutgoingLinksStrategy`).
- Application:
  - `IndexerService` (scan, parse, résolution, backlinks).
  - `QueryService` (résoudre cible, appliquer stratégie, trier).
- Adapters:
  - FS local (os + filepath).
  - Parser Markdown/frontmatter/wikilinks.
  - CLI flags + formatters texte.

Règles fonctionnelles MVP:
- Entrée `vo <path-note>`:
  - vérifie existence et extension Markdown (`.md`, configurable plus tard).
  - racine scan = dossier parent de la note cible.
- Résolution wikilinks:
  1. match exact `title` (si unique),
  2. sinon match basename fichier sans extension (si unique),
  3. sinon non résolu + warning `warn`.
- Dangling links:
  - affichés en sortie par défaut avec marqueur stable (ex: `[dangling] <label>`),
  - masquables via flag dédié (ex: `--no-dangling`).
- Tri:
  - défaut: ordre découverte,
  - option: alphabétique (`--sort alpha`).
- Format sortie:
  - défaut: une ligne par relation, lisible/scriptable,
  - détaillé: chemin + taille human-readable + date modif (style inspiré `ls`, sans promesse de format byte-perfect).
