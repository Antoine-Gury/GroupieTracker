# Script PowerShell pour configurer PayPal
# Usage: .\setup-paypal.ps1

Write-Host "Configuration PayPal pour Groupie Tracker" -ForegroundColor Green
Write-Host ""

# Configuration PayPal Sandbox
$env:PAYPAL_CLIENT_ID = "AYZTk4mq-RDQ1wx_cV8_OL8x6Z7DLwdIlVgh9VA1-hxIpVl90W0CsIx0LOPnPJhbZUUXtMYGl3005mPi"
$env:PAYPAL_SECRET = "EN_zEbAcKwJluLRQOUJEZbqUmVgRFYxtuy3gD5WoTuLozW8ptEQyp_6uqd3-_6NGQUQxI3h7-88jc-gq"
$env:PAYPAL_MODE = "sandbox"

Write-Host "✅ Variables d'environnement PayPal configurées :" -ForegroundColor Green
Write-Host "   PAYPAL_CLIENT_ID = $env:PAYPAL_CLIENT_ID"
Write-Host "   PAYPAL_SECRET = $env:PAYPAL_SECRET (masqué)"
Write-Host "   PAYPAL_MODE = $env:PAYPAL_MODE"
Write-Host ""
Write-Host "📧 Compte Sandbox de test :" -ForegroundColor Yellow
Write-Host "   Email: sb-4a7lm47621519@business.example.com"
Write-Host "   Password: M^`$922tT"
Write-Host ""
Write-Host "⚠️  Ces variables sont définies pour cette session PowerShell uniquement." -ForegroundColor Yellow
Write-Host "   Pour les rendre permanentes, ajoutez-les dans les variables système Windows." -ForegroundColor Yellow
Write-Host ""
Write-Host "🚀 Vous pouvez maintenant lancer le serveur avec :" -ForegroundColor Green
Write-Host "   go run main.go"
Write-Host ""

