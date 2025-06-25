# RPC Cache System

Ce système de cache a été implémenté pour optimiser les performances des requêtes RPC dans helios-chain.

## Fonctionnalités

- **Cache LRU** : Utilise un cache LRU (Least Recently Used) pour gérer automatiquement la mémoire
- **Expiration automatique** : Chaque entrée a un TTL (Time To Live) configurable
- **Génération de clés sécurisée** : Utilise SHA-256 pour générer des clés uniques
- **Thread-safe** : Support complet pour les accès concurrents
- **Statistiques** : Fournit des statistiques d'utilisation du cache
- **Proxy d'API** : `CachedPublicAPI` qui proxifie directement les services RPC
- **Configuration simplifiée** : 4 méthodes hardcodées avec TTL de 15 secondes

## Méthodes mises en cache

Le système de cache est configuré pour les 5 méthodes suivantes avec un TTL de 15 secondes :

- `eth_getAllHyperionTransferTxs`
- `eth_getHyperionAccountTransferTxsByPageAndSize`
- `eth_getValidatorWithHisAssetsAndCommission`
- `eth_getAllWhitelistedAssets`
- `eth_getLastTransactionsInfo`

## Utilisation

### Initialisation

```go
import "helios-core/helios-chain/rpc/cache"

// Créer un nouveau cache avec 1000 entrées et TTL par défaut de 30 secondes
rpcCache, err := cache.NewRPCCache(1000, 30*time.Second)
if err != nil {
    // Gérer l'erreur
}
```

### Proxy d'API (Recommandé)

Le `CachedPublicAPI` proxifie directement les services RPC et intercepte automatiquement les 4 méthodes configurées :

```go
// Dans apis.go - configuration automatique
func createEthAPI(ctx *server.Context, clientCtx client.Context, allowUnprotectedTxs bool, indexer types.EVMTxIndexer) []rpc.API {
    evmBackend := backend.NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
    
    return []rpc.API{
        {
            Namespace: "eth",
            Version:   "1.0",
            Service:   eth.NewCachedPublicAPI(ctx.Logger, evmBackend), // Proxy d'API
            Public:    true,
        },
    }
}
```

**Avantages du proxy d'API :**
- ✅ **Interception au niveau service** : Cache appliqué directement au niveau RPC
- ✅ **Transparence totale** : Aucune modification du code backend nécessaire
- ✅ **Interface identique** : Même interface que l'API originale
- ✅ **Configuration simplifiée** : 5 méthodes hardcodées avec 15 secondes de TTL
- ✅ **Performance optimale** : Cache appliqué au niveau le plus haut
- ✅ **Maintenance facile** : Pas de code de cache dispersé
- ✅ **Compatibilité parfaite** : Remplace directement l'API originale
- ✅ **Cache indépendant** : Chaque service a son propre cache

### Approche d'interception automatique

L'approche d'interception automatique rend l'utilisation du cache complètement transparente :

