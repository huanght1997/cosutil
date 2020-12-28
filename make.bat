@ECHO OFF

SETLOCAL
SET PATH=%GOPATH%\bin;%PATH%
SET CGO_ENABLED=0
SET GO111MODULE=on
SET LDFLAGS="-s -w"

IF /I "%1"=="all" GOTO all
IF /I "%1"=="build" GOTO build
IF /I "%1"=="app" GOTO app
IF /I "%1"=="clean" GOTO clean
IF /I "%1"=="" GOTO all
GOTO error

:all
	CALL make.bat build
	GOTO :EOF

:build
	CALL make.bat app
	GOTO :EOF

:app
	SET os=darwin freebsd linux windows
	SET arch=386 amd64
	FOR %%i in (%os%) do (
		SET GOOS=%%i
		FOR %%j in (%arch%) do (
			SET GOARCH=%%j
			IF "%%i"=="windows" (
				go build -ldflags %LDFLAGS% -o release\cosutil_%%i_%%j.exe
			) ELSE (
				go build -ldflags %LDFLAGS% -o release\cosutil_%%i_%%j
			)
		)
	)
	SET GOOS=linux
	SET GOARCH=arm
	go build -ldflags %LDFLAGS% -o release\cosutil_linux_arm
	SET GOARCH=arm64
	go build -ldflags %LDFLAGS% -o release\cosutil_linux_arm64
	GOTO :EOF

:clean
	DEL /Q release
	RD release
	GOTO :EOF

:error
    IF "%1"=="" (
        ECHO make: *** No targets specified and no makefile found.  Stop.
    ) ELSE (
        ECHO make: *** No rule to make target '%1%'. Stop.
    )
    GOTO :EOF
