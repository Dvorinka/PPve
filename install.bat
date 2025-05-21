@echo off
echo Installing FolderOpener...

REM Create destination directory
set INSTALL_DIR=%APPDATA%\FolderOpener
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

REM Copy executable
copy /Y "build\FolderOpener.exe" "%INSTALL_DIR%"

REM Create startup shortcut
set STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup
echo Set oWS = WScript.CreateObject("WScript.Shell") > "%TEMP%\CreateShortcut.vbs"
echo sLinkFile = "%STARTUP_DIR%\FolderOpener.lnk" >> "%TEMP%\CreateShortcut.vbs"
echo Set oLink = oWS.CreateShortcut(sLinkFile) >> "%TEMP%\CreateShortcut.vbs"
echo oLink.TargetPath = "%INSTALL_DIR%\FolderOpener.exe" >> "%TEMP%\CreateShortcut.vbs"
echo oLink.WorkingDirectory = "%INSTALL_DIR%" >> "%TEMP%\CreateShortcut.vbs"
echo oLink.Description = "FolderOpener - Windows Folder Opener Service" >> "%TEMP%\CreateShortcut.vbs"
echo oLink.Save >> "%TEMP%\CreateShortcut.vbs"
cscript /nologo "%TEMP%\CreateShortcut.vbs"
del "%TEMP%\CreateShortcut.vbs"

echo Installation complete! FolderOpener will start automatically when you log in.
echo Starting FolderOpener now...

REM Start the application
start "" "%INSTALL_DIR%\FolderOpener.exe"

pause
