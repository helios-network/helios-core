# Examples of Store Cron Callback Data

Primitive types:

```go
k.keeper.StoreCronCallBackData(ctx, newCron.Id, &types.CronCallBackData{
    Data:  []byte("example string"), // Exemple de chaîne de caractères
    Error: []byte("An error occurred"), // Exemple de message d'erreur
})

k.keeper.StoreCronCallBackData(ctx, newCron.Id, &types.CronCallBackData{
    Data:  []byte{1, 2, 3, 4, 5}, // Exemple de tableau d'entiers
    Error: []byte("Failed to execute transaction due to insufficient funds"), // Exemple de message d'erreur détaillé
})

k.keeper.StoreCronCallBackData(ctx, newCron.Id, &types.CronCallBackData{
    Data:  common.BigToHash(big.NewInt(1234567890)).Bytes(), // Exemple de Uint256
    Error: []byte{}, // Pas d'erreur
})

k.keeper.StoreCronCallBackData(ctx, newCron.Id, &types.CronCallBackData{
    Data:  common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678").Bytes(), // Exemple d'adresse Ethereum
    Error: []byte("Invalid address format"), // Exemple d'erreur
})
```

Array example:

```go
import (
    "math/big"
    "github.com/ethereum/go-ethereum/common"
)

// Exemple de tableau de uint256
uint256Array := []*big.Int{
    big.NewInt(1000),
    big.NewInt(2000),
    big.NewInt(3000),
}

// Convertir chaque élément en bytes
var dataBytes []byte
for _, num := range uint256Array {
    dataBytes = append(dataBytes, common.BigToHash(num).Bytes()...)
}

// Utiliser dataBytes dans votre fonction
k.keeper.StoreCronCallBackData(ctx, newCron.Id, &types.CronCallBackData{
    Data:  dataBytes,
    Error: []byte("No error"),
})
```

Exemple of retrieving in solidity

```solidity
contract Counter {

    uint256 public count = 0;

    // Définition de l'événement
    event CountIncremented(uint256 newCount);

    constructor() {
        require(msg.sender != address(0), "ABORT sender - address(0)");
    }

    function callBack(bytes memory data, bytes memory err) public {
        require(err.length == 0, "Error is not empty");
        uint256 ethPrice = abi.decode(data, (uint256));
        count += ethPrice;
    }

    function increment() public {
        count += 1;
        emit CountIncremented(count);
    }
}
```
