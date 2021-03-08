cd ./web_res
echo "Compiling TypeScript Code.."
tsc
cd ..
echo "Compiling Golang Code.."
go build