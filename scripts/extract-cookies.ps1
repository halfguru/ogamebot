param(
    [Parameter(Mandatory=$true)]
    [string]$BearerToken,
    [Parameter(Mandatory=$true)]
    [string]$PhpSessId
)

$cookiesJson = @{
    cookies = @(
        @{
            Name       = "gf-token-production"
            Value      = $BearerToken
            Domain     = ".gameforge.com"
            Path       = "/"
            Expires    = (Get-Date).AddDays(7).ToString("yyyy-MM-ddTHH:mm:ssZ")
            HttpOnly   = $false
            Secure     = $true
            Session    = $false
        },
        @{
            Name       = "PHPSESSID"
            Value      = $PhpSessId
            Domain     = "lobby.ogame.gameforge.com"
            Path       = "/"
            Expires    = (Get-Date).AddDays(7).ToString("yyyy-MM-ddTHH:mm:ssZ")
            HttpOnly   = $true
            Secure     = $true
            Session    = $false
        }
    )
} | ConvertTo-Json -Depth 5

$outputPath = Join-Path $PSScriptRoot "..\cookies.json"
$cookiesJson | Out-File -FilePath $outputPath -Encoding utf8 -Force
Write-Host "Cookies written to $outputPath"
