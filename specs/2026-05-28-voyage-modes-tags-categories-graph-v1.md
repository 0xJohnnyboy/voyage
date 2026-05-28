# Spec: Modes `tags` / `categories` pour navigation graphe

## 1. Meta
- Date: 2026-05-28
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: extension CLI avec modes exclusifs `links|tags|categories`

## 2. Contexte / Problème
Voyage navigue aujourd'hui un graphe basé sur les wikilinks (`links`). Le besoin est d'explorer aussi un graphe sémantique construit depuis le frontmatter (`tags`, `categories`) avec un contrat machine robuste pour `voyage.nvim`.

Le contrat JSON `1.0.0` actuel (`id,label,path,dangling,children`) n'encode pas explicitement le type de nœud. Avec des nœuds attribut (`tag/category`), cela crée une ambiguïté sémantique côté client.

## 3. Objectifs
- Ajouter des modes exclusifs: `links` (défaut), `tags`, `categories`.
- Garder `target` obligatoire dans tous les modes.
- Permettre une exploration tree cohérente basée sur tags/categories.
- Conserver les erreurs JSON structurées actuelles.
- Faire évoluer le schéma JSON de succès sans rupture structurelle brutale.

## 4. Non-objectifs
- Pas de filtres cumulables (`links+tags`, etc.) en V1.
- Pas de mode global sans note cible.
- Pas de filtrage regex des tags/categories en V1.
- Pas de normalisation avancée autre que case-insensitive.

## 5. Scope
In:
- Nouveau flag: `--mode links|tags|categories` (alias `-m`).
- Valeur par défaut: `links`.
- `target` obligatoire inchangé.
- Extraction frontmatter pour `tags` et `categories`:
  - accepter string unique et liste de strings,
  - ignorer silencieusement valeurs vides / non exploitables,
  - matching insensible à la casse.
- Mode flat:
  - `links`: comportement actuel.
  - `tags` / `categories`: sortie groupée par attribut (bloc par tag/catégorie), restreinte à l'exploration issue de la note cible.
- Mode tree:
  - racine = note cible,
  - alternance bipartite: `note -> attribut -> note -> attribut ...`,
  - détection de cycle identique au mode links (arrêt + marqueur cycle en texte),
  - règle de profondeur spécifique: en mode `tags/categories`, `depth=1` inclut le saut complet `attribut + notes associées`.
- JSON succès (`--format json --tree`):
  - schéma unique évolutif, versionné `schema_version: "1.1.0"`.
  - champs racine:
    - `schema_version` (string)
    - `mode` (`links|tags|categories`)
    - `root` (node)
  - champs node:
    - `id`, `label`, `path`, `dangling`, `children` (compat V1)
    - `node_kind` (`note|tag|category`) (nouveau)
  - conventions:
    - nœud attribut: `path=""`, `dangling=false`, `node_kind=tag|category`, `id` préfixé (`tag:`/`category:`)
    - nœud note: `node_kind=note`, `path` absolu
- Erreurs:
  - conserver format JSON structuré actuel en cas d'erreur avec `--format json`.
  - frontmatter invalide: ignorer attributs de la note concernée + warning (sans échec commande).

Out:
- Nouveau schéma JSON séparé radicalement (pas de `2.0.0` en V1).
- Changement du comportement par défaut en `links`.
- Introduction d'un système de filtres combinables.

## 6. Risques et Impacts
- Ambiguïté sémantique de `depth` entre modes.
- Risque de régression côté clients qui valident strictement `schema_version==1.0.0`.
- Explosion du nombre de nœuds sur vaults très tagués.

Mitigations:
- Documenter explicitement la profondeur par mode.
- Ajouter `node_kind` et `mode` pour supprimer l'ambiguïté des nœuds.
- Préfixer `id` des attributs (`tag:`, `category:`) pour éviter collisions.
- Annoncer le bump de schéma vers `1.1.0` pour mettre à jour `voyage.nvim`.

## 7. Approche proposée
- CLI:
  - Ajouter `mode` dans options de query.
  - Validation stricte des valeurs autorisées.
- Indexing:
  - Étendre la note indexée pour contenir `tags` et `categories` normalisés (case-insensitive pour matching, valeur brute conservée pour affichage prioritaire).
