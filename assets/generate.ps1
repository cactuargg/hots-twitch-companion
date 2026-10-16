$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go run assets/generate.go
    if ($LASTEXITCODE -ne 0) { throw 'Icon generation failed' }
    # Wails v2 expects RT_GROUP_ICON #3. Commit the generated resource so normal
    # builds do not need a separate resource compiler or network download.
    go run github.com/tc-hib/go-winres@v0.3.3 make --in assets/winres.json --arch amd64 --out rsrc
    if ($LASTEXITCODE -ne 0) { throw 'Resource generation failed' }
} finally { Pop-Location }
