# Spec: voyage-cli tree view + depth (MVP extension)

## 1. Meta
- Date: 2026-05-27
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: extension CLI `tree` et `depth`

## 2. Contexte / Problème
La sortie actuelle est plate (liste des liens sortants). Pour explorer des relations multi-niveaux, il faut une vue hiérarchique avec profondeur contrôlable.

## 3. Objectifs
- Ajouter une vue arborescente des relations sortantes.
- Ajouter une option de profondeur pour expansion récursive.
- Garder le comportement existant inchangé hors mode tree.

## 4. Non-objectifs
- Pas de nouvelle stratégie de relation (reste sortant).
- Pas de mode interactif/TUI.
- Pas de cache persistant.

## 5. Scope
In:
- Nouveau flag `--tree` (optionnel alias court à discuter ultérieurement).
- Nouveau flag `--depth <n>`.
- `--depth` valide uniquement en mode `--tree`.
- Valeur par défaut `depth=1`.
- `depth=0` rejeté (erreur utilisateur).
- En mode tree, affichage hiérarchique des liens sortants.
- Gestion des cycles: arrêt à la première revisite sur la branche courante + marqueur `(cycle)`.
- Déduplication: aucune dédup globale; une note peut réapparaître sur plusieurs branches.
- Dangling en tree: affichés comme `[dangling] <label>` si dangling activé; masqués sinon.
- Tri en tree:
  - `--sort alpha` tri par niveau.
  - `--sort discovery` conserve l’ordre découverte par niveau.
- Option `--long` applicable en mode tree (ajoute métadonnées par nœud résolu: taille/date/path).

Out:
- Changement du comportement par défaut hors `--tree`.
- Support de `--depth` hors `--tree`.

## 6. Risques et Impacts
- Sortie potentiellement volumineuse avec profondeur élevée.
- Cycles: risque de boucles si garde incorrecte.
- Lisibilité de l’arbre en mode `--long`.

Mitigations:
- Validation stricte des flags (`--depth` only with `--tree`, `depth>=1`).
- Détection cycle par chemin de visite (branch-local visited set).
- Format tree stable et testable.

## 7. Approche proposée
- CLI:
  - Ajouter `--tree` (bool), `--depth` (int, défaut 1).
  - Validation:
    - si `--depth` explicitement fourni sans `--tree` => erreur.
    - si `--depth < 1` => erreur.
  - `--tree` override `--format` (liste/detailed ignorés pour la forme générale, `--long` reste pris en compte pour enrichir chaque ligne tree).
- App layer:
  - Ajouter un renderer tree basé sur DFS bornée par `depth`.
  - Cycle marker quand revisite d’un ID déjà présent dans la pile courante.
  - Réutiliser stratégie sortante et options `sort`, `dangling`, `long`.
- Output:
  - Connecteurs ASCII (`|-`, ``-`) pour compat script/term.
  - Dangling: `[dangling] label`.

## 8. Alternatives considérées
- `--format tree` au lieu de `--tree`: rejeté pour garder un switch explicite et prioritaire.
- Dédup globale: rejetée (perte d’information de chemins multiples).

## 9. Plan d'implémentation
1. Étendre parsing flags CLI (`--tree`, `--depth`) + validations.
2. Ajouter structure interne pour rendu tree avec profondeur/cycles.
3. Implémenter renderer tree (normal + long).
4. Intégrer tri/dangling dans expansion par niveau.
5. Ajouter tests unitaires/E2E sur corpus de notes.

## 10. Plan de test
- Tests flags:
  - `--tree` seul (depth=1 implicite).
  - `--tree --depth 2`.
  - `--depth 2` sans tree => erreur.
  - `--tree --depth 0` => erreur.
- Tests fonctionnels tree:
  - profondeur 1 vs 2.
  - cycle marqué et coupé.
  - dangling visible/masqué.
  - tri alpha par niveau.
  - `--long` en tree ajoute métadonnées.
- Non-régression:
  - sortie non-tree inchangée.

## 11. Critères d'acceptation
- AC-1: `vo --tree <note>` affiche un arbre des liens sortants directs (profondeur 1 par défaut).
- AC-2: `vo --tree --depth N <note>` (N>=1) explore récursivement jusqu’à N niveaux.
- AC-3: `vo --depth N <note>` sans `--tree` renvoie une erreur utilisateur explicite.
- AC-4: les cycles sont marqués `(cycle)` et n’entrent pas en récursion infinie.
- AC-5: les notes peuvent réapparaître sur des branches différentes (pas de dédup globale).
- AC-6: dangling affichés en tree avec `[dangling]` si `--dangling`, absents si `--no-dangling`.
- AC-7: `--sort alpha` trie les enfants à chaque niveau en mode tree.
- AC-8: `--long` enrichit les lignes tree avec métadonnées (taille/date/path).
- AC-9: sans `--tree`, comportement CLI existant inchangé.
