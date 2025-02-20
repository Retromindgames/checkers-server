@echo off
setlocal enableextensions enabledelayedexpansion

:: Start Memurai
start "" "C:\Program Files\Memurai\memurai.exe"

:: Start Go programs and store their PIDs
start "" cmd /c "go run C:\Projetos\golang\go-websocket-prototype\main.go" 
start "" cmd /c "go run C:\Projetos\golang\go-websocket-prototype\pstatusworker\main.go"
start "" cmd /c "go run C:\Projetos\golang\go-websocket-prototype\roomworker\main.go"
start "" cmd /c "go run C:\Projetos\golang\go-websocket-prototype\broadcastworker\main.go" 
start "" cmd /c "go run C:\Projetos\golang\go-websocket-prototype\gameworker\main.go" 

echo All services started.
pause

