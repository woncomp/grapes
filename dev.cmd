@echo off
setlocal EnableExtensions

set "SCRIPT_DIR=%~dp0"
set "ORIGINAL_ARGS=%*"
set "MODE="
set /a EXTRA_ARG_COUNT=0

pushd "%SCRIPT_DIR%" >nul

:parse_args
if "%~1"=="" goto after_parse

if /I "%~1"=="-b" (
	call :set_mode build
	if errorlevel 1 goto fail
) else if /I "%~1"=="--build" (
	call :set_mode build
	if errorlevel 1 goto fail
) else if /I "%~1"=="-r" (
	call :set_mode release
	if errorlevel 1 goto fail
) else if /I "%~1"=="--release" (
	call :set_mode release
	if errorlevel 1 goto fail
) else (
	set /a EXTRA_ARG_COUNT+=1
)

shift
goto parse_args

:after_parse
if /I "%MODE%"=="build" (
	if not "%EXTRA_ARG_COUNT%"=="0" (
		echo error: build mode does not accept extra arguments 1>&2
		goto fail
	)
	if not exist "bin" mkdir "bin"
	go build -o ".\bin\grapes.exe" .\cmd\grapes
	goto finish
)

if /I "%MODE%"=="release" (
	if not "%EXTRA_ARG_COUNT%"=="0" (
		echo error: release mode does not accept extra arguments 1>&2
		goto fail
	)
	goreleaser release --snapshot --clean
	goto finish
)

go run .\cmd\grapes %ORIGINAL_ARGS%
goto finish

:set_mode
if not defined MODE (
	set "MODE=%~1"
	exit /b 0
)

if /I "%MODE%"=="%~1" exit /b 0

echo error: cannot combine build and release modes 1>&2
exit /b 1

:fail
set "EXIT_CODE=%ERRORLEVEL%"
goto cleanup

:finish
set "EXIT_CODE=%ERRORLEVEL%"

:cleanup
popd >nul
exit /b %EXIT_CODE%
