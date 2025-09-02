# Helios Core Server

Ce dossier contient le serveur JSON-RPC pour Helios Core, organisé de manière modulaire et maintenable.

## Structure des dossiers

### 📁 `middleware/`
Contient tous les composants de middleware pour la gestion des requêtes :

- **`rate_limiter.go`** - Rate limiting par IP
- **`connection_limiter.go`** - Limitation des connexions simultanées
- **`method_tracker.go`** - Suivi des méthodes et temps de réponse
- **`method_rate_limiter.go`** - Rate limiting spécifique par méthode
- **`compute_time_tracker.go`** - Suivi et **prédiction** du temps de calcul par IP
- **`middleware.go`** - Fonctions middleware combinées

### 📁 `routes/`
Contient toutes les routes et handlers HTTP :

- **`rpc_routes.go`** - Routes JSON-RPC principales et monitoring
- **`package.go`** - Export du package

### 📁 `config/`
Configuration du serveur :

- **`config.go`** - Structures de configuration
- **`toml.go`** - Templates TOML
- **`example_app.toml`** - Exemple de configuration

### 📁 `utils/`
Fonctions utilitaires communes :

- **`helpers.go`** - Fonctions d'aide (parsing, etc.)

## Fonctionnalités principales

### 🚦 Rate Limiting
- **Global** : Limite par IP par fenêtre de temps
- **Par méthode** : Limites spécifiques pour `eth_call`, `eth_estimateGas`, etc.
- **Configurable** : Via `app.toml`

### 🔌 Connection Limiting
- Limitation du nombre de connexions simultanées
- Protection contre les attaques DDoS

### 📊 Monitoring
- **`/status`** - État général du serveur
- **`/metrics`** - Métriques détaillées
- **`/reset`** - Réinitialisation des compteurs

### ⚡ Timeout Management
- Timeout configurable par requête
- Annulation propre des requêtes longues
- Pas de crash du serveur principal

### 🎛️ RPC Control
- **`/rpc/control`** - Activation/désactivation dynamique de l'API
- **`/compute-time/reset`** - Réinitialisation du temps de calcul par IP

### 🧠 **PRÉDICTION INTELLIGENTE DU TEMPS DE CALCUL** (NOUVEAU !)
- **Prédiction avant exécution** : Vérifie si une requête dépassera la limite AVANT de l'exécuter
- **Moyennes historiques** : Utilise l'historique des temps d'exécution de chaque méthode
- **Valeur par défaut** : 1 seconde si aucune moyenne n'est disponible
- **Moyenne mobile exponentielle (EMA)** : Donne plus de poids aux exécutions récentes
- **Prévention des timeouts** : Bloque les requêtes qui dépasseraient la limite

## Configuration

### Exemple `app.toml`
```toml
[json-rpc]
# Rate limiting
rate-limit-requests-per-second = 10
rate-limit-window = "1s"

# Connection limiting
max-concurrent-connections = 1000

# Request timeout
max-request-duration = "30s"

# Method-specific limits
method-rate-limits = "eth_call:5,eth_estimateGas:10,eth_getLogs:3"

# Compute time limiting
compute-time-window = "5m"
```

## Utilisation

### Création du serveur
```go
server := NewJSONRPCServer(config, logger, rpcServer)
err := server.Start()
```

### Arrêt du serveur
```go
err := server.Stop()
```

## Avantages de la nouvelle structure

1. **Modularité** : Chaque composant a sa responsabilité
2. **Maintenabilité** : Code organisé et facile à modifier
3. **Testabilité** : Composants isolés et testables
4. **Réutilisabilité** : Middleware réutilisable dans d'autres projets
5. **Clarté** : Structure claire et logique

## Migration depuis l'ancienne structure

L'ancien code monolithique a été divisé en :
- **Middleware** : Logique de gestion des requêtes
- **Routes** : Handlers HTTP et endpoints
- **Config** : Configuration centralisée
- **Utils** : Fonctions communes

## Tests

Chaque composant peut être testé individuellement :
```bash
go test ./middleware/
go test ./routes/
go test ./utils/
```

## 🔬 Fonctionnalité de prédiction du temps de calcul

### Comment ça marche ?

1. **Avant l'exécution** : `PredictComputeTime(ip, method)` vérifie si la requête dépassera la limite
2. **Calcul de la prédiction** : Temps actuel + moyenne historique de la méthode
3. **Décision** : Autorise ou bloque la requête avant exécution
4. **Mise à jour automatique** : Les moyennes sont mises à jour après chaque exécution

### Exemple de prédiction

```go
// IP 192.168.1.100 a déjà utilisé 3 secondes dans la fenêtre
// eth_call a une moyenne historique de 2 secondes
// Limite : 6 secondes par fenêtre

if tracker.PredictComputeTime("192.168.1.100", "eth_call") {
    // 3s + 2s = 5s < 6s → AUTORISÉ
    executeRequest()
} else {
    // 3s + 2s = 5s > 6s → BLOQUÉ
    returnError("Predicted compute time limit exceeded")
}
```

### Avantages de la prédiction

- **Prévention des timeouts** : Bloque les requêtes problématiques avant exécution
- **Meilleure expérience utilisateur** : Réponse immédiate au lieu d'attendre un timeout
- **Protection des ressources** : Évite de consommer des ressources pour des requêtes qui échoueront
- **Apprentissage automatique** : Les moyennes s'adaptent aux changements de performance
- **Configurable** : Valeur par défaut et fenêtre de temps ajustables 