# Windows Installer - Анализ и Исправления

## Оригинальная Проблема
При установке Ali/Ollama на Windows выяснялись следующие ошибки:
```
winget install Inno Setup failed on attempt 1/3: winget install failed for winget install Inno Setup (JRSoftware.InnoSetup)
...
Add-Content : Џа®жҐбб ­Ґ ¬®¦Ґв Ї®«гзЁвм ¤®бвгЇ Є д ©«г
```

**Корень проблемы:** Множественные ошибки в обработке установки зависимостей и логирования.

---

## Исправленные Проблемы

### 1. **Обнаружение Inno Setup (КРИТИЧНО)**
**Проблема:** Функция `Find-InnoSetupCompiler()` использовала `Get-ChildItem "C:\Program Files*\..."` с глобальным паттерном, который работает неправильно.

**Решение:** Переписана функция с явным указанием всех стандартных путей установки:
```powershell
function Find-InnoSetupCompiler {
    $candidates = @(
        "C:\Program Files\Inno Setup 6",
        "C:\Program Files (x86)\Inno Setup 6",
        "C:\Program Files\Inno Setup",
        "C:\Program Files (x86)\Inno Setup"
    )
    
    # + поиск в реестре
    # + проверка команды в PATH
}
```

**Файл:** `scripts/full_install_windows.ps1`

---

### 2. **Обработка "Already Installed" Ошибок (КРИТИЧНО)**
**Проблема:** Winget возвращает ошибку, когда пакет уже установлен, но скрипт трактует это как критическую ошибку.

**Решение:** Расширена функция `Install-WithWinget()` для обработки этого случая:
```powershell
# Старый код:
if ($LASTEXITCODE -ne 0) {
    throw "winget install failed for $Name ($Id)"
}

# Новый код:
if ($LASTEXITCODE -eq 0) {
    Write-Info "$Name installation completed."
} elseif ($LASTEXITCODE -eq -1978335694 -or $output -match "Found an existing package|already installed") {
    Write-Info "$Name is already installed. Skipping."
} else {
    throw "winget install failed for $Name ($Id) with exit code $LASTEXITCODE"
}
```

**Файл:** `scripts/full_install_windows.ps1`

---

### 3. **Улучшенная Проверка Установленных Инструментов**
**Проблема:** Функция `Ensure-BuildDependencies()` не информировала пользователя, что инструменты уже установлены, и всегда пыталась переустановить.

**Решение:** Добавлены информационные сообщения и повторная проверка после установки:
```powershell
if (-not (Test-CommandAvailable git)) {
    Install-WithWinget ...
} else {
    Write-Info "Git is already installed."
}

# После установки Inno Setup:
$innoSetup = Find-InnoSetupCompiler
if (-not $innoSetup) {
    throw "Inno Setup installation failed..."
}
```

**Файл:** `scripts/full_install_windows.ps1`

---

### 4. **Проблемы с Логированием Файла (КРИТИЧНО)**
**Проблема:** Ошибка кодировки в сообщении об ошибке указывала на конфликт блокировки файла:
```
Add-Content : Џа®жҐбб ­Ґ ¬®¦Ґв Ї®«гзЁвм ¤®бвгЇ Є д ©«г "install.log"
```

Это происходило потому, что `Start-Transcript` блокировал файл, а затем `Add-Content` пытался добавить данные.

