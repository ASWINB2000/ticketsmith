# scripts/windows/create-self-signed-cert.ps1
# One-time: generates a free self-signed Authenticode code-signing certificate
# for "Aswin Biju", valid 5 years. Run once, then follow the printed
# instructions to store it as GitHub secrets.

param(
    [Parameter(Mandatory=$true)]
    [string]$Password
)

$cert = New-SelfSignedCertificate `
    -Type CodeSigningCert `
    -Subject "CN=Aswin Biju" `
    -CertStoreLocation "Cert:\CurrentUser\My" `
    -NotAfter (Get-Date).AddYears(5) `
    -KeyUsage DigitalSignature `
    -KeyAlgorithm RSA `
    -KeyLength 2048

$securePassword = ConvertTo-SecureString -String $Password -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath ".\ticketsmith-codesign.pfx" -Password $securePassword | Out-Null
Export-Certificate -Cert $cert -FilePath ".\ticketsmith-codesign.cer" | Out-Null

$base64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes(".\ticketsmith-codesign.pfx"))
$base64 | Set-Content ".\ticketsmith-codesign.pfx.base64.txt"

Write-Host ""
Write-Host "Done. Next steps:"
Write-Host "1. Add repo secret WINDOWS_CERT_PFX_BASE64 = contents of ticketsmith-codesign.pfx.base64.txt"
Write-Host "2. Add repo secret WINDOWS_CERT_PASSWORD = the password you passed to this script"
Write-Host "3. Delete ticketsmith-codesign.pfx, .cer, and .base64.txt locally once secrets are saved (don't commit them)"
Write-Host "4. Optional, to suppress SmartScreen on machines you control:"
Write-Host "   Import-Certificate -FilePath .\ticketsmith-codesign.cer -CertStoreLocation Cert:\LocalMachine\TrustedPublisher"
