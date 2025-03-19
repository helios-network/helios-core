# On Mac OS: brew install gnu-sed
# On Linux: change command to sed


# cp helios-chain/x/chronos/types/*.go ../sdk-go/chain/chronos/types/
# # Parcourir tous les fichiers du répertoire
# for file in $(find "../sdk-go/chain/chronos/types/" -type f); do
#   echo "Processing file: $file"
  
#   # Remplacer les chaînes dans chaque fichier
#   awk '{
#     gsub("helios-core/helios-chain/modules", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     print;
#   }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
# done

# cp helios-chain/x/hyperion/types/*.go ../sdk-go/chain/hyperion/types/
# # Parcourir tous les fichiers du répertoire
# for file in $(find "../sdk-go/chain/hyperion/types/" -type f); do
#   echo "Processing file: $file"
  
#   # Remplacer les chaînes dans chaque fichier
#   awk '{
#     gsub("helios-core/helios-chain/modules", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     print;
#   }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
# done

# cp helios-chain/x/tokenfactory/types/*.go ../sdk-go/chain/tokenfactory/types/
# # Parcourir tous les fichiers du répertoire
# for file in $(find "../sdk-go/chain/tokenfactory/types/" -type f); do
#   echo "Processing file: $file"
  
#   # Remplacer les chaînes dans chaque fichier
#   awk '{
#     gsub("helios-core/helios-chain/modules", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
#     print;
#   }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
# done

# Fonction pour traiter les fichiers d'un type donné
process_files() {
  local type=$1
  local source_dir="helios-chain/x/$type/"
  local dest_dir="../sdk-go/chain/$type"

  # Vérifier si le répertoire source existe
  if [ -d "$source_dir" ]; then
    cp "$source_dir"*.go "$dest_dir"
    
    # Parcourir tous les fichiers du répertoire
    for file in $(find "$dest_dir" -type f); do
      echo "Processing file: $file"
      
      # Remplacer les chaînes dans chaque fichier
      awk '{
        gsub("helios-core/helios-chain/x", "github.com/Helios-Chain-Labs/sdk-go/chain");
        gsub("helios-core/helios-chain", "github.com/Helios-Chain-Labs/sdk-go/chain");
        print;
      }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    done
  else
    echo "Warning: Source directory $source_dir does not exist. Skipping $type."
  fi
}

cp -r proto/ ../sdk-go/proto

current_dir=$(pwd)
types=("chronos/types" "epochs/types" "erc20/types" "evm/types" "evm/statedb" "evm/core/vm" "hyperion/types" "ibc/transfer/types" "inflation/types" "tokenfactory/types" "vesting/types" "feemarket/types")

for type in "${types[@]}"; do

  process_files "$type"

  cd "../sdk-go/chain/$type/" || exit
  rm -f *test.go
  rm -f *gw.go
  gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/x/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go
  cd "$current_dir" || exit
done