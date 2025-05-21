@echo off
echo Building FolderOpener executable...

REM Create build directory if it doesn't exist
if not exist "build" mkdir build

REM Build for Windows as a GUI application (no console window)
go build -ldflags="-H windowsgui" -o build/FolderOpener.exe main.go

echo Done! Executable is in the build folder.
echo The application will run silently in the background.
pause
