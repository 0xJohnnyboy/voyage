# Spec: Migration CLI vers Cobra (architecture hexagonale)

## 1. Meta
- Date: 2026-05-29
- Statut: DRAFT
- Auteur: Codex + tlambert
- Portée: remplacement du parsing CLI `flag` par Cobra

## 2. Contexte / Problème
Le CLI actuel repose sur `flag` et nécessite du code manuel pour l'aide, notamment pour corriger le rendu des flags courts/longs. Cela augmente le coût de maintenance et crée des régressions UX évitables.

Objectif: migrer vers Cobra (dans `cmd/vo`) avec une séparation claire adapter/hexagone, sans changer le contrat fonctionnel existant.

## 3. Objectifs
- Remplacer totalement l'implémentation CLI `flag` par Cobra.
- Garder compatibilité stricte de tous les flags et alias existants.
- Conserver les sorties (texte/JSON), schémas JSON et codes de sortie actuels.
- Conserver l'UX souhaitée:
  - `vo` sans args: sortie courte + figlet.
  - `vo -h/--help`: aide complète.
- Garder une architecture hexagonale (Cobra = adapter d'entrée).

## 4. Non-objectifs
- Pas de nouveaux sous-commandes.
- Pas de nouveau flag métier.
- Pas de redesign des erreurs/JSON.
- Pas d'ajout de nouveaux tests au-delà de l'existant.

## 5. Scope
In:
- Adapter Cobra dans `cmd/vo`.
- Extraction/maintien d'une orchestration réutilisable hors framework CLI (dans `internal/adapters/cli` ou `internal/cli` selon choix final).
- Mapping strict des flags existants:
  - `-s/--sort`
  - `-f/--format`
  - `-m/--mode`
  - `-l/--long`
  - `-d/--dangling`
  - `-D/--no-dangling`
  - `-L/--log-level`
  - `-t/--tree`
  - `-n/--depth`
  - `-c/--color`
  - `-v/--version`
- Conservation des validations actuelles (ex: `--format json` seulement avec `--tree`).
- Conservation des schémas JSON:
  - succès: `schema_version="1.1.0"`
  - erreur: `schema_version="1.0.0"`
- Conservation des exit codes.

Out:
- Dual-run `flag`+Cobra en parallèle.
- Fallback runtime / feature flag de migration.
- Ajout de nouveaux tests non nécessaires.

## 6. Risques et Impacts
- Régression comportementale subtile dans parsing des flags/args.
- Différences d'UX Cobra par défaut (usage/errors) si non configurées.
- Couplage accidentel de Cobra avec logique métier si boundaries mal posées.

Mitigations:
- Conserver une fonction d'orchestration indépendante de Cobra.
- Configurer Cobra pour reproduire les sorties attendues (`SilenceUsage`, `SilenceErrors`, templates help).
- Exécuter la suite e2e existante complète comme garde de non-régression.

## 7. Approche proposée
- Créer `rootCmd` Cobra dans `cmd/vo`.
- Déplacer la logique de construction des dépendances et exécution vers une couche interne appelable depuis Cobra (pas de types Cobra dans le coeur).
- Mapper les flags Cobra vers une structure d'input interne (équivalent `QueryOptions` + validations CLI).
- Implémenter:
  - sortie courte sur `vo` sans args,
  - aide complète sur `-h/--help`,
  - version `-v/--version`.
- Retirer l'ancien chemin `flag` (remplacement total).

## 8. Alternatives considérées
- Conserver `flag` et améliorer l'aide manuellement:
  - Avantage: pas de dépendance.
  - Inconvénient: maintenance/UX coûteuses, duplication, plus de dette.
  - Décision: rejeté.
- Ajouter Cobra sans extraire l'orchestration:
  - Avantage: migration rapide.
  - Inconvénient: architecture non hexa, logique métier dans adapter.
  - Décision: rejeté.

## 9. Plan d'implémentation
1. Introduire Cobra dans `cmd/vo` (root command, flags, help/version).
2. Extraire/adapter une orchestration interne framework-agnostic.
3. Mapper validations CLI existantes dans la couche d'orchestration ou d'entrée dédiée.
4. Brancher Cobra vers cette orchestration.
5. Supprimer l'ancien chemin `flag`.
6. Mettre à jour README si nécessaire (mention Cobra implicite et usage inchangé).
7. Lancer la suite e2e/tests existants.

## 10. Plan de test
- Exécuter `go test ./...` (incluant `internal/cli` e2e existants).
- Vérifier que les tests actuels valident:
  - compat flags,
  - output tree/flat,
  - JSON succès/erreur,
  - no-args short usage,
  - `-h` help complet,
  - exit codes.

## 11. Critères d'acceptation
- AC-1: le CLI est exécuté via Cobra depuis `cmd/vo`.
- AC-2: tous les flags/alias existants restent compatibles sans changement.
- AC-3: `vo` sans args conserve la sortie courte + figlet.
- AC-4: `vo -h/--help` fournit une aide complète avec rendu correct des flags courts/longs.
- AC-5: sorties métier (flat/tree) restent inchangées fonctionnellement pour mêmes entrées/options.
- AC-6: `--format json --tree` conserve succès en `schema_version="1.1.0"`.
- AC-7: erreurs JSON conservent le format actuel en `schema_version="1.0.0"`.
- AC-8: codes de sortie restent compatibles.
- AC-9: la suite de tests existante passe sans ajout de nouveaux tests obligatoires.
