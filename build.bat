cd app/ui/app
npm install --silent
npm run build
cd ../../..
go build -ldflags="-s -w -H=windowsgui" -o "EOC.exe" ./app/cmd/app/

