cd ./web_res
echo "Compiling TypeScript Code.."
tsc --pretty 
cd ..
echo "Compiling Golang Code.."
go build -a