```go
func (b *Backend) GetMyData(param string) (interface{}, error) {
    // Le cache est appliqué automatiquement selon la configuration
    result, err := b.interceptCall("GetMyData", []interface{}{param}, func() (interface{}, error) {
        return b.fetchMyData(param)
    })
    
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### Approche transparente manuelle

La nouvelle approche transparente rend l'utilisation du cache presque invisible :

```go
func (b *Backend) GetMyData(param string) (interface{}, error) {
    cacheKey := cache.GenerateKey("GetMyData", param)
    
    // Vérifier le cache
    if b.hasCachedReturn(cacheKey, "30s") {
        return b.cacheOf(cacheKey), nil
    }
    
    // Exécuter la requête coûteuse
    result, err := b.fetchMyData(param)
    if err != nil {
        return nil, err
    }
    
    // Mettre en cache et retourner
    return b.cacheReturn(cacheKey, result, "30s"), nil
}
```

### Approche générique avec `withCache`

La méthode `withCache` simplifie l'ajout de cache à n'importe quelle fonction RPC :

```go
func (b *Backend) GetMyData(param string) (interface{}, error) {
    cacheKey := cache.GenerateKey("GetMyData", param)
    
    result, err := b.withCache(cacheKey, func() (interface{}, error) {
        return b.fetchMyData(param)
    }, 5*time.Minute) // TTL optionnel
    
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### TTL par type de requête

- **Blocs historiques** : 5 minutes
- **Bloc latest** : 5 secondes  
- **Bloc pending** : 1 seconde
- **BlockNumber** : 2 secondes
- **Transactions** : 5-10 minutes
- **Balances** : 10-30 secondes
- **Codes de contrats** : 5-10 minutes

## Configuration

### Configuration simplifiée

Le système utilise une configuration hardcodée pour les 5 méthodes spécifiques :

```go
// Dans cache.go - configuration automatique
cachedMethods := map[string]time.Duration{
    "GetAllHyperionTransferTxs": 15 * time.Second,
    "GetHyperionAccountTransferTxsByPageAndSize": 15 * time.Second,
    "GetValidatorWithHisAssetsAndCommission": 15 * time.Second,
    "GetAllWhitelistedAssets": 15 * time.Second,
    "GetLastTransactionsInfo": 15 * time.Second,
}
```

### Taille du cache

La taille du cache détermine le nombre maximum d'entrées stockées. Une taille de 1000-2000 entrées est recommandée pour la plupart des cas d'usage.

### TTL (Time To Live)

Le TTL détermine combien de temps une entrée reste valide dans le cache :

- **Court (1-30 secondes)** : Pour les données qui changent fréquemment
- **Moyen (1-5 minutes)** : Pour les données semi-statiques
- **Long (5-60 minutes)** : Pour les données très statiques

## API

### Méthodes principales

```go
// Récupérer une valeur du cache
value, found := cache.GetBlock(key)

// Stocker une valeur avec TTL par défaut
cache.SetBlock(key, value)

// Stocker une valeur avec TTL personnalisé
cache.SetBlockWithTTL(key, value, customTTL)

// Fonction générique pour gérer le cache
result, err := cache.GetWithCache(key, fetchFunc, ttl...)

// Obtenir les statistiques
stats := cache.Stats()

// Vider le cache
cache.Clear()

// Nettoyer les entrées expirées
cache.Cleanup()
```

### Fonction helper du Backend

```go
// Utilisation simplifiée dans le backend
result, err := b.withCache(cacheKey, fetchFunc, ttl...)
```

### Proxy d'API

```go
// Créer un service RPC avec cache automatique
evmBackend := backend.NewBackend(ctx, logger, clientCtx, allowUnprotectedTxs, indexer)

// Créer le proxy d'API qui intercepte les 5 méthodes configurées
cachedAPI := eth.NewCachedPublicAPI(logger, evmBackend)

// Les 5 méthodes sont automatiquement mises en cache avec 15 secondes de TTL
// - GetAllHyperionTransferTxs
// - GetHyperionAccountTransferTxsByPageAndSize
// - GetValidatorWithHisAssetsAndCommission
// - GetAllWhitelistedAssets
// - GetLastTransactionsInfo

// Obtenir les statistiques du cache
stats := cachedAPI.GetCacheStats()
```

### Interception automatique

```go
// Interception automatique basée sur la configuration
result, err := b.interceptCall("GetBlockByNumber", []interface{}{blockNum, fullTx}, func() (interface{}, error) {
    return b.fetchBlockByNumber(blockNum, fullTx)
})
```

### Approche transparente du Backend

```go
// Vérifier si on a un résultat en cache
if b.hasCachedReturn(cacheKey, "30s") {
    return b.cacheOf(cacheKey), nil
}

// Mettre en cache et retourner le résultat
return b.cacheReturn(cacheKey, result, "30s"), nil

// Ou avec TTL par défaut
return b.cacheReturnWithDefaultTTL(cacheKey, result), nil
```

### Génération de clés

```go
// Clés pour les blocs
key := cache.GetBlockByNumberKey(blockNum, fullTx)
key := cache.GetBlockByHashKey(hash, fullTx)
key := cache.GetBlockNumberKey()

// Clés pour les transactions
key := cache.GetTransactionByHashKey(hash)
key := cache.GetTransactionReceiptKey(hash)

// Clés pour les comptes
key := cache.GetBalanceKey(address, blockNrOrHash)
key := cache.GetCodeKey(address, blockNrOrHash)

// Clés pour les infos de chaîne
key := cache.GetGasPriceKey()
key := cache.ChainIDKey()

// Clé générique
key := cache.GenerateKey("GetAllHyperionTransferTxs", size)
key := cache.GenerateKey("GetHyperionAccountTransferTxsByPageAndSize", address, page, size)
key := cache.GenerateKey("GetValidatorWithHisAssetsAndCommission", address)
key := cache.GenerateKey("GetAllWhitelistedAssets")
key := cache.GenerateKey("GetLastTransactionsInfo", size)
```

## Monitoring

### Statistiques disponibles

```go
stats := cache.Stats()
// Retourne :
// - block_cache_size: nombre d'entrées actuelles
// - default_ttl: TTL par défaut
// - cached_methods: méthodes mises en cache avec leurs TTL
```

### Logs

Le système génère des logs de debug pour :
- Cache hits
- Cache misses
- Mise en cache de nouvelles données
- Erreurs d'initialisation

## Performance

### Avantages

- **Réduction de la latence** : Les requêtes en cache sont servies instantanément
- **Réduction de la charge** : Moins de requêtes vers la base de données
- **Meilleure scalabilité** : Support de plus de requêtes simultanées
- **Simplicité d'utilisation** : Configuration automatique des 5 méthodes
- **Transparence** : Le cache est invisible dans le code
- **Configuration simplifiée** : 5 méthodes hardcodées avec 15 secondes de TTL
- **Interception automatique** : Application automatique du cache selon la configuration
- **Proxy automatique** : Interception transparente des appels RPC

### Considérations

- **Utilisation mémoire** : Surveiller l'utilisation mémoire du cache
- **Cohérence** : Les données en cache peuvent être légèrement obsolètes (15 secondes max)
- **Configuration** : TTL fixé à 15 secondes pour les 5 méthodes

## Exemples d'utilisation

### Utilisation du proxy d'API

```go
// Dans apis.go - configuration automatique
func createEthAPI(ctx *server.Context, clientCtx client.Context, allowUnprotectedTxs bool, indexer types.EVMTxIndexer) []rpc.API {
    evmBackend := backend.NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
    
    return []rpc.API{
        {
            Namespace: "eth",
            Version:   "1.0",
            Service:   eth.NewCachedPublicAPI(ctx.Logger, evmBackend), // Proxy d'API
            Public:    true,
        },
    }
}
```

### Configuration personnalisée

```go
// Cache avec configuration par défaut
rpcCache, err := cache.NewRPCCache(1000, 30*time.Second)
```