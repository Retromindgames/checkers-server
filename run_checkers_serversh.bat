@echo off
:: Start Memurai
start "" "C:\Program Files\Memurai\memurai.exe"

:: Start Go programs in separate windows using 'go run'
start "" cmd /k "go run C:\Projetos\golang\go-websocket-prototype\main.go"
start "" cmd /k "go run C:\Projetos\golang\go-websocket-prototype\pstatusworker\main.go"
start "" cmd /k "go run C:\Projetos\golang\go-websocket-prototype\roomworker\main.go"
start "" cmd /k "go run C:\Projetos\golang\go-websocket-prototype\broadcastworker\main.go"

echo All services started.
pause
