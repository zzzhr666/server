#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"

$rules = @(
    @{
        Name = "ProjectEcho-LAN-TCP"
        DisplayName = "Project Echo LAN TCP"
        Protocol = "TCP"
        LocalPort = @("8080", "8081")
    },
    @{
        Name = "ProjectEcho-LAN-UDP"
        DisplayName = "Project Echo LAN UDP"
        Protocol = "UDP"
        LocalPort = @("7001", "7002")
    }
)

foreach ($rule in $rules) {
    $existing = Get-NetFirewallRule -Name $rule.Name -ErrorAction SilentlyContinue
    if ($null -ne $existing) {
        Enable-NetFirewallRule -Name $rule.Name
        continue
    }

    New-NetFirewallRule `
        -Name $rule.Name `
        -DisplayName $rule.DisplayName `
        -Direction Inbound `
        -Action Allow `
        -Protocol $rule.Protocol `
        -LocalPort $rule.LocalPort `
        -RemoteAddress LocalSubnet `
        -Profile Any | Out-Null
}

Write-Host "Project Echo LAN firewall rules are enabled."
