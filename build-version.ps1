# One version source for local packages and GitHub Actions.
$versionMatch = [regex]::Match((Get-Content "$PSScriptRoot/cmd/companion/core.go" -Raw), 'const appVersion = "([0-9]+\.[0-9]+\.[0-9]+)"')
if (!$versionMatch.Success) { throw 'Cannot read appVersion from core.go' }
$buildVersion = 'v' + $versionMatch.Groups[1].Value
