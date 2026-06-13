@echo off
setlocal enabledelayedexpansion

if "%~1"=="" goto help
goto %~1 2>nul || (
    echo Unknown target: %~1
    goto help
)

:setup
call :deps || exit /b 1
call :web  || exit /b 1
call :generate || exit /b 1
echo ==^> Setup complete
goto :eof

:deps
echo ==^> Downloading Go modules
go mod download || exit /b 1
go mod tidy     || exit /b 1
goto :eof

:web
call :web-deps || exit /b 1
call :spa      || exit /b 1
goto :eof

:web-deps
echo ==^> Installing web dependencies
pushd web
pnpm install --no-frozen-lockfile || (popd & exit /b 1)
popd
goto :eof

:spa
echo ==^> Building admin SPA
pushd web
pnpm build || (popd & exit /b 1)
popd
goto :eof

:generate
echo ==^> Generating module imports
go generate ./internal/bootstrap/mod || exit /b 1
goto :eof

:build
call :setup || exit /b 1
echo ==^> Building binary
if not exist dist mkdir dist
for /f "tokens=*" %%c in ('git rev-parse --short=12 HEAD 2^>nul') do set "COMMIT=%%c"
if not defined COMMIT set "COMMIT=none"
if not defined VERSION set "VERSION=dev"
set "CGO_ENABLED=0"
go build -trimpath -ldflags "-s -w -X main.Version=%VERSION% -X main.Commit=%COMMIT% -X main.Build=local" -o dist\backend-go.exe ./cmd/server || exit /b 1
echo ==^> Built dist\backend-go.exe
goto :eof

:dev
call :deps     || exit /b 1
call :generate || exit /b 1
echo ==^> Building binary (dev)
if not exist dist mkdir dist
for /f "tokens=*" %%c in ('git rev-parse --short=12 HEAD 2^>nul') do set "COMMIT=%%c"
if not defined COMMIT set "COMMIT=none"
if not defined VERSION set "VERSION=dev"
go build -ldflags "-s -w -X main.Version=%VERSION% -X main.Commit=%COMMIT% -X main.Build=local" -o dist\backend-go.exe ./cmd/server || exit /b 1
echo ==^> Built dist\backend-go.exe
goto :eof

:run
call :dev || exit /b 1
echo ==^> Running
dist\backend-go.exe
goto :eof

:clean
echo ==^> Cleaning
if exist dist rd /s /q dist
if exist internal\bootstrap\mod\autogen_imports.go del /f internal\bootstrap\mod\autogen_imports.go
goto :eof

:help
echo.
echo Usage: make ^<target^>
echo.
echo   setup       Full init: deps + web SPA + codegen
echo   deps        Download and tidy Go modules
echo   web         Install web deps and build admin SPA
echo   web-deps    Install web (pnpm) dependencies only
echo   spa         Build admin SPA only (assumes deps installed)
echo   generate    Generate module import file (autogen_imports.go)
echo   build       Full build: setup + compile binary
echo   dev         Quick dev build (skip web)
echo   run         Build and run
echo   clean       Remove build artifacts
echo   help        Show this help
echo.
goto :eof
