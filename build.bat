cd app/ui/app
call npm install --silent
call npm run build
cd ../../..
go build -ldflags="-s -w -H=windowsgui" -o "EOC.exe" ./app/cmd/app/

