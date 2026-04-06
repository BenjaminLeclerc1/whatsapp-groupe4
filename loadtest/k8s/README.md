# k6 distribué sur Kubernetes (recommandation prof : répartir la charge)

Ce dossier permet de lancer le script `loadtest/k6/benchmark_10k_health.js` avec **plusieurs pods k6 en parallèle** via le **Grafana k6 Operator**. Chaque runner reçoit une **fraction des VUs** définis dans le script (voir doc officielle : *parallelism*).

## Prérequis

- Un cluster Kubernetes (ex. **AKS**, ou **kind** / **minikube** pour essayer).
- `kubectl` et idéalement **Kustomize** (intégré à `kubectl apply -k`).
- Le **k6 Operator** installé (souvent après **cert-manager**). Référence : [Set up k6 operator](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/k6-operator/).

Exemple d’installation operator (à adapter : version, namespace) :

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install k6-operator grafana/k6-operator --namespace k6-operator-system --create-namespace
```

## Configurer l’URL cible

Ouvrir `testrun.yaml` et remplacer la valeur de `BASE_URL` (FQDN HTTPS de la gateway Azure, **sans** `/` final).

Ou après coup :

```bash
kubectl patch testrun whatsapp-gateway-health -n k6-loadtest --type merge -p \
  '{"spec":{"runner":{"env":[{"name":"BASE_URL","value":"https://votre-app....azurecontainerapps.io"}]}}}'
```

## Déployer le ConfigMap + TestRun

À la **racine du dépôt** (le `kustomization.yaml` est dans `loadtest/` pour inclure le script sous `k6/`) :

```bash
kubectl apply -k loadtest
```

Cela crée le namespace `k6-loadtest`, le ConfigMap du script (via Kustomize) et le `TestRun`.

## Suivi

```bash
kubectl get testruns,pods,jobs -n k6-loadtest
kubectl logs -n k6-loadtest -l job-name -f --tail=100
```

Avec `cleanup: post`, les ressources du run sont nettoyées à la fin. Pour voir le résumé k6, lancez `kubectl logs -f` sur un pod **runner** pendant qu’il est `Running` (après coup les pods peuvent avoir disparu).

## 10k VUs (défaut)

`kubectl apply -k loadtest` applique `testrun.yaml` (~10k VUs au pic, `parallelism: 4`). Sur **kind** à un nœud, `separate: false` est nécessaire pour éviter des pods `Pending`.

## 100k VUs (reco prof : beaucoup de générateurs + cluster dimensionné)

1. Mettre à jour le script dans le cluster : `kubectl apply -k loadtest` (régénère le ConfigMap).
2. Un seul `TestRun` à la fois :
   ```bash
   kubectl delete testrun whatsapp-gateway-health -n k6-loadtest --ignore-not-found
   kubectl apply -f loadtest/k8s/testrun-100k.yaml
   ```
3. Sur **AKS multi-nœuds** : dans `testrun-100k.yaml`, augmenter `parallelism` (ex. 16–32), repasser `separate: true`, et monter `resources` si les pods restent en `OOMKilled` ou `Pending`.

Le script lit les variables `VUS_*`, `RAMP_*`, `HOLD`, `VUS_MAX` (voir `benchmark_10k_health.js`). `STRICT_THRESHOLDS=false` évite d’échouer le run uniquement à cause des seuils latence quand la charge est extrême.

## Ajuster la parallélisation

Dans `testrun.yaml`, modifier `spec.parallelism` (ex. `8` ou `16` pour plus de générateurs de charge). Surveillez les quotas CPU du cluster et la capacité réelle de **l’application** testée : distribuer k6 ne remplace pas le scale-out de la gateway.

## Lien avec le reste du projet

- **Local / CI** : `loadtest/docker-compose.k6.yml` ou `docker run ... grafana/k6`.
- **Prod Azure** : cible du test = URL sortie Terraform (`terraform output container_apps` → `api_gateway_fqdn`).
