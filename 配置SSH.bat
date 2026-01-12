@echo off
echo ========================================
echo SSH Key Configuration
echo Server: 115.190.239.146
echo Password: GGsuperman1211
echo ========================================
echo.
echo Please enter password when prompted...
echo.

type "%USERPROFILE%\.ssh\id_rsa.pub" | ssh -o StrictHostKeyChecking=no root@115.190.239.146 "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo SSH key configured!"

echo.
echo Testing connection...
ssh root@115.190.239.146 "echo 'SSH test successful!'"

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo SUCCESS! SSH key is configured!
    echo ========================================
    echo.
    echo Now run: bash deploy-now.sh
    echo.
) else (
    echo.
    echo Configuration may have failed.
    echo Please try running in Git Bash instead.
)

pause