- Stratégies de relation:
  - `links`: existante.
  - `tags`: relation via intersection de tags.
  - `categories`: relation via intersection de categories.
- Tree engine:
  - Introduire des nœuds intermédiaires attributs en mémoire de rendu.
  - Appliquer depth par "hop sémantique" (attribut + notes associées = 1 niveau utilisateur en modes tags/categories).
- Flat renderer:
  - Sortie par blocs attribut, puis notes triées selon `--sort`.
- JSON renderer:
  - Évoluer vers schéma `1.1.0`.
  - Conserver les champs V1 et ajouter `mode` + `node_kind`.

## 8. Alternatives considérées
- Conserver strictement `1.0.0` sans nouveaux champs:
  - Avantage: zéro changement de parsing minimal.
  - Inconvénient: ambiguïté durable (`note` vs `tag/category`), heuristiques fragiles côté plugin.
  - Décision: rejeté.
- Nouveau schéma séparé `2.0.0`:
  - Avantage: rupture nette, design libre.
  - Inconvénient: migration plus lourde et double support probable.
  - Décision: rejeté en V1.
- Modes + filtres cumulables dès V1:
  - Avantage: puissance expressive.
  - Inconvénient: tree ambigu, complexité API/UX élevée.
  - Décision: rejeté en V1.

## 9. Plan d'implémentation
1. Ajouter `--mode/-m` et validation CLI.
2. Étendre parsing/index pour `categories` (même logique que tags).
3. Ajouter stratégie de relation pour `tags` et `categories`.
4. Adapter render flat groupé par attribut.
5. Adapter render tree bipartite et logique depth sémantique.
6. Faire évoluer renderer JSON tree vers `schema_version=1.1.0` avec `mode` + `node_kind`.
7. Ajouter tests unitaires + E2E par mode + tests contrat JSON v1.1.0.
8. Mettre à jour README et documenter migration `voyage.nvim`.

## 10. Plan de test
- CLI:
  - `--mode` défaut = `links`.
  - valeurs invalides -> erreur explicite.
- Flat:
  - `--mode tags` et `--mode categories` groupent en blocs attribut.
  - target obligatoire toujours appliquée.
- Tree:
  - alternance `note -> attribut -> note`.
  - cycle détecté et coupé.
  - vérification de la sémantique `depth=1` en mode tags/categories.
- JSON succès:
  - `schema_version="1.1.0"`.
  - présence `mode`.
  - node contient les champs V1 + `node_kind`.
  - attributs avec `id` préfixé, `path=""`, `dangling=false`, `node_kind` correct.
- JSON erreurs:
  - format structuré inchangé.
  - exit code non-zéro inchangé.
- Robustesse:
  - frontmatter invalide: warning + poursuite.
  - catégories/tags vides ignorés silencieusement.

## 11. Critères d'acceptation
- AC-1: `vo --mode links <note>` conserve exactement le comportement existant en sortie texte.
- AC-2: `vo --mode tags <note>` et `vo --mode categories <note>` sont acceptés; toute autre valeur est rejetée.
- AC-3: `target` reste obligatoire pour tous les modes.
- AC-4: en mode flat `tags/categories`, la sortie est groupée par attribut puis notes associées.
- AC-5: en mode tree `tags/categories`, la structure alterne `note -> attribut -> note` avec détection de cycles.
- AC-6: en mode tree `tags/categories`, `depth=1` inclut le niveau attribut et les notes immédiatement associées.
- AC-7: `--format json --tree` renvoie un JSON succès en `schema_version="1.1.0"` avec `mode` au niveau racine.
- AC-8: chaque nœud JSON contient les champs V1 (`id,label,path,dangling,children`) plus `node_kind` (`note|tag|category`).
- AC-9: les nœuds attribut JSON utilisent un `id` préfixé (`tag:` ou `category:`), `path:""`, `dangling:false`.
- AC-10: en cas de frontmatter invalide, la commande n'échoue pas; attributs ignorés pour la note et warning émis.
- AC-11: erreurs en `--format json` conservent le format structuré actuel et code de sortie non-zéro.