**Решение:** Создана безопасная функция логирования:
```powershell
function SafeAddContent {
    param([string]$Message)
    try {
        Add-Content -LiteralPath $LogFile -Value $Message -ErrorAction SilentlyContinue
    } catch {
        # Fallback: используем System.IO.File API
        try {
            [System.IO.File]::AppendAllText($LogFile, "$Message`n", [System.Text.Encoding]::UTF8)
        } catch {
            # Last resort: резервный лог
            $backupLog = "$LogFile.backup"
            try {
                Add-Content -LiteralPath $backupLog -Value $Message -ErrorAction SilentlyContinue
            } catch {
                # Молчаливо падаем - не хотим, чтобы логирование остановило установку
            }
        }
    }
}
```

**Файл:** `scripts/full_install_windows.ps1`

---

### 5. **Обработка Ошибок в Bootstrap (Исправлено)**
**Проблема:** В `build_web_installer_exe.ps1` блок `catch` пытался логировать ошибку, но это могло вызвать еще одну ошибку блокировки:
```powershell
} catch {
    Add-Content -LiteralPath $LogFile -Value "ERROR: ..."  # Может упасть!
    throw
}
```

**Решение:** Добавлена обработка ошибок логирования:
```powershell
} catch {
    try {
        Add-Content -LiteralPath $LogFile -Value "ERROR: $($_.Exception.Message)" -ErrorAction SilentlyContinue
    } catch {
        # Loggging failed - write to console as backup
        Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    }
    throw
}
```

**Файл:** `scripts/build_web_installer_exe.ps1`

---

## Новые Утилиты

### `verify_windows_setup.ps1`
Новый скрипт для проверки окружения перед установкой. Помогает диагностировать проблемы:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify_windows_setup.ps1
```

**Проверяет:**
- ✓ Git, Go, Node.js, npm, CMake
- ✓ C# Compiler (csc.exe)
- ✓ Inno Setup
- ✓ Visual Studio Build Tools
- ✓ Исходный код проекта (app/ollama.iss, go.mod)

---

## Как Тестировать Исправления

### 1. Проверить Окружение
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify_windows_setup.ps1
```

### 2. Полная Установка (Новая версия с исправлениями)
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly
```

### 3. Web Installer
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1
```

---

## Технические Детали

| Проблема | Решение | Тип | Файл |
|----------|---------|-----|------|
| Неправильное обнаружение Inno Setup | Явные пути + реестр + PATH | Критическое | `full_install_windows.ps1` |
| Winget не обрабатывает "Already Installed" | Проверка exit codes и output | Критическое | `full_install_windows.ps1` |
| Блокировка файла лога | SafeAddContent с fallbacks | Критическое | `full_install_windows.ps1` |
| Нет информации о статусе инструментов | Добавлены информационные сообщения | Улучшение | `full_install_windows.ps1` |
| Ошибка в обработке исключений | -ErrorAction SilentlyContinue | Улучшение | `build_web_installer_exe.ps1` |
| Нет диагностики окружения | Новый verify скрипт | Новое | `verify_windows_setup.ps1` |

---

## Рекомендации по Использованию

### Первый Запуск на Новой Машине
```powershell
# 1. Проверить окружение
.\scripts\verify_windows_setup.ps1

# 2. Установить Ali/Ollama
.\scripts\full_install_windows.ps1 -CpuOnly
```

### Переустановка
```powershell
# Скрипт умно обработает уже установленные инструменты
.\scripts\full_install_windows.ps1
```

### Создание Web Setup
```powershell
# Создаст AliWebSetup.exe с встроенным исходным кодом
.\scripts\build_web_installer_exe.ps1
```

---

## Известные Ограничения

1. **Требуется Windows 10+** - для корректной работы PowerShell и winget
2. **Требуется App Installer** - winget работает только если установлен Microsoft Store App Installer
3. **Требуется Admin привилегии** - для установки некоторых компонентов

---

## Отладка

### Проверить Логи Установки
```powershell
Get-Content "$env:TEMP\ali-full-install.log"
Get-Content "$env:TEMP\ali-full-install.log.backup"
```

### Проверить Файлы Bootstrap
```powershell
Get-ChildItem "$env:TEMP\AliWebSetup-*" -Recurse
```

### Запустить Verify Skript для Диагностики
```powershell
.\scripts\verify_windows_setup.ps1
```

---

## Версия Документа
- **Дата:** May 1, 2026
- **Статус:** Ready for Testing
- **Исправлено:** 5 критических проблем, 1 улучшение

