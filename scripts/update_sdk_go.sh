# On Mac OS: brew install gnu-sed
# On Linux: change command to sed

cp helios-chain/x/hyperion/types/*.go ../sdk-go/chain/hyperion/types/
# Parcourir tous les fichiers du répertoire
for file in $(find "../sdk-go/chain/hyperion/types/" -type f); do
  echo "Processing file: $file"
  
  # Remplacer les chaînes dans chaque fichier
  awk '{
    gsub("helios-core/helios-chain/modules", "github.com/Helios-Chain-Labs/sdk-go/chain");
    gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
    print;
  }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

cp helios-chain/modules/tokenfactory/types/*.go ../sdk-go/chain/tokenfactory/types/
# Parcourir tous les fichiers du répertoire
for file in $(find "../sdk-go/chain/tokenfactory/types/" -type f); do
  echo "Processing file: $file"
  
  # Remplacer les chaînes dans chaque fichier
  awk '{
    gsub("helios-core/helios-chain/modules", "github.com/Helios-Chain-Labs/sdk-go/chain");
    gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
    print;
  }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

cp -r proto/ ../sdk-go/proto

cd ../../hyperion/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../tokenfactory/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go
