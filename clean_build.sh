cd ./web_res
rm -rf dist/*
echo "Compiling TypeScript Code.."
tsc
cd ..
echo "Compiling Golang Code.."
go build -a