# Spec: Output JSON machine-oriented pour tree (V1)

## 1. Meta
- Date: 2026-05-27
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: contrat JSON strictement versionné pour `vo --format json --tree --depth <N> <note>`

## 2. Contexte / Problème
Le cas d'usage principal est l'intégration avec le plugin Neovim `voyage.nvim`. La sortie texte actuelle est lisible humainement mais fragile pour des parseurs machines. Un contrat JSON stable est nécessaire pour fiabiliser l'intégration.

## 3. Objectifs
- Fournir un format JSON minimal, stable et déterministe pour le mode tree.
- Versionner strictement le schéma dès la V1.
- Exposer explicitement les nœuds dangling et la hiérarchie complète.
- Définir un canal d'erreurs structuré en JSON.

## 4. Non-objectifs
- Pas d'extension aux autres modes que `--tree` en V1.
- Pas d'ajout de limites/perf avancées en V1.
- Pas de masquage/anonymisation des champs `label`/`path`.

## 5. Scope
In:
- Commande cible: `vo --format json --tree --depth <N> <note>`.
- JSON de succès avec:
  - `schema_version` (string)
  - `root` (node)
- Contrat `node`:
  - `id` (string, déterministe)
  - `label` (string)
  - `path` (string, chemin absolu)
  - `dangling` (bool)
  - `children` (array de node)
- Ordre des `children` déterministe selon la stratégie de tri active.
- JSON d'erreur structuré quand `--format json` est demandé.
- Code de sortie non-zéro en cas d'erreur.

Out:
- Contrat JSON pour modes non-tree.
- Nouveaux champs JSON en succès au-delà du minimal ci-dessus.
- Politique de performance/quotas spécifique.

## 6. Risques et Impacts
- Couplage fort côté `voyage.nvim` au schéma V1.
- Breaking change futur coûteux si la V1 est ambiguë.
- Divergence potentielle entre sortie texte et JSON sur cas limites.

Mitigations:
- Version explicite obligatoire (`schema_version`).
- Contrat minimal figé en V1.
- Tests de non-régression sur snapshots JSON.

## 7. Approche proposée
- Schéma de succès V1:
  - `schema_version: "1.0.0"`.
  - `root` toujours présent en cas de succès.
  - `id` déterministe d'un run à l'autre pour la même note.
  - `path` absolu.
  - `dangling=true` pour une référence cassée (note cible introuvable).
  - `children` toujours présent (tableau, potentiellement vide).
- Déterminisme:
  - Pour des inputs et options identiques, l'ordre et le contenu JSON sont identiques.
- Erreurs JSON (si `--format json`):
  - Forme proposée:
    - `schema_version` (string)
    - `error` (object): `code` (string), `message` (string), `details` (object optionnel)
  - Pas de champ `root` en cas d'erreur.
  - Exit code non-zéro conservé.

## 8. Alternatives considérées
- Erreurs non structurées (`stderr` texte uniquement):
  - Avantages: implémentation plus simple, moins de contrat à maintenir.
  - Inconvénients: parsing fragile, UX plugin dégradée, faible testabilité.
- Erreurs JSON structurées (retenu):
  - Avantages: parsing robuste, meilleure observabilité côté plugin, tests stables.
  - Inconvénients: surface de contrat plus large, discipline de versioning nécessaire.

## 9. Plan d'implémentation
1. Formaliser structures Go du payload succès/erreur JSON V1.
2. Brancher le renderer tree JSON sur ces structures.
3. Implémenter chemin d'erreur JSON pour `--format json`.
4. Garantir déterminisme de l'ordre des enfants.
5. Ajouter tests unitaires et tests CLI de non-régression.

## 10. Plan de test
- Succès minimal:
  - présence de `schema_version` et `root`.
  - présence stricte des champs node requis (`id`, `label`, `path`, `dangling`, `children`).
- Sémantique:
  - `path` absolu.
  - `id` déterministe entre 2 runs identiques.
  - `dangling=true` sur référence introuvable.
- Déterminisme:
  - snapshot JSON identique sur exécutions répétées.
- Erreurs:
  - sortie JSON d'erreur valide avec `error.code` et `error.message`.
  - exit code non-zéro.
- Non-régression:
  - modes non-tree inchangés.

## 11. Critères d'acceptation
- AC-1: `vo --format json --tree --depth <N> <note>` renvoie un JSON valide avec `schema_version` et `root` en succès.
- AC-2: chaque nœud JSON contient exactement les champs requis V1: `id`, `label`, `path`, `dangling`, `children`.
- AC-3: `path` est absolu pour tous les nœuds.
- AC-4: `id` est déterministe pour des entrées/options identiques.
- AC-5: une référence cassée est représentée avec `dangling=true`.
- AC-6: en cas d'erreur avec `--format json`, la sortie respecte le format structuré `error{code,message,details?}` et `root` est absent.
- AC-7: en cas d'erreur, la commande termine avec un code non-zéro.
- AC-8: le contrat est explicitement versionné en `schema_version="1.0.0"` et traité comme strict en V1.
- AC-9: les autres modes (hors tree JSON) ne sont pas modifiés par cette V1.
