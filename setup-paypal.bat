@echo off
REM Script batch pour configurer PayPal sur Windows
REM Usage: setup-paypal.bat

echo Configuration PayPal pour Groupie Tracker
echo.

REM Configuration PayPal Sandbox
set PAYPAL_CLIENT_ID=AYZTk4mq-RDQ1wx_cV8_OL8x6Z7DLwdIlVgh9VA1-hxIpVl90W0CsIx0LOPnPJhbZUUXtMYGl3005mPi
set PAYPAL_SECRET=EN_zEbAcKwJluLRQOUJEZbqUmVgRFYxtuy3gD5WoTuLozW8ptEQyp_6uqd3-_6NGQUQxI3h7-88jc-gq
set PAYPAL_MODE=sandbox

echo Variables d'environnement PayPal configurees:
echo   PAYPAL_CLIENT_ID = %PAYPAL_CLIENT_ID%
echo   PAYPAL_SECRET = %PAYPAL_SECRET% (masque)
echo   PAYPAL_MODE = %PAYPAL_MODE%
echo.
echo Compte Sandbox de test:
echo   Email: sb-4a7lm47621519@business.example.com
echo   Password: M^$922tT
echo.
echo ATTENTION: Ces variables sont definies pour cette session CMD uniquement.
echo   Pour les rendre permanentes, utilisez les variables systeme Windows.
echo.
echo Vous pouvez maintenant lancer le serveur avec:
echo   go run main.go
echo.
pause

