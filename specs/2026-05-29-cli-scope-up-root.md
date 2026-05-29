# Spec: Scope d'indexation CLI (`--scope up:N|root:<path>`)

## 1. Meta
- Date: 2026-05-29
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: extension du CLI pour contrôler la racine d'indexation

## 2. Contexte / Problème
Actuellement, Voyage indexe le dossier parent direct de la note cible. Pour des notes imbriquées, cela tronque le graphe: les relations vers des notes situées dans des dossiers parents (ou leurs sous-dossiers) ne sont pas visibles.

Le besoin est d'élargir explicitement l'espace d'indexation sans changer la racine logique de la requête (la note cible reste le nœud racine de sortie).

## 3. Objectifs
- Ajouter un contrôle explicite du scope d'indexation via un flag unique.
- Éviter les graphes tronqués pour les notes imbriquées.
- Préserver la compatibilité par défaut (`up:0`).
- Garder un comportement prévisible sans auto-détection implicite.

## 4. Non-objectifs
- Pas d'auto-détection de "vault root".
- Pas de changement de la note racine de rendu/requête.
- Pas d'indexation globale implicite.

## 5. Scope
In:
- Nouveau flag: `--scope <value>`.
- Valeurs supportées:
  - `up:N` avec `N >= 0`.
  - `root:<path>`.
- Valeur par défaut: `up:0` (comportement actuel).
- Résolution:
  - `up:N`: partir du dossier parent de la note cible et remonter `N` niveaux max (arrêt naturel à `/`).
  - `root:<path>`: `path` relatif résolu depuis le `cwd`.
- Validation:
  - format invalide => erreur utilisateur explicite.
  - `N < 0` => erreur utilisateur explicite.
  - en mode `root:<path>`, la note cible doit être incluse dans `root`; sinon erreur.
- Effet fonctionnel:
  - scope appliqué à tous les modes (`links|tags|categories`), formats (flat/tree/json), options de tri/profondeur.
  - la note cible reste la racine logique de la requête/rendu.

Out:
- Détection automatique de root (`.git`, `.obsidian`, etc.).
- Multiples flags de scope concurrents.

## 6. Risques et Impacts
- Coût CPU/IO supérieur avec scopes larges.
- Bruit informationnel accru (plus de notes candidates, plus de relations visibles).
- Erreurs de configuration utilisateur (scope invalide / root non cohérent).

Mitigations:
- Valeur par défaut conservatrice (`up:0`).
- Erreurs explicites de validation.
- Documentation claire des exemples et tradeoffs perf.

## 7. Approche proposée
- Ajouter `Scope string` à la config CLI/options query.
- Implémenter un résolveur de scope:
  - parse `up:N` et `root:<path>`.
  - retourne le chemin racine d'indexation effectif.
- Avant indexation, calculer la racine effective selon scope.
- Vérifier la contrainte `target ∈ root` pour `root:<path>`.
- Réutiliser ensuite le flux existant (`indexer.Build(root)` + query/render).

## 8. Alternatives considérées
- Auto-detection de root:
  - Avantage: UX "magique".
  - Inconvénient: implicite, parfois trop large, peu prévisible.
  - Décision: rejetée.
- Flag booléen simpliste (ex: "include-parents"):
  - Avantage: simple.
  - Inconvénient: manque de contrôle granulaire, ambigu.
  - Décision: rejeté.

## 9. Plan d'implémentation
1. Ajouter `--scope` (défaut `up:0`) dans le CLI Cobra.
2. Ajouter parse/validation scope (`up:N`, `root:<path>`).
3. Implémenter résolution du root effectif d'indexation.
4. Appliquer la contrainte `target ∈ root` pour `root:<path>`.
5. Brancher le root effectif dans `indexer.Build(...)`.
6. Ajouter tests E2E (scope up/root + erreurs).
7. Mettre à jour README (usage + exemples).

## 10. Plan de test
- Compat défaut:
  - sans `--scope`, comportement identique à aujourd'hui.
- `up:N`:
  - `up:0` == comportement actuel.
  - `up:1`, `up:2` élargissent les relations visibles au-dessus.
  - arrêt naturel à `/` sans panic.
- `root:<path>`:
  - path relatif résolu depuis `cwd`.
  - target dans root => OK.
  - target hors root => erreur explicite.
- Validation:
  - `--scope up:-1` => erreur.
  - `--scope up:abc` => erreur.
  - `--scope foo` => erreur.
- Non-régression:
  - modes `links|tags|categories`, format JSON et erreurs JSON inchangés hors effet scope.

## 11. Critères d'acceptation
- AC-1: `--scope` supporte `up:N` et `root:<path>` avec défaut `up:0`.
- AC-2: `up:0` reproduit le comportement actuel.
- AC-3: `up:N` remonte de N niveaux max avec arrêt naturel à `/`.
- AC-4: `root:<path>` accepte un path relatif depuis `cwd`.
- AC-5: en mode `root:<path>`, si target n'est pas sous root, la commande échoue avec erreur explicite.
- AC-6: la note ciblée reste la racine logique de sortie/requête, quel que soit le scope.
- AC-7: le scope s'applique à tous les modes (`links|tags|categories`) et formats supportés.
- AC-8: formats invalides de `--scope` renvoient une erreur utilisateur claire.
- AC-9: README documente le flag et des exemples `up`/`root`.
