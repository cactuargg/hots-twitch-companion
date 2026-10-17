# One version source for local packages and GitHub Actions.
$versionMatch = [regex]::Match((Get-Content "$PSScriptRoot/cmd/companion/version.go" -Raw), 'const appVersion = "([0-9]+\.[0-9]+\.[0-9]+)"')
if (!$versionMatch.Success) { throw 'Cannot read appVersion from version.go' }
$buildVersion = 'v' + $versionMatch.Groups[1].Value
