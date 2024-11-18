# On Mac OS: brew install gnu-sed
# On Linux: change command to sed

cp helios-chain/modules/auction/types/*.go ../sdk-go/chain/auction/types/
cp helios-chain/modules/exchange/types/*.go ../sdk-go/chain/exchange/types/
cp helios-chain/modules/ocr/types/*.go ../sdk-go/chain/ocr/types/
cp helios-chain/modules/peggy/types/*.go ../sdk-go/chain/peggy/types/
cp helios-chain/modules/wasmx/types/*.go ../sdk-go/chain/wasmx/types/
cp helios-chain/modules/insurance/types/*.go ../sdk-go/chain/insurance/types/
cp helios-chain/modules/oracle/types/*.go ../sdk-go/chain/oracle/types/
cp helios-chain/modules/tokenfactory/types/*.go ../sdk-go/chain/tokenfactory/types/
cp -r proto/ ../sdk-go/proto


cd ../sdk-go/chain/auction/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-/github.com\/Helios-Chain-Labs\/sdk-go\//g" *.go

cd ../../exchange/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../ocr/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../peggy/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../wasmx/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../insurance/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../oracle/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go

cd ../../tokenfactory/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/Helios-Chain-Labs\/helios-core\/helios-chain\/modules/github.com\/Helios-Chain-Labs\/sdk-go\/chain/g" *.go